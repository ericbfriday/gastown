package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/style"
)

// Template command flags
var (
	templateListJSON bool
	templateShowJSON bool
	templateVars     []string // --var key=value format
)

var templateCmd = &cobra.Command{
	Use:     "template",
	Aliases: []string{"templates", "tmpl"},
	GroupID: GroupWork,
	Short:   "Manage epic templates for batch work",
	RunE:    requireSubcommand,
	Long: `Manage epic templates for creating batch work patterns.

Templates are TOML files that define epics with predefined subtask patterns
and variable substitution. They're useful for creating batch work like
"add feature X to all rigs" or "refactor pattern Y everywhere".

Commands:
  list    List available epic templates
  show    Display template details
  create  Create epic from template with variable expansion

Template location:
  .beads/templates/*.template.toml

Built-in templates:
  cross-rig-feature   Add a feature across multiple rigs
  refactor-pattern    Refactor code pattern across files
  security-audit      Security audit across components
  feature-rollout     Phased feature rollout

Examples:
  gt template list
  gt template show cross-rig-feature
  gt template create cross-rig-feature --var feature="logging" --var rigs="gastown,beads"`,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available epic templates",
	Long: `List all available epic templates.

Templates are loaded from .beads/templates/ directory.

Examples:
  gt template list
  gt template list --json`,
	RunE: runTemplateList,
}

var templateShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Display template details",
	Args:  cobra.ExactArgs(1),
	Long: `Display detailed information about an epic template.

Shows:
  - Template metadata
  - Required and optional variables
  - Epic structure
  - Subtask patterns

Examples:
  gt template show cross-rig-feature
  gt template show security-audit --json`,
	RunE: runTemplateShow,
}

var templateCreateCmd = &cobra.Command{
	Use:   "create <template-name>",
	Short: "Create epic from template",
	Args:  cobra.ExactArgs(1),
	Long: `Create an epic and subtasks from a template with variable expansion.

Variables are provided via --var flags in key=value format.
The template will be expanded and an epic with all subtasks will be created.

Examples:
  gt template create cross-rig-feature \\
    --var feature="logging middleware" \\
    --var rigs="gastown,beads,aardwolf_snd"

  gt template create refactor-pattern \\
    --var pattern="manual error handling" \\
    --var replacement="error wrapper" \\
    --var locations="beads.go,routes.go,catalog.go"

  gt template create security-audit \\
    --var scope="authentication" \\
    --var components="API,CLI,daemon"`,
	RunE: runTemplateCreate,
}

func init() {
	// List flags
	templateListCmd.Flags().BoolVar(&templateListJSON, "json", false, "Output as JSON")

	// Show flags
	templateShowCmd.Flags().BoolVar(&templateShowJSON, "json", false, "Output as JSON")

	// Create flags
	templateCreateCmd.Flags().StringArrayVar(&templateVars, "var", nil, "Template variable in key=value format (repeatable)")

	// Register subcommands
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateCreateCmd)

	// Register to root
	rootCmd.AddCommand(templateCmd)
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	templates, err := beads.ListTemplates()
	if err != nil {
		return fmt.Errorf("listing templates: %w", err)
	}

	if templateListJSON {
		data := map[string]interface{}{
			"templates": templates,
			"count":     len(templates),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found in .beads/templates/")
		fmt.Println()
		fmt.Println("Create templates with .template.toml extension in .beads/templates/")
		return nil
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Available Epic Templates"))

	for _, name := range templates {
		// Try to load template to get description
		tmpl, err := beads.LoadTemplate(name)
		if err != nil {
			fmt.Printf("  %-25s %s\n", name, style.Dim.Render("(error loading)"))
			continue
		}

		fmt.Printf("  %-25s %s\n", name, tmpl.Description)
	}

	fmt.Println()
	fmt.Printf("Use %s for details\n", style.Bold.Render("gt template show <name>"))

	return nil
}

func runTemplateShow(cmd *cobra.Command, args []string) error {
	name := args[0]

	tmpl, err := beads.LoadTemplate(name)
	if err != nil {
		return fmt.Errorf("loading template: %w", err)
	}

	if templateShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tmpl)
	}

	// Human-readable output
	fmt.Printf("%s\n", style.Bold.Render(tmpl.Name))
	fmt.Printf("%s\n\n", tmpl.Description)

	fmt.Printf("%s\n", style.Bold.Render("Variables:"))
	if len(tmpl.Vars) == 0 {
		fmt.Println("  (none)")
	} else {
		for varName, varDef := range tmpl.Vars {
			required := ""
			if varDef.Required {
				required = " " + style.Dim.Render("(required)")
			}
			defaultVal := ""
			if varDef.Default != "" {
				defaultVal = fmt.Sprintf(" %s", style.Dim.Render(fmt.Sprintf("[default: %s]", varDef.Default)))
			}
			fmt.Printf("  %-15s %s%s%s\n", varName, varDef.Description, required, defaultVal)
		}
	}
	fmt.Println()

	fmt.Printf("%s\n", style.Bold.Render("Epic:"))
	fmt.Printf("  Title: %s\n", tmpl.Epic.Title)
	if tmpl.Epic.Priority != "" {
		fmt.Printf("  Priority: %s\n", tmpl.Epic.Priority)
	}
	fmt.Println()

	fmt.Printf("%s (%d)\n", style.Bold.Render("Subtasks:"), len(tmpl.Subtasks))
	for i, subtask := range tmpl.Subtasks {
		expandNote := ""
		if subtask.ExpandOver != "" {
			expandNote = fmt.Sprintf(" %s", style.Dim.Render(fmt.Sprintf("(expands over: %s)", subtask.ExpandOver)))
		}
		fmt.Printf("  %d. %s%s\n", i+1, subtask.Title, expandNote)
		if subtask.ExpandOver != "" {
			fmt.Printf("     %s\n", style.Dim.Render(fmt.Sprintf("Creates one subtask per item in %s", subtask.ExpandOver)))
		}
	}
	fmt.Println()

	fmt.Printf("%s\n", style.Bold.Render("Usage:"))
	fmt.Printf("  gt template create %s", name)

	// Show required variables
	for varName, varDef := range tmpl.Vars {
		if varDef.Required {
			fmt.Printf(" \\\n    --var %s=<value>", varName)
		}
	}
	fmt.Println()

	return nil
}

func runTemplateCreate(cmd *cobra.Command, args []string) error {
	templateName := args[0]

	// Parse variables from --var flags
	vars := make(map[string]string)
	for _, varFlag := range templateVars {
		parts := strings.SplitN(varFlag, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --var format: %s (expected key=value)", varFlag)
		}
		vars[parts[0]] = parts[1]
	}

	// Create beads client
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	bd := beads.New(cwd)

	// Create epic from template
	epic, subtasks, err := bd.CreateFromTemplate(templateName, vars)
	if err != nil {
		return fmt.Errorf("creating from template: %w", err)
	}

	// Display results
	fmt.Printf("%s %s\n", style.Bold.Render("✓ Epic created:"), style.Bold.Render(epic.ID))
	fmt.Printf("  %s\n\n", epic.Title)

	if len(subtasks) > 0 {
		fmt.Printf("%s\n", style.Bold.Render("Subtasks created:"))
		for _, subtask := range subtasks {
			fmt.Printf("  %-12s %s\n", subtask.ID, subtask.Title)
		}
		fmt.Println()
	}

	fmt.Printf("View epic: %s\n", style.Bold.Render(fmt.Sprintf("gt show %s", epic.ID)))
	fmt.Printf("View all:  %s\n", style.Bold.Render(fmt.Sprintf("bd list --parent %s", epic.ID)))

	return nil
}
