package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/workspace/cleanup"
)

var (
	configShow   bool
	configExport bool
	configOutput string
)

func newWorkspaceConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage workspace cleanup configuration",
		Long: `View and manage workspace cleanup configuration.

Examples:
  # Show current configuration
  gt workspace config --show

  # Export default configurations to file
  gt workspace config --export --output cleanup.json

  # View default configuration for crew workspaces
  gt workspace config --show --type crew
`,
		RunE: runWorkspaceConfig,
	}

	cmd.Flags().BoolVar(&configShow, "show", false, "Show current configuration")
	cmd.Flags().BoolVar(&configExport, "export", false, "Export default configurations")
	cmd.Flags().StringVarP(&configOutput, "output", "o", "", "Output file path")
	cmd.Flags().StringVarP(&cleanWorkspaceType, "type", "t", "", "Workspace type (for --show)")

	return cmd
}

func runWorkspaceConfig(cmd *cobra.Command, args []string) error {
	if configExport {
		return exportConfigs()
	}

	if configShow {
		return showConfig()
	}

	// Default: show help
	return cmd.Help()
}

func exportConfigs() error {
	outputPath := configOutput
	if outputPath == "" {
		outputPath = "cleanup-config.json"
	}

	if err := cleanup.ExportDefaultConfigs(outputPath); err != nil {
		return fmt.Errorf("failed to export configs: %w", err)
	}

	fmt.Printf("Default configurations exported to: %s\n", outputPath)
	return nil
}

func showConfig() error {
	workingDir := "."

	if cleanWorkspaceType != "" {
		// Show specific workspace type config
		var wsType cleanup.WorkspaceType
		switch cleanWorkspaceType {
		case "crew":
			wsType = cleanup.WorkspaceTypeCrew
		case "polecat":
			wsType = cleanup.WorkspaceTypePolecat
		case "mayor":
			wsType = cleanup.WorkspaceTypeMayor
		case "refinery":
			wsType = cleanup.WorkspaceTypeRefinery
		case "town":
			wsType = cleanup.WorkspaceTypeTown
		default:
			return fmt.Errorf("unknown workspace type: %s", cleanWorkspaceType)
		}

		config, err := cleanup.LoadConfig(workingDir, wsType)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		fmt.Println(string(data))
		return nil
	}

	// Show all default configs
	configs := cleanup.DefaultConfigs()
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
