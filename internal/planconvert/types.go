package planconvert

import "time"

// PlanDocument represents a parsed planning document.
type PlanDocument struct {
	Title    string
	FilePath string
	Sections []Section
	Metadata Metadata
}

// Metadata contains document-level metadata.
type Metadata struct {
	Version string
	Status  string
	Date    string
	Author  string
	Phase   string
}

// Section represents a major section in the document.
type Section struct {
	Title    string
	Level    int // Header level (1-6)
	Content  string
	Subsections []Section
	Tasks    []Task
	Type     SectionType
}

// SectionType indicates the type of section.
type SectionType string

const (
	SectionTypePhase         SectionType = "phase"
	SectionTypeImplementation SectionType = "implementation"
	SectionTypeTasks         SectionType = "tasks"
	SectionTypeOverview      SectionType = "overview"
	SectionTypeGeneric       SectionType = "generic"
)

// Task represents a work item extracted from the document.
type Task struct {
	Title        string
	Description  string
	Phase        string
	Dependencies []string
	Deliverables []string
	Criteria     []string
	Priority     int
	Order        int // Order within phase
}

// Epic represents a beads epic with subtasks.
type Epic struct {
	ID           string
	Title        string
	Description  string
	Status       string
	Priority     int
	IssueType    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	SourceFile   string
	Subtasks     []Bead
	Dependencies []Dependency
}

// Bead represents a beads issue.
type Bead struct {
	ID           string
	Title        string
	Description  string
	Status       string
	Priority     int
	IssueType    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Dependencies []Dependency
	Phase        string
	Order        int
}

// Dependency represents a dependency between beads.
type Dependency struct {
	IssueID     string `json:"issue_id"`
	DependsOnID string `json:"depends_on_id"`
	Type        string `json:"type"` // "blocks", "depends", "related"
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string `json:"created_by"`
}

// ConversionOptions contains options for converting a plan to epic.
type ConversionOptions struct {
	Prefix         string // ID prefix for generated beads
	Priority       int    // Default priority
	DryRun         bool   // Preview mode
	OutputFile     string // Output file path
	CreateBeads    bool   // Create beads directly via CLI
	TargetRig      string // Target rig for bead creation
	IncludeContext bool   // Include full document context in descriptions
}
