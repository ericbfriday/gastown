// plan-to-epic-demo demonstrates the plan-to-epic converter
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/steveyegge/gastown/internal/planconvert"
)

func main() {
	var (
		outputFile string
		format     string
		prefix     string
		priority   int
	)

	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.StringVar(&format, "format", "jsonl", "Output format: jsonl, json, pretty, shell")
	flag.StringVar(&prefix, "prefix", "demo", "ID prefix")
	flag.IntVar(&priority, "priority", 2, "Default priority")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: plan-to-epic-demo <plan-file>")
		os.Exit(1)
	}

	planFile := flag.Arg(0)

	// Parse document
	fmt.Fprintf(os.Stderr, "→ Parsing: %s\n", planFile)
	doc, err := planconvert.ParsePlanDocument(planFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✓ Found: %s\n", doc.Title)

	// Convert to epic
	opts := planconvert.ConversionOptions{
		Prefix:   prefix,
		Priority: priority,
	}

	epic, err := planconvert.ConvertToEpic(doc, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✓ Generated epic with %d tasks\n\n", len(epic.Subtasks))

	// Determine format
	var outFormat planconvert.OutputFormat
	switch format {
	case "jsonl":
		outFormat = planconvert.FormatJSONL
	case "json":
		outFormat = planconvert.FormatJSON
	case "pretty":
		outFormat = planconvert.FormatPretty
	case "shell":
		outFormat = planconvert.FormatBeadsShell
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", format)
		os.Exit(1)
	}

	// Output
	if outputFile != "" {
		err := planconvert.SaveToFile(epic, outputFile, outFormat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "✓ Wrote output to: %s\n", outputFile)
	} else {
		err := planconvert.WriteEpic(epic, os.Stdout, outFormat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	}
}
