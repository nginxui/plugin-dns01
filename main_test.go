package main

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	sdk "github.com/nginxui/plugin-sdk-go"
	"github.com/nginxui/plugin-sdk-go/jsonrpc"
	"github.com/nginxui/plugin-sdk-go/protocol"

	"github.com/nginxui/plugin-dns01/provider"
)

// TestEndToEnd drives the real plugin wiring over in-memory pipes: handshake,
// a dns01.options call for a provider that needs no network, then shutdown.
func TestEndToEnd(t *testing.T) {
	handler := provider.New()

	pluginIn, hostW := io.Pipe()
	hostR, pluginOut := io.Pipe()

	host := jsonrpc.NewConn(hostR, hostW)
	host.Handle(protocol.MethodHostLog, func(context.Context, json.RawMessage) (any, error) {
		return protocol.EmptyResult{}, nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	hostDone := make(chan struct{})
	go func() { defer close(hostDone); _ = host.Serve(ctx) }()

	runErr := make(chan error, 1)
	go func() {
		runErr <- sdk.Run(ctx, sdk.Plugin{
			DNS01: handler,
			Configure: func(_ context.Context, settings map[string]any) error {
				handler.SetSettings(provider.ParseSettings(settings))
				return nil
			},
		}, pluginIn, pluginOut)
	}()

	t.Cleanup(func() {
		cancel()
		host.Close()
		_ = hostW.Close()
		_ = hostR.Close()
		<-hostDone
	})

	// plugin.initialize
	var init protocol.InitializeResult
	params := protocol.InitializeParams{
		Host:        protocol.HostInfo{Version: "2.7.0", OS: "linux", Arch: "amd64", Locale: "en"},
		Settings:    map[string]any{"default_propagation_timeout_seconds": float64(120)},
		Permissions: []string{protocol.PermissionNetwork},
	}
	if err := host.Call(t.Context(), protocol.MethodInitialize, params, &init); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if init.APIVersion != protocol.APIVersion {
		t.Fatalf("api_version = %d", init.APIVersion)
	}
	if len(init.Capabilities) != 1 || init.Capabilities[0] != protocol.CapabilityDNS01 {
		t.Fatalf("capabilities = %v", init.Capabilities)
	}

	if err := host.Notify(t.Context(), protocol.MethodInitialized, nil); err != nil {
		t.Fatalf("initialized: %v", err)
	}

	// dns01.options for "manual": it needs no credentials and no network, and
	// it reports both a propagation timeout and a sequential interval.
	var opts protocol.DNS01OptionsResult
	if err := host.Call(t.Context(), protocol.MethodDNS01Options,
		protocol.DNS01OptionsParams{Provider: "manual"}, &opts); err != nil {
		t.Fatalf("dns01.options: %v", err)
	}
	if opts.PropagationTimeoutSeconds != 60 {
		t.Fatalf("propagation timeout = %d, want lego's default of 60", opts.PropagationTimeoutSeconds)
	}
	if opts.PollingIntervalSeconds != 2 {
		t.Fatalf("polling interval = %d, want 2", opts.PollingIntervalSeconds)
	}
	if opts.SequentialIntervalSeconds != 60 {
		t.Fatalf("sequential interval = %d, want 60", opts.SequentialIntervalSeconds)
	}

	// An unknown provider is a config error naming the offending field.
	err := host.Call(t.Context(), protocol.MethodDNS01Options,
		protocol.DNS01OptionsParams{Provider: "definitely-not-a-provider"}, nil)
	rpcErr, ok := err.(*protocol.Error)
	if !ok {
		t.Fatalf("err = %v, want *protocol.Error", err)
	}
	if rpcErr.Code != protocol.CodeInvalidConfig {
		t.Fatalf("code = %d, want %d", rpcErr.Code, protocol.CodeInvalidConfig)
	}

	// dns01.validate rejects a provider whose credentials are missing.
	err = host.Call(t.Context(), protocol.MethodDNS01Validate,
		protocol.DNS01ValidateParams{Provider: "cloudflare", Config: map[string]string{}}, nil)
	rpcErr, ok = err.(*protocol.Error)
	if !ok {
		t.Fatalf("validate err = %v, want *protocol.Error", err)
	}
	if rpcErr.Code != protocol.CodeInvalidConfig {
		t.Fatalf("validate code = %d, want %d", rpcErr.Code, protocol.CodeInvalidConfig)
	}
	data, ok := rpcErr.Data.(map[string]any)
	if !ok || data["field"] == "" {
		t.Fatalf("validate data = %#v, want a field name", rpcErr.Data)
	}

	// plugin.shutdown then plugin.exit.
	if err := host.Call(t.Context(), protocol.MethodShutdown, nil, nil); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := host.Notify(t.Context(), protocol.MethodExit, nil); err != nil {
		t.Fatalf("exit: %v", err)
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run = %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the plugin did not exit after plugin.exit")
	}
}

// TestValidateAcceptsCompleteCredentials makes sure a filled in configuration
// passes without touching the vendor API.
func TestValidateAcceptsCompleteCredentials(t *testing.T) {
	handler := provider.New()

	err := handler.Validate(t.Context(), "cloudflare", map[string]string{
		"CF_DNS_API_TOKEN": "not-a-real-token",
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
}
