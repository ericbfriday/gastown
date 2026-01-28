package planconvert

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// OutputFormat represents the output format for the conversion.
type OutputFormat string

const (
	FormatJSON       OutputFormat = "json"
	FormatJSONL      OutputFormat = "jsonl"
	FormatPretty     OutputFormat = "pretty"
	FormatBeadsShell OutputFormat = "shell"
)

// WriteEpic writes the epic to the specified output.
func WriteEpic(epic *Epic, writer io.Writer, format OutputFormat) error {
	switch format {
	case FormatJSON:
		return writeJSON(epic, writer)
	case FormatJSONL:
		return writeJSONL(epic, writer)
	case FormatPretty:
		return writePretty(epic, writer)
	case FormatBeadsShell:
		return writeBeadsShell(epic, writer)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// writeJSON writes the epic as formatted JSON.
func writeJSON(epic *Epic, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(epic)
}

// writeJSONL writes the epic and tasks as JSONL (beads format).
func writeJSONL(epic *Epic, writer io.Writer) error {
	// Convert epic to beads format
	epicBead := epicToBead(epic)

	// Write epic
	epicJSON, err := json.Marshal(epicBead)
	if err != nil {
		return fmt.Errorf("failed to marshal epic: %w", err)
	}
	if _, err := writer.Write(epicJSON); err != nil {
		return err
	}
	if _, err := writer.Write([]byte("\n")); err != nil {
		return err
	}

	// Write each subtask
	for _, task := range epic.Subtasks {
		taskBead := taskToBead(&task, epic.ID)
		taskJSON, err := json.Marshal(taskBead)
		if err != nil {
			return fmt.Errorf("failed to marshal task: %w", err)
		}
		if _, err := writer.Write(taskJSON); err != nil {
			return err
		}
		if _, err := writer.Write([]byte("\n")); err != nil {
			return err
		}
	}

	return nil
}

// writePretty writes a human-readable summary.
func writePretty(epic *Epic, writer io.Writer) error {
	fmt.Fprintf(writer, "Epic: %s\n", epic.Title)
	fmt.Fprintf(writer, "ID: %s\n", epic.ID)
	fmt.Fprintf(writer, "Priority: %d\n", epic.Priority)
	fmt.Fprintf(writer, "Source: %s\n\n", epic.SourceFile)

	fmt.Fprintf(writer, "Tasks (%d):\n", len(epic.Subtasks))
	for i, task := range epic.Subtasks {
		fmt.Fprintf(writer, "  %d. [%s] %s\n", i+1, task.ID, task.Title)
		if task.Phase != "" {
			fmt.Fprintf(writer, "     Phase: %s\n", task.Phase)
		}
	}

	return nil
}

// writeBeadsShell writes bd CLI commands to create the beads.
func writeBeadsShell(epic *Epic, writer io.Writer) error {
	fmt.Fprintln(writer, "#!/usr/bin/env bash")
	fmt.Fprintln(writer, "# Auto-generated beads creation script")
	fmt.Fprintln(writer, "# Source:", epic.SourceFile)
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "set -e")
	fmt.Fprintln(writer, "")

	// Create epic
	epicDesc := strings.ReplaceAll(epic.Description, "\"", "\\\"")
	epicDesc = strings.ReplaceAll(epicDesc, "\n", "\\n")

	fmt.Fprintf(writer, "# Create epic\n")
	fmt.Fprintf(writer, "EPIC_ID=$(bd create \\\n")
	fmt.Fprintf(writer, "  --title \"%s\" \\\n", epic.Title)
	fmt.Fprintf(writer, "  --description \"%s\" \\\n", epicDesc)
	fmt.Fprintf(writer, "  --type epic \\\n")
	fmt.Fprintf(writer, "  --priority %d)\n\n", epic.Priority)

	// Create tasks
	for i, task := range epic.Subtasks {
		taskDesc := strings.ReplaceAll(task.Description, "\"", "\\\"")
		taskDesc = strings.ReplaceAll(taskDesc, "\n", "\\n")

		fmt.Fprintf(writer, "# Task %d: %s\n", i+1, task.Title)
		fmt.Fprintf(writer, "bd create \\\n")
		fmt.Fprintf(writer, "  --title \"%s\" \\\n", task.Title)
		fmt.Fprintf(writer, "  --description \"%s\" \\\n", taskDesc)
		fmt.Fprintf(writer, "  --type task \\\n")
		fmt.Fprintf(writer, "  --priority %d \\\n", task.Priority)
		fmt.Fprintf(writer, "  --depends-on \"$EPIC_ID\"\n\n")
	}

	fmt.Fprintln(writer, "echo \"Created epic $EPIC_ID with\", len(epic.Subtasks), \"tasks\"")

	return nil
}

// epicToBead converts an Epic to beads JSONL format.
func epicToBead(epic *Epic) map[string]interface{} {
	bead := map[string]interface{}{
		"id":          epic.ID,
		"title":       epic.Title,
		"description": epic.Description,
		"status":      epic.Status,
		"priority":    epic.Priority,
		"issue_type":  epic.IssueType,
		"created_at":  epic.CreatedAt.Format("2006-01-02T15:04:05.000000-07:00"),
		"updated_at":  epic.UpdatedAt.Format("2006-01-02T15:04:05.000000-07:00"),
	}

	if len(epic.Dependencies) > 0 {
		bead["dependencies"] = epic.Dependencies
	}

	return bead
}

// taskToBead converts a task Bead to beads JSONL format.
func taskToBead(task *Bead, epicID string) map[string]interface{} {
	bead := map[string]interface{}{
		"id":          task.ID,
		"title":       task.Title,
		"description": task.Description,
		"status":      task.Status,
		"priority":    task.Priority,
		"issue_type":  task.IssueType,
		"created_at":  task.CreatedAt.Format("2006-01-02T15:04:05.000000-07:00"),
		"updated_at":  task.UpdatedAt.Format("2006-01-02T15:04:05.000000-07:00"),
	}

	// Add dependency on epic
	deps := []Dependency{{
		IssueID:     task.ID,
		DependsOnID: epicID,
		Type:        "blocks",
		CreatedAt:   task.CreatedAt,
		CreatedBy:   "plan-converter",
	}}

	// Add any other dependencies
	deps = append(deps, task.Dependencies...)

	if len(deps) > 0 {
		bead["dependencies"] = deps
	}

	return bead
}

// SaveToFile saves the epic to a file.
func SaveToFile(epic *Epic, path string, format OutputFormat) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return WriteEpic(epic, file, format)
}
