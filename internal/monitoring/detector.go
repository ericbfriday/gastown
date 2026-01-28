package monitoring

import (
	"regexp"
	"strings"
	"time"
)

// PatternRegistry manages activity patterns for status detection.
type PatternRegistry struct {
	patterns []ActivityPattern
}

// NewPatternRegistry creates a new pattern registry with default patterns.
func NewPatternRegistry() *PatternRegistry {
	r := &PatternRegistry{
		patterns: make([]ActivityPattern, 0),
	}
	r.registerDefaultPatterns()
	return r
}

// registerDefaultPatterns adds the standard set of activity patterns.
func (r *PatternRegistry) registerDefaultPatterns() {
	// High priority patterns (specific states)
	r.AddPattern(ActivityPattern{
		Pattern:     "BLOCKED:",
		Status:      StatusBlocked,
		Priority:    100,
		Description: "Agent explicitly blocked",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "ERROR:",
		Status:      StatusError,
		Priority:    100,
		Description: "Error condition detected",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Error:",
		Status:      StatusError,
		Priority:    100,
		Description: "Error condition detected",
	})

	// Medium priority patterns (working states)
	r.AddPattern(ActivityPattern{
		Pattern:     "Thinking...",
		Status:      StatusThinking,
		Priority:    80,
		Description: "Agent thinking/processing",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Reading",
		Status:      StatusWorking,
		Priority:    70,
		Description: "Reading files",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Writing",
		Status:      StatusWorking,
		Priority:    70,
		Description: "Writing files",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Editing",
		Status:      StatusWorking,
		Priority:    70,
		Description: "Editing files",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Searching",
		Status:      StatusWorking,
		Priority:    70,
		Description: "Searching code",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Running",
		Status:      StatusWorking,
		Priority:    70,
		Description: "Running commands",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Executing",
		Status:      StatusWorking,
		Priority:    70,
		Description: "Executing tasks",
	})

	// Tool use patterns
	r.AddPattern(ActivityPattern{
		Pattern:     "Using tool:",
		Status:      StatusWorking,
		Priority:    75,
		Description: "Using a tool",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "<function_calls>",
		Status:      StatusWorking,
		Priority:    75,
		Description: "Making function calls",
	})

	// Review patterns
	r.AddPattern(ActivityPattern{
		Pattern:     "Reviewing",
		Status:      StatusReviewing,
		Priority:    70,
		Description: "Reviewing changes",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Checking",
		Status:      StatusReviewing,
		Priority:    65,
		Description: "Checking work",
	})

	// Waiting patterns
	r.AddPattern(ActivityPattern{
		Pattern:     "waiting for",
		Status:      StatusWaiting,
		Priority:    60,
		Description: "Waiting for input",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Would you like",
		Status:      StatusWaiting,
		Priority:    60,
		Description: "Asking user for input",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "Should I",
		Status:      StatusWaiting,
		Priority:    60,
		Description: "Asking user for decision",
	})

	// Completion patterns
	r.AddPattern(ActivityPattern{
		Pattern:     "completed",
		Status:      StatusAvailable,
		Priority:    50,
		Description: "Task completed",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "finished",
		Status:      StatusAvailable,
		Priority:    50,
		Description: "Task finished",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     "done",
		Status:      StatusAvailable,
		Priority:    50,
		Description: "Task done",
	})

	// Regex patterns for more complex matching
	r.AddPattern(ActivityPattern{
		Pattern:     `(?i)(test|build|compile|install|deploy)ing`,
		Status:      StatusWorking,
		Priority:    65,
		IsRegex:     true,
		Description: "Build/test operations",
	})

	r.AddPattern(ActivityPattern{
		Pattern:     `(?i)(debug|diagnos|investigat)ing`,
		Status:      StatusWorking,
		Priority:    65,
		IsRegex:     true,
		Description: "Debugging/investigation",
	})
}

// AddPattern adds a new activity pattern to the registry.
func (r *PatternRegistry) AddPattern(pattern ActivityPattern) {
	r.patterns = append(r.patterns, pattern)
}

// DetectStatus analyzes output text and returns the detected status.
// Returns the matched status and true if a pattern matched, or StatusAvailable and false if no match.
func (r *PatternRegistry) DetectStatus(output string) (AgentStatus, bool) {
	if output == "" {
		return StatusAvailable, false
	}

	// Track best match
	var bestMatch *ActivityPattern
	var bestPriority int = -1

	for i := range r.patterns {
		pattern := &r.patterns[i]

		matched := false
		if pattern.IsRegex {
			matched = r.matchRegex(pattern.Pattern, output)
		} else {
			matched = strings.Contains(output, pattern.Pattern)
		}

		if matched && pattern.Priority > bestPriority {
			bestMatch = pattern
			bestPriority = pattern.Priority
		}
	}

	if bestMatch != nil {
		return bestMatch.Status, true
	}

	return StatusAvailable, false
}

// matchRegex compiles and matches a regex pattern.
func (r *PatternRegistry) matchRegex(pattern string, text string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(text)
}

// ListPatterns returns all registered patterns.
func (r *PatternRegistry) ListPatterns() []ActivityPattern {
	result := make([]ActivityPattern, len(r.patterns))
	copy(result, r.patterns)
	return result
}

// GetPatternsByStatus returns all patterns that match a specific status.
func (r *PatternRegistry) GetPatternsByStatus(status AgentStatus) []ActivityPattern {
	result := make([]ActivityPattern, 0)
	for _, pattern := range r.patterns {
		if pattern.Status == status {
			result = append(result, pattern)
		}
	}
	return result
}

// Detector provides activity detection from agent output.
type Detector struct {
	registry *PatternRegistry
}

// NewDetector creates a new activity detector.
func NewDetector() *Detector {
	return &Detector{
		registry: NewPatternRegistry(),
	}
}

// NewDetectorWithRegistry creates a detector with a custom pattern registry.
func NewDetectorWithRegistry(registry *PatternRegistry) *Detector {
	return &Detector{
		registry: registry,
	}
}

// Detect analyzes output and returns a status report.
func (d *Detector) Detect(agentID string, output string) StatusReport {
	status, matched := d.registry.DetectStatus(output)

	source := SourceInferred
	if !matched {
		// No pattern matched, default to available
		status = StatusAvailable
	}

	return StatusReport{
		AgentID:   agentID,
		Status:    status,
		Source:    source,
		Timestamp: time.Now(),
		Message:   output,
	}
}

// DetectMultiline analyzes multiple lines of output and returns the highest priority match.
func (d *Detector) DetectMultiline(agentID string, lines []string) StatusReport {
	var bestStatus AgentStatus = StatusAvailable
	var bestMatched bool = false
	var bestPriority int = -1
	var matchedLine string

	for _, line := range lines {
		status, matched := d.registry.DetectStatus(line)
		if matched {
			// Get priority of this match
			patterns := d.registry.GetPatternsByStatus(status)
			priority := 0
			for _, p := range patterns {
				if p.Priority > priority {
					priority = p.Priority
				}
			}

			if priority > bestPriority {
				bestStatus = status
				bestMatched = true
				bestPriority = priority
				matchedLine = line
			}
		}
	}

	if !bestMatched {
		matchedLine = ""
		if len(lines) > 0 {
			matchedLine = lines[len(lines)-1] // Use last line
		}
	}

	return StatusReport{
		AgentID:   agentID,
		Status:    bestStatus,
		Source:    SourceInferred,
		Timestamp: time.Now(),
		Message:   matchedLine,
	}
}

// AddPattern adds a custom pattern to the detector's registry.
func (d *Detector) AddPattern(pattern ActivityPattern) {
	d.registry.AddPattern(pattern)
}

// Registry returns the pattern registry (for inspection/modification).
func (d *Detector) Registry() *PatternRegistry {
	return d.registry
}
