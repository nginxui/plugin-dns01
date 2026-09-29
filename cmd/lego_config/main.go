// Command lego_config refreshes catalog/data from a lego release.
//
//	go run ./cmd/lego_config          # latest release
//	go run ./cmd/lego_config v5.5.2   # a specific tag
//
// It downloads the lego source archive, copies every providers/*/*.toml file
// into catalog/data as a plain file and drops the ones the release no longer
// ships. Run ./cmd/manifest afterwards to regenerate plugin.json.
//
// Ported from cmd/lego_config of nginx-ui, which stores the same files in a
// compressed archive instead.
package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	githubAPIURL  = "https://cloud.nginxui.com/https://api.github.com/repos/go-acme/lego/releases/latest"
	downloadURL   = "https://cloud.nginxui.com/https://github.com/go-acme/lego/archive/refs/tags/%s.zip"
	userAgent     = "NGINX-UI-DNS01-LegoConfig"
	catalogDir    = "catalog/data"
	requestTimout = 5 * time.Minute
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "lego_config:", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := repoRoot()
	if err != nil {
		return err
	}

	tag := ""
	if len(os.Args) > 1 {
		tag = os.Args[1]
	}
	if tag == "" {
		tag, err = latestReleaseTag()
		if err != nil {
			return fmt.Errorf("could not resolve the latest release: %w", err)
		}
	}
	fmt.Println("lego release:", tag)

	archive, err := download(fmt.Sprintf(downloadURL, tag))
	if err != nil {
		return fmt.Errorf("could not download the release: %w", err)
	}
	defer os.Remove(archive)

	written, err := extractTOML(archive, tag, filepath.Join(root, catalogDir))
	if err != nil {
		return err
	}

	removed, err := pruneStale(filepath.Join(root, catalogDir), written)
	if err != nil {
		return err
	}

	fmt.Printf("wrote %d provider files, removed %d stale files\n", len(written), removed)
	fmt.Println("now run: go run ./cmd/manifest")
	return nil
}

// repoRoot resolves the module root from this file's location.
func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to locate the generator source")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
}

func latestReleaseTag() (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubAPIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: requestTimout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status from the GitHub API: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("the latest release carries no tag name")
	}
	return release.TagName, nil
}

// download fetches url into a temporary file and returns its path.
func download(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: requestTimout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.CreateTemp("", "lego-*.zip")
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}
	return out.Name(), nil
}

// extractTOML copies every providers/*/*.toml entry into dest and returns the
// base names it wrote.
func extractTOML(archive, tag, dest string) (map[string]bool, error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return nil, fmt.Errorf("could not open the archive: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}

	prefix := "lego-" + strings.TrimPrefix(tag, "v") + "/providers/"
	written := map[string]bool{}

	for _, f := range reader.File {
		if f.FileInfo().IsDir() || !strings.HasSuffix(f.Name, ".toml") {
			continue
		}
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}

		name := path.Base(f.Name)

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("could not read %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("could not read %s: %w", f.Name, err)
		}

		if err := os.WriteFile(filepath.Join(dest, name), data, 0o644); err != nil {
			return nil, err
		}
		written[name] = true
	}

	if len(written) == 0 {
		return nil, fmt.Errorf("no provider file found under %s", prefix)
	}
	return written, nil
}

// pruneStale removes the TOML files the release no longer ships.
func pruneStale(dir string, keep map[string]bool) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	var removed int
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".toml") || keep[name] {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
