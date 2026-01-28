package planconvert

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	// Pattern for phase headers: "### Phase 3.1: Title"
	phaseHeaderRe = regexp.MustCompile(`^#{2,4}\s+Phase\s+(\d+(?:\.\d+)?)[:\s]+(.+)$`)

	// Pattern for metadata in frontmatter
	metadataRe = regexp.MustCompile(`^\*\*([^:]+):\*\*\s+(.+)$`)

	// Pattern for task lists: "1. Task description"
	taskListRe = regexp.MustCompile(`^(\d+)\.\s+(.+)$`)

	// Pattern for checkboxes: "- ✅ Deliverable" or "- [ ] Item"
	checkboxRe = regexp.MustCompile(`^[-*]\s+(?:✅|☐|\[[ x]\])\s+(.+)$`)

	// Pattern for section markers: "**Tasks:**", "**Deliverables:**"
	sectionMarkerRe = regexp.MustCompile(`^\*\*([^:]+):\*\*\s*$`)
)

// ParsePlanDocument parses a markdown planning document.
func ParsePlanDocument(filePath string) (*PlanDocument, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	doc := &PlanDocument{
		FilePath: filePath,
		Sections: []Section{},
	}

	scanner := bufio.NewScanner(file)
	var currentSection *Section
	var sectionStack []*Section
	var inMetadata bool
	var lineNum int

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Extract document title from first H1
		if strings.HasPrefix(line, "# ") && doc.Title == "" {
			doc.Title = strings.TrimPrefix(line, "# ")
			inMetadata = true
			continue
		}

		// Parse metadata section
		if inMetadata {
			if matches := metadataRe.FindStringSubmatch(line); matches != nil {
				key := strings.TrimSpace(matches[1])
				value := strings.TrimSpace(matches[2])

				switch key {
				case "Document Version":
					doc.Metadata.Version = value
				case "Status":
					doc.Metadata.Status = value
				case "Date":
					doc.Metadata.Date = value
				case "Author":
					doc.Metadata.Author = value
				case "Phase":
					doc.Metadata.Phase = value
				}
				continue
			}

			// End of metadata section
			if line == "" || strings.HasPrefix(line, "##") {
				inMetadata = false
			}
		}

		// Parse phase headers
		if matches := phaseHeaderRe.FindStringSubmatch(line); matches != nil {
			phaseNum := matches[1]
			title := strings.TrimSpace(matches[2])

			level := strings.Count(line, "#")
			section := &Section{
				Title: fmt.Sprintf("Phase %s: %s", phaseNum, title),
				Level: level,
				Type:  SectionTypePhase,
				Tasks: []Task{},
			}

			// Manage section hierarchy
			for len(sectionStack) > 0 && sectionStack[len(sectionStack)-1].Level >= level {
				sectionStack = sectionStack[:len(sectionStack)-1]
			}

			if len(sectionStack) == 0 {
				doc.Sections = append(doc.Sections, *section)
				currentSection = &doc.Sections[len(doc.Sections)-1]
			} else {
				parent := sectionStack[len(sectionStack)-1]
				parent.Subsections = append(parent.Subsections, *section)
				currentSection = &parent.Subsections[len(parent.Subsections)-1]
			}

			sectionStack = append(sectionStack, currentSection)
			continue
		}

		// Parse generic headers for section tracking
		if strings.HasPrefix(line, "#") && currentSection != nil {
			level := strings.Count(line, "#")
			title := strings.TrimPrefix(line, strings.Repeat("#", level))
			title = strings.TrimSpace(title)

			// Determine section type
			sectionType := SectionTypeGeneric
			if strings.Contains(strings.ToLower(title), "task") {
				sectionType = SectionTypeTasks
			} else if strings.Contains(strings.ToLower(title), "implementation") {
				sectionType = SectionTypeImplementation
			}

			section := &Section{
				Title: title,
				Level: level,
				Type:  sectionType,
				Tasks: []Task{},
			}

			// Manage section hierarchy
			for len(sectionStack) > 0 && sectionStack[len(sectionStack)-1].Level >= level {
				sectionStack = sectionStack[:len(sectionStack)-1]
			}

			if len(sectionStack) == 0 {
				doc.Sections = append(doc.Sections, *section)
				currentSection = &doc.Sections[len(doc.Sections)-1]
			} else {
				parent := sectionStack[len(sectionStack)-1]
				parent.Subsections = append(parent.Subsections, *section)
				currentSection = &parent.Subsections[len(parent.Subsections)-1]
			}

			sectionStack = append(sectionStack, currentSection)
		}

		// Accumulate content
		if currentSection != nil {
			if currentSection.Content != "" {
				currentSection.Content += "\n"
			}
			currentSection.Content += line
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return doc, nil
}

// ExtractTasks extracts tasks from section content.
func ExtractTasks(section *Section, phaseTitle string) []Task {
	var tasks []Task
	var currentTask *Task
	var inTaskSection bool
	var inDeliverablesSection bool
	var inCriteriaSection bool

	lines := strings.Split(section.Content, "\n")
	taskOrder := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect section markers
		if matches := sectionMarkerRe.FindStringSubmatch(line); matches != nil {
			marker := strings.ToLower(matches[1])
			inTaskSection = strings.Contains(marker, "task")
			inDeliverablesSection = strings.Contains(marker, "deliverable")
			inCriteriaSection = strings.Contains(marker, "success criteria") || strings.Contains(marker, "acceptance")
			continue
		}

		// Extract numbered tasks
		if inTaskSection {
			if matches := taskListRe.FindStringSubmatch(line); matches != nil {
				// Save previous task if exists
				if currentTask != nil {
					tasks = append(tasks, *currentTask)
				}

				taskOrder++
				title := strings.TrimSpace(matches[2])
				currentTask = &Task{
					Title:        title,
					Phase:        phaseTitle,
					Priority:     2, // Default priority
					Order:        taskOrder,
					Deliverables: []string{},
					Criteria:     []string{},
					Dependencies: []string{},
				}
			}
		}

		// Extract deliverables
		if inDeliverablesSection && currentTask != nil {
			if matches := checkboxRe.FindStringSubmatch(line); matches != nil {
				deliverable := strings.TrimSpace(matches[1])
				currentTask.Deliverables = append(currentTask.Deliverables, deliverable)
			}
		}

		// Extract success criteria
		if inCriteriaSection && currentTask != nil {
			if matches := taskListRe.FindStringSubmatch(line); matches != nil {
				criterion := strings.TrimSpace(matches[2])
				currentTask.Criteria = append(currentTask.Criteria, criterion)
			} else if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "**") {
				// Continuation of criteria
				if len(currentTask.Criteria) > 0 {
					currentTask.Criteria[len(currentTask.Criteria)-1] += " " + line
				}
			}
		}
	}

	// Save last task
	if currentTask != nil {
		tasks = append(tasks, *currentTask)
	}

	return tasks
}
