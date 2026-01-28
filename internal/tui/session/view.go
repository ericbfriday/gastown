package session

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Styles for the TUI
var (
	// Colors
	colorPrimary   = lipgloss.Color("205") // Pink
	colorSecondary = lipgloss.Color("135") // Purple
	colorSuccess   = lipgloss.Color("42")  // Green
	colorWarning   = lipgloss.Color("214") // Orange
	colorError     = lipgloss.Color("196") // Red
	colorMuted     = lipgloss.Color("241") // Gray
	colorInfo      = lipgloss.Color("39")  // Blue

	// Base styles
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Italic(true)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleInfo = lipgloss.NewStyle().
			Foreground(colorInfo)

	// Component styles
	stylePhase = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			MarginBottom(1)

	styleContext = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1).
			MarginBottom(1)

	styleMessage = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(colorInfo).
			PaddingLeft(2).
			MarginBottom(1)

	styleErrorBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(1, 2).
			MarginBottom(1)
)

// renderHeader renders the header section.
func (m Model) renderHeader() string {
	title := styleTitle.Render("Session Cycling")
	subtitle := styleSubtitle.Render(fmt.Sprintf("%s/%s", m.state.RigName, m.state.PolecatName))
	return title + "\n" + subtitle
}

// renderPhase renders the current transition phase.
func (m Model) renderPhase() string {
	var icon string
	var phaseStyle lipgloss.Style
	var description string

	switch m.state.Phase {
	case PhaseIdle:
		icon = "⏸"
		phaseStyle = styleMuted
		description = "Session idle - ready for operations"

	case PhasePreShutdown:
		icon = m.spinner.View()
		phaseStyle = styleWarning
		description = "Running pre-shutdown checks..."

	case PhaseShuttingDown:
		icon = m.spinner.View()
		phaseStyle = styleWarning
		description = "Stopping session and preserving context..."

	case PhaseShutdownHook:
		icon = m.spinner.View()
		phaseStyle = styleInfo
		description = "Running post-shutdown hooks..."

	case PhasePreStart:
		icon = m.spinner.View()
		phaseStyle = styleInfo
		description = "Running pre-start checks..."

	case PhaseStarting:
		icon = m.spinner.View()
		phaseStyle = styleInfo
		description = "Starting new session..."

	case PhaseStartupHook:
		icon = m.spinner.View()
		phaseStyle = styleInfo
		description = "Running post-startup hooks..."

	case PhaseComplete:
		icon = "✓"
		phaseStyle = styleSuccess
		description = "Transition complete"

	case PhaseError:
		icon = "✗"
		phaseStyle = styleError
		description = "Transition failed"
	}

	content := fmt.Sprintf("%s %s\n%s",
		icon,
		phaseStyle.Render(string(m.state.Phase)),
		styleMuted.Render(description),
	)

	if m.state.CurrentStep > 0 && m.state.TotalSteps > 0 {
		stepInfo := styleMuted.Render(
			fmt.Sprintf("Step %d/%d", m.state.CurrentStep, m.state.TotalSteps),
		)
		content += "\n" + stepInfo
	}

	return stylePhase.Render(content)
}

// renderProgress renders the progress bar.
func (m Model) renderProgress() string {
	if m.state.Progress < 0 {
		m.state.Progress = 0
	}
	if m.state.Progress > 1 {
		m.state.Progress = 1
	}

	bar := m.progress.ViewAs(m.state.Progress)

	elapsed := ""
	if !m.state.StartTime.IsZero() {
		duration := time.Since(m.state.StartTime)
		elapsed = styleMuted.Render(fmt.Sprintf("Elapsed: %s", formatDuration(duration)))
	}

	return bar + "\n" + elapsed
}

// renderContextPreview renders preserved context data.
func (m Model) renderContextPreview() string {
	var lines []string
	lines = append(lines, styleInfo.Render("Preserved Context:"))

	if m.state.PreviousIssue != "" {
		lines = append(lines, styleMuted.Render(fmt.Sprintf("  Previous issue: %s", m.state.PreviousIssue)))
	}

	if m.state.NextIssue != "" {
		lines = append(lines, styleMuted.Render(fmt.Sprintf("  Next issue: %s", m.state.NextIssue)))
	}

	if output, ok := m.state.PreservedData["last_output"].(string); ok && output != "" {
		preview := truncateString(output, 100)
		lines = append(lines, styleMuted.Render(fmt.Sprintf("  Last output: %s", preview)))
	}

	for k, v := range m.state.PreservedData {
		if k != "last_output" {
			lines = append(lines, styleMuted.Render(fmt.Sprintf("  %s: %v", k, v)))
		}
	}

	return styleContext.Render(strings.Join(lines, "\n"))
}

// renderMessage renders status messages.
func (m Model) renderMessage() string {
	return styleMessage.Render(m.state.Message)
}

// renderError renders error information.
func (m Model) renderError() string {
	content := styleError.Render("Error:") + "\n"
	content += m.state.Error.Error()

	if m.state.Message != "" {
		content += "\n\n" + styleMuted.Render(m.state.Message)
	}

	return styleErrorBox.Render(content)
}

// renderHelp renders help text.
func (m Model) renderHelp() string {
	if m.help.ShowAll {
		return m.help.View(m.keys)
	}
	return styleMuted.Render(m.help.ShortHelpView(m.keys.ShortHelp()))
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", mins, secs)
}

func truncateString(s string, maxLen int) string {
	// Remove newlines
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// renderTransitionSummary renders a summary view after transition completes.
func (m Model) renderTransitionSummary() string {
	var lines []string

	lines = append(lines, styleSuccess.Render("✓ Transition Complete"))
	lines = append(lines, "")

	if !m.state.StartTime.IsZero() {
		duration := time.Since(m.state.StartTime)
		lines = append(lines, styleMuted.Render(fmt.Sprintf("Total time: %s", formatDuration(duration))))
	}

	if m.state.PreviousIssue != "" && m.state.NextIssue != "" {
		lines = append(lines, "")
		lines = append(lines, styleMuted.Render("Transition:"))
		lines = append(lines, styleMuted.Render(fmt.Sprintf("  %s → %s", m.state.PreviousIssue, m.state.NextIssue)))
	}

	if len(m.state.PreservedData) > 0 {
		lines = append(lines, "")
		lines = append(lines, styleMuted.Render(fmt.Sprintf("Preserved %d context items", len(m.state.PreservedData))))
	}

	return stylePhase.Render(strings.Join(lines, "\n"))
}
