package analyzer

import (
	"testing"

	"github.com/steveyegge/gastown/internal/planoracle/models"
)

func TestExtractMarkdownTasks(t *testing.T) {
	decomposer := NewDecomposer(models.NewHistoricalMetrics())

	tests := []struct {
		name        string
		description string
		wantCount   int
		wantFirst   string
	}{
		{
			name: "simple task list",
			description: `
## Tasks
- [ ] Design data models
- [ ] Implement core logic
- [ ] Write tests
`,
			wantCount: 3,
			wantFirst: "Design data models",
		},
		{
			name: "mixed checked and unchecked",
			description: `
- [x] Already done
- [ ] TODO: Implement feature
- [ ] TODO: Add tests
`,
			wantCount: 3,
			wantFirst: "Already done",
		},
		{
			name: "no tasks",
			description: `
This is just a description without tasks.
Nothing to extract here.
`,
			wantCount: 0,
		},
		{
			name: "asterisk style",
			description: `
* [ ] Task one
* [ ] Task two
`,
			wantCount: 2,
			wantFirst: "Task one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := decomposer.extractMarkdownTasks(tt.description)

			if len(tasks) != tt.wantCount {
				t.Errorf("extractMarkdownTasks() got %d tasks, want %d", len(tasks), tt.wantCount)
			}

			if tt.wantCount > 0 && tasks[0].Title != tt.wantFirst {
				t.Errorf("extractMarkdownTasks() first task = %q, want %q", tasks[0].Title, tt.wantFirst)
			}
		})
	}
}

func TestExtractSteps(t *testing.T) {
	decomposer := NewDecomposer(models.NewHistoricalMetrics())

	tests := []struct {
		name        string
		description string
		wantCount   int
		wantFirst   string
	}{
		{
			name: "numbered steps",
			description: `
## Steps

1. Design the architecture
2. Implement core logic
3. Add unit tests
4. Write documentation
`,
			wantCount: 4,
			wantFirst: "Design the architecture",
		},
		{
			name: "phases section",
			description: `
## Phases

1. Research and design
2. Implementation
3. Testing
`,
			wantCount: 3,
			wantFirst: "Research and design",
		},
		{
			name: "no steps section",
			description: `
## Description

This has no steps section.
`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := decomposer.extractSteps(tt.description)

			if len(tasks) != tt.wantCount {
				t.Errorf("extractSteps() got %d tasks, want %d", len(tasks), tt.wantCount)
			}

			if tt.wantCount > 0 && tasks[0].Title != tt.wantFirst {
				t.Errorf("extractSteps() first task = %q, want %q", tasks[0].Title, tt.wantFirst)
			}
		})
	}
}

func TestApplyTemplate(t *testing.T) {
	decomposer := NewDecomposer(models.NewHistoricalMetrics())

	tests := []struct {
		name      string
		itemType  string
		wantCount int
		wantFirst string
	}{
		{
			name:      "epic template",
			itemType:  "epic",
			wantCount: 4,
			wantFirst: "Design and architecture",
		},
		{
			name:      "feature template",
			itemType:  "feature",
			wantCount: 3,
			wantFirst: "Implementation",
		},
		{
			name:      "convoy template",
			itemType:  "convoy",
			wantCount: 3,
			wantFirst: "Leg 1: (parallel work)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &models.WorkItem{
				Type: tt.itemType,
			}

			tasks := decomposer.applyTemplate(item)

			if len(tasks) != tt.wantCount {
				t.Errorf("applyTemplate(%s) got %d tasks, want %d", tt.itemType, len(tasks), tt.wantCount)
			}

			if tt.wantCount > 0 && tasks[0].Title != tt.wantFirst {
				t.Errorf("applyTemplate(%s) first task = %q, want %q", tt.itemType, tasks[0].Title, tt.wantFirst)
			}
		})
	}
}

func TestEstimateSubtask(t *testing.T) {
	decomposer := NewDecomposer(models.NewHistoricalMetrics())
	parent := &models.WorkItem{Type: "epic"}

	tests := []struct {
		title    string
		wantDays float64
	}{
		{"Design data models", 2.0},
		{"Implement core logic", 3.0},
		{"Add unit tests", 1.0},
		{"Write documentation", 0.5},
		{"Refactor authentication", 1.5},
		{"Unknown task type", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			subtask := &models.SubTask{
				Title: tt.title,
			}

			estimate := decomposer.estimateSubtask(subtask, parent)

			if estimate != tt.wantDays {
				t.Errorf("estimateSubtask(%q) = %.1f days, want %.1f days", tt.title, estimate, tt.wantDays)
			}
		})
	}
}

func TestDecompose(t *testing.T) {
	decomposer := NewDecomposer(models.NewHistoricalMetrics())

	tests := []struct {
		name        string
		item        *models.WorkItem
		wantError   bool
		wantSource  string
		minSubtasks int
	}{
		{
			name: "epic with markdown tasks",
			item: &models.WorkItem{
				ID:   "test-epic-1",
				Type: "epic",
				Description: `
## Tasks
- [ ] Design data models
- [ ] Implement core logic
- [ ] Write tests
`,
			},
			wantError:   false,
			wantSource:  "markdown",
			minSubtasks: 3,
		},
		{
			name: "feature with steps",
			item: &models.WorkItem{
				ID:   "test-feat-1",
				Type: "feature",
				Description: `
## Steps
1. Implement API endpoint
2. Add validation
3. Write unit tests
`,
			},
			wantError:   false,
			wantSource:  "steps",
			minSubtasks: 3,
		},
		{
			name: "epic with no explicit structure (uses template)",
			item: &models.WorkItem{
				ID:          "test-epic-2",
				Type:        "epic",
				Description: "Just a plain description",
			},
			wantError:   false,
			wantSource:  "template",
			minSubtasks: 4, // Epic template has 4 tasks
		},
		{
			name: "task cannot be decomposed",
			item: &models.WorkItem{
				ID:   "test-task-1",
				Type: "task",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decomposer.Decompose(tt.item)

			if tt.wantError {
				if err == nil {
					t.Errorf("Decompose() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Decompose() unexpected error: %v", err)
			}

			if result.Source != tt.wantSource {
				t.Errorf("Decompose() source = %q, want %q", result.Source, tt.wantSource)
			}

			if len(result.Subtasks) < tt.minSubtasks {
				t.Errorf("Decompose() got %d subtasks, want at least %d", len(result.Subtasks), tt.minSubtasks)
			}

			// Verify all subtasks have estimates
			for i, subtask := range result.Subtasks {
				if subtask.EstimatedDays <= 0 {
					t.Errorf("Subtask %d has no estimate", i)
				}
			}

			// Verify total estimate is sum of subtasks
			expectedTotal := 0.0
			for _, subtask := range result.Subtasks {
				expectedTotal += subtask.EstimatedDays
			}

			if result.TotalEstimate != expectedTotal {
				t.Errorf("TotalEstimate = %.1f, want %.1f", result.TotalEstimate, expectedTotal)
			}
		})
	}
}
