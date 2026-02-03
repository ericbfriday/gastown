package beads

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/steveyegge/gastown/internal/filelock"
)

// EpicTemplate represents an epic template definition.
type EpicTemplate struct {
	Name        string                 `toml:"name"`
	Description string                 `toml:"description"`
	Version     int                    `toml:"version"`
	Vars        map[string]TemplateVar `toml:"vars"`
	Epic        EpicDefinition         `toml:"epic"`
	Subtasks    []SubtaskDefinition    `toml:"subtasks"`
}

// TemplateVar defines a template variable with metadata.
type TemplateVar struct {
	Description string `toml:"description"`
	Required    bool   `toml:"required"`
	Default     string `toml:"default"`
}

// EpicDefinition defines the epic issue to create.
type EpicDefinition struct {
	Title       string `toml:"title"`
	Description string `toml:"description"`
	Priority    string `toml:"priority"`
	Type        string `toml:"type"`
}

// SubtaskDefinition defines a subtask template.
type SubtaskDefinition struct {
	Title       string `toml:"title"`
	Description string `toml:"description"`
	Type        string `toml:"type"`
	Priority    string `toml:"priority"`
	ExpandOver  string `toml:"expand_over"` // Variable name to expand over (comma-separated list)
}

// ExpandedTemplate contains the epic and expanded subtasks ready for creation.
type ExpandedTemplate struct {
	Epic     CreateOptions
	Subtasks []CreateOptions
}

// LoadTemplate loads a template from the templates directory.
func LoadTemplate(name string) (*EpicTemplate, error) {
	// Try .beads/templates/ in current directory first
	beadsDir := ".beads"
	if bd := os.Getenv("BEADS_DIR"); bd != "" {
		beadsDir = bd
	}

	templatePath := filepath.Join(beadsDir, "templates", name+".template.toml")

	// Check if file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	var tmpl EpicTemplate
	err := filelock.WithReadLock(templatePath, func() error {
		_, decodeErr := toml.DecodeFile(templatePath, &tmpl)
		return decodeErr
	})
	if err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", name, err)
	}

	return &tmpl, nil
}

// ListTemplates returns all available template names.
func ListTemplates() ([]string, error) {
	beadsDir := ".beads"
	if bd := os.Getenv("BEADS_DIR"); bd != "" {
		beadsDir = bd
	}

	templatesDir := filepath.Join(beadsDir, "templates")

	// Check if templates directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Use read lock with .list.lock marker for directory listing
	listLockPath := filepath.Join(templatesDir, ".list.lock")
	var templates []string

	err := filelock.WithReadLock(listLockPath, func() error {
		entries, err := os.ReadDir(templatesDir)
		if err != nil {
			return fmt.Errorf("reading templates directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".template.toml") {
				// Strip .template.toml suffix
				templates = append(templates, strings.TrimSuffix(name, ".template.toml"))
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return templates, nil
}

// ExpandTemplate expands a template with the given variables.
func ExpandTemplate(tmpl *EpicTemplate, vars map[string]string) (*ExpandedTemplate, error) {
	// Validate required variables
	for varName, varDef := range tmpl.Vars {
		if varDef.Required {
			if _, ok := vars[varName]; !ok {
				return nil, fmt.Errorf("required variable missing: %s (%s)", varName, varDef.Description)
			}
		}
	}

	// Apply defaults for missing variables
	for varName, varDef := range tmpl.Vars {
		if _, ok := vars[varName]; !ok && varDef.Default != "" {
			vars[varName] = varDef.Default
		}
	}

	// Expand epic
	epicTitle, err := expandString(tmpl.Epic.Title, vars)
	if err != nil {
		return nil, fmt.Errorf("expanding epic title: %w", err)
	}

	epicDesc, err := expandString(tmpl.Epic.Description, vars)
	if err != nil {
		return nil, fmt.Errorf("expanding epic description: %w", err)
	}

	priority := 2 // default priority
	if tmpl.Epic.Priority != "" {
		priorityStr, err := expandString(tmpl.Epic.Priority, vars)
		if err != nil {
			return nil, fmt.Errorf("expanding epic priority: %w", err)
		}
		// Parse priority
		fmt.Sscanf(priorityStr, "%d", &priority)
	}

	epic := CreateOptions{
		Title:       epicTitle,
		Description: epicDesc,
		Priority:    priority,
		Type:        "epic",
	}

	// Expand subtasks
	var subtasks []CreateOptions
	for _, subtaskDef := range tmpl.Subtasks {
		if subtaskDef.ExpandOver != "" {
			// This subtask should be expanded over a list variable
			expandVar := subtaskDef.ExpandOver
			listValue, ok := vars[expandVar]
			if !ok {
				return nil, fmt.Errorf("expand_over variable not found: %s", expandVar)
			}

			// Split comma-separated list
			items := strings.Split(listValue, ",")
			for _, item := range items {
				item = strings.TrimSpace(item)

				// Create context with the item
				itemVars := make(map[string]string)
				for k, v := range vars {
					itemVars[k] = v
				}

				// Add singular version of the expand variable
				// e.g., if expanding over "rigs", add "rig" with the current item
				singularKey := strings.TrimSuffix(expandVar, "s")
				if singularKey == expandVar {
					// Try other common patterns
					if strings.HasSuffix(expandVar, "es") {
						singularKey = strings.TrimSuffix(expandVar, "es")
					} else if strings.HasSuffix(expandVar, "ies") {
						singularKey = strings.TrimSuffix(expandVar, "ies") + "y"
					}
				}
				itemVars[singularKey] = item
				itemVars[expandVar+"_item"] = item // Also add <var>_item as fallback

				subtask, err := expandSubtask(subtaskDef, itemVars)
				if err != nil {
					return nil, fmt.Errorf("expanding subtask for %s=%s: %w", expandVar, item, err)
				}
				subtasks = append(subtasks, subtask)
			}
		} else {
			// Regular subtask, no expansion
			subtask, err := expandSubtask(subtaskDef, vars)
			if err != nil {
				return nil, fmt.Errorf("expanding subtask: %w", err)
			}
			subtasks = append(subtasks, subtask)
		}
	}

	return &ExpandedTemplate{
		Epic:     epic,
		Subtasks: subtasks,
	}, nil
}

// expandSubtask expands a single subtask definition.
func expandSubtask(def SubtaskDefinition, vars map[string]string) (CreateOptions, error) {
	title, err := expandString(def.Title, vars)
	if err != nil {
		return CreateOptions{}, fmt.Errorf("expanding title: %w", err)
	}

	desc, err := expandString(def.Description, vars)
	if err != nil {
		return CreateOptions{}, fmt.Errorf("expanding description: %w", err)
	}

	priority := 2
	if def.Priority != "" {
		priorityStr, err := expandString(def.Priority, vars)
		if err != nil {
			return CreateOptions{}, fmt.Errorf("expanding priority: %w", err)
		}
		fmt.Sscanf(priorityStr, "%d", &priority)
	}

	issueType := "task"
	if def.Type != "" {
		issueType = def.Type
	}

	return CreateOptions{
		Title:       title,
		Description: desc,
		Priority:    priority,
		Type:        issueType,
	}, nil
}

// expandString expands template variables in a string using simple {{var}} substitution.
// This is simpler than Go's text/template and matches the formula pattern.
func expandString(tmplStr string, vars map[string]string) (string, error) {
	result := tmplStr

	// Replace all {{var}} patterns
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Check for any remaining unsubstituted variables
	if strings.Contains(result, "{{") && strings.Contains(result, "}}") {
		// Extract unsubstituted variable name for error message
		start := strings.Index(result, "{{")
		end := strings.Index(result[start:], "}}") + start
		if end > start {
			varName := result[start+2 : end]
			return "", fmt.Errorf("undefined variable: %s", varName)
		}
	}

	return result, nil
}

// CreateFromTemplate creates an epic and its subtasks from a template.
func (b *Beads) CreateFromTemplate(templateName string, vars map[string]string) (*Issue, []*Issue, error) {
	// Load template
	tmpl, err := LoadTemplate(templateName)
	if err != nil {
		return nil, nil, err
	}

	// Expand template
	expanded, err := ExpandTemplate(tmpl, vars)
	if err != nil {
		return nil, nil, err
	}

	// Create epic
	epic, err := b.Create(expanded.Epic)
	if err != nil {
		return nil, nil, fmt.Errorf("creating epic: %w", err)
	}

	// Create subtasks with epic as parent
	var subtasks []*Issue
	for _, subtaskOpts := range expanded.Subtasks {
		subtaskOpts.Parent = epic.ID
		subtask, err := b.Create(subtaskOpts)
		if err != nil {
			// If subtask creation fails, we still return the epic and any created subtasks
			// The user can see partial creation and fix manually
			return epic, subtasks, fmt.Errorf("creating subtask '%s': %w", subtaskOpts.Title, err)
		}
		subtasks = append(subtasks, subtask)
	}

	return epic, subtasks, nil
}
