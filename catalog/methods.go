package catalog

import (
	"regexp"
	"strings"
)

// exampleBlock is one way to sign in as the upstream example shows it.
type exampleBlock struct {
	Name string
	Keys []string
}

var (
	reOrLine      = regexp.MustCompile(`(?i)^\s*#\s*or\b[\s,]*(.*)$`)
	reCommentLine = regexp.MustCompile(`^\s*#\s*(.*)$`)
	reAssignment  = regexp.MustCompile(`^\s*([A-Z][A-Z0-9_]*)=`)
	reNamePrefix  = regexp.MustCompile(`(?i)^(?:setup using|using|with)\s+`)
)

// parseExample splits an upstream example into blocks at "# or" lines and
// returns the keys each block assigns, in order.
func parseExample(example string) []exampleBlock {
	var (
		blocks  []exampleBlock
		current exampleBlock
		started bool
	)
	flush := func() {
		if started {
			blocks = append(blocks, current)
		}
		current = exampleBlock{}
		started = true
	}
	flush()

	for _, line := range strings.Split(example, "\n") {
		if m := reOrLine.FindStringSubmatch(line); m != nil {
			flush()
			current.Name = methodName(m[1])
			continue
		}
		if m := reCommentLine.FindStringSubmatch(line); m != nil {
			if current.Name == "" {
				current.Name = methodName(m[1])
			}
			continue
		}
		if m := reAssignment.FindStringSubmatch(line); m != nil {
			current.Keys = append(current.Keys, m[1])
		}
	}
	blocks = append(blocks, current)

	if len(blocks) < 2 {
		return nil
	}
	return blocks
}

// methodName turns a comment such as "Setup using instance RAM role" into a
// method name.
func methodName(comment string) string {
	name := trimSentence(comment)
	name = reNamePrefix.ReplaceAllString(name, "")
	return capitalize(trimSentence(name))
}
