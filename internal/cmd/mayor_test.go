package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGetMayorSessionName(t *testing.T) {
	// Test that getMayorSessionName returns the expected session name
	sessionName := getMayorSessionName()
	expected := "hq-mayor"

	if sessionName != expected {
		t.Errorf("getMayorSessionName() = %q, want %q", sessionName, expected)
	}
}

func TestMayorCommandStructure(t *testing.T) {
	// Verify mayor command has expected subcommands
	expectedSubcommands := map[string]bool{
		"start":  true,
		"attach": true,
		"stop":   true,
		"status": true,
	}

	for _, cmd := range mayorCmd.Commands() {
		name := cmd.Name()
		if !expectedSubcommands[name] {
			t.Errorf("unexpected subcommand: %s", name)
		}
		delete(expectedSubcommands, name)
	}

	// Check for missing subcommands
	for name := range expectedSubcommands {
		t.Errorf("missing expected subcommand: %s", name)
	}
}

func TestMayorCommandAliases(t *testing.T) {
	// Verify attach command has the expected alias
	var attachCmd *cobra.Command
	for _, cmd := range mayorCmd.Commands() {
		if cmd.Name() == "attach" {
			attachCmd = cmd
			break
		}
	}

	if attachCmd == nil {
		t.Fatal("attach subcommand not found")
	}

	expectedAliases := []string{"at"}
	if len(attachCmd.Aliases) != len(expectedAliases) {
		t.Errorf("attach command has %d aliases, want %d", len(attachCmd.Aliases), len(expectedAliases))
	}

	for i, alias := range expectedAliases {
		if i >= len(attachCmd.Aliases) || attachCmd.Aliases[i] != alias {
			t.Errorf("attach command alias[%d] = %q, want %q", i,
				func() string {
					if i < len(attachCmd.Aliases) {
						return attachCmd.Aliases[i]
					}
					return "<missing>"
				}(), alias)
		}
	}
}

func TestMayorCommandFlags(t *testing.T) {
	tests := []struct {
		subcommand string
		flagName   string
		flagType   string
	}{
		{"start", "continue", "bool"},
		{"start", "agent", "string"},
		{"stop", "grace-period", "int"},
		{"status", "json", "bool"},
	}

	for _, tt := range tests {
		var cmd *cobra.Command
		for _, c := range mayorCmd.Commands() {
			if c.Name() == tt.subcommand {
				cmd = c
				break
			}
		}

		if cmd == nil {
			t.Errorf("subcommand %q not found", tt.subcommand)
			continue
		}

		flag := cmd.Flags().Lookup(tt.flagName)
		if flag == nil {
			t.Errorf("%s subcommand missing flag: %s", tt.subcommand, tt.flagName)
			continue
		}

		if flag.Value.Type() != tt.flagType {
			t.Errorf("%s --%s flag type = %s, want %s",
				tt.subcommand, tt.flagName, flag.Value.Type(), tt.flagType)
		}
	}
}
