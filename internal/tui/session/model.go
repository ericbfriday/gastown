package session

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/rig"
)

// TransitionPhase represents the current phase of a session transition.
type TransitionPhase string

const (
	PhaseIdle         TransitionPhase = "idle"
	PhasePreShutdown  TransitionPhase = "pre-shutdown"
	PhaseShuttingDown TransitionPhase = "shutting-down"
	PhaseShutdownHook TransitionPhase = "post-shutdown-hook"
	PhasePreStart     TransitionPhase = "pre-start"
	PhaseStarting     TransitionPhase = "starting"
	PhaseStartupHook  TransitionPhase = "post-startup-hook"
	PhaseComplete     TransitionPhase = "complete"
	PhaseError        TransitionPhase = "error"
)

// TransitionState holds state for a session transition.
type TransitionState struct {
	Phase          TransitionPhase
	Message        string
	Error          error
	Progress       float64 // 0.0 to 1.0
	StartTime      time.Time
	CurrentStep    int
	TotalSteps     int
	PreservedData  map[string]interface{} // Context preserved between sessions
	SessionName    string
	RigName        string
	PolecatName    string
	PreviousIssue  string // Issue from previous session
	NextIssue      string // Issue for next session
}

// Model is the bubbletea model for session cycling.
type Model struct {
	state       TransitionState
	spinner     spinner.Model
	progress    progress.Model
	help        help.Model
	keys        KeyMap
	width       int
	height      int
	sessionMgr  *polecat.SessionManager
	rig         *rig.Rig
	autoAdvance bool // Automatically advance through phases
}

// New creates a new session cycling TUI model.
func New(sessionMgr *polecat.SessionManager, r *rig.Rig, rigName, polecatName string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		state: TransitionState{
			Phase:       PhaseIdle,
			SessionName: fmt.Sprintf("gt-%s-%s", rigName, polecatName),
			RigName:     rigName,
			PolecatName: polecatName,
		},
		spinner:     s,
		progress:    progress.New(progress.WithDefaultGradient()),
		help:        help.New(),
		keys:        DefaultKeyMap(),
		sessionMgr:  sessionMgr,
		rig:         r,
		autoAdvance: true,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.checkCurrentState(),
	)
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 4
		m.help.Width = msg.Width
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case transitionStepMsg:
		return m.handleTransitionStep(msg)

	case transitionCompleteMsg:
		m.state.Phase = PhaseComplete
		m.state.Message = msg.message
		m.state.Progress = 1.0
		if m.autoAdvance {
			return m, tea.Quit
		}
		return m, nil

	case transitionErrorMsg:
		m.state.Phase = PhaseError
		m.state.Error = msg.err
		m.state.Message = msg.message
		return m, nil

	case currentStateMsg:
		// Initial state loaded
		m.state.PreviousIssue = msg.currentIssue
		return m, nil

	case StartCycleMsg:
		return m, m.StartCycle()

	case StartSessionMsg:
		return m, m.StartNewSession(msg.IssueID)

	case StopSessionMsg:
		return m, m.StopSession(msg.Force)
	}

	return m, nil
}

// View renders the current view.
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Current phase
	b.WriteString(m.renderPhase())
	b.WriteString("\n\n")

	// Progress bar
	if m.state.Phase != PhaseIdle && m.state.Phase != PhaseComplete && m.state.Phase != PhaseError {
		b.WriteString(m.renderProgress())
		b.WriteString("\n\n")
	}

	// Context preview
	if len(m.state.PreservedData) > 0 {
		b.WriteString(m.renderContextPreview())
		b.WriteString("\n\n")
	}

	// Status message
	if m.state.Message != "" {
		b.WriteString(m.renderMessage())
		b.WriteString("\n\n")
	}

	// Error display
	if m.state.Error != nil {
		b.WriteString(m.renderError())
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(m.renderHelp())

	return b.String()
}

// StartCycle initiates a session cycle (restart).
func (m *Model) StartCycle() tea.Cmd {
	m.state.Phase = PhasePreShutdown
	m.state.Progress = 0.0
	m.state.StartTime = time.Now()
	m.state.TotalSteps = 7 // pre-shutdown, shutdown, post-shutdown, pre-start, start, post-start, complete
	m.state.CurrentStep = 0
	return m.executePhase()
}

// StartNewSession initiates a new session (without shutdown).
func (m *Model) StartNewSession(issueID string) tea.Cmd {
	m.state.Phase = PhasePreStart
	m.state.NextIssue = issueID
	m.state.Progress = 0.0
	m.state.StartTime = time.Now()
	m.state.TotalSteps = 4 // pre-start, start, post-start, complete
	m.state.CurrentStep = 0
	return m.executePhase()
}

// StopSession initiates session shutdown.
func (m *Model) StopSession(force bool) tea.Cmd {
	m.state.Phase = PhasePreShutdown
	m.state.Progress = 0.0
	m.state.StartTime = time.Now()
	m.state.TotalSteps = 3 // pre-shutdown, shutdown, post-shutdown
	m.state.CurrentStep = 0
	if force {
		return m.forceShutdown()
	}
	return m.executePhase()
}

// executePhase executes the current transition phase.
func (m *Model) executePhase() tea.Cmd {
	return func() tea.Msg {
		switch m.state.Phase {
		case PhasePreShutdown:
			return m.doPreShutdown()
		case PhaseShuttingDown:
			return m.doShutdown()
		case PhaseShutdownHook:
			return m.doPostShutdown()
		case PhasePreStart:
			return m.doPreStart()
		case PhaseStarting:
			return m.doStart()
		case PhaseStartupHook:
			return m.doPostStart()
		default:
			return transitionCompleteMsg{message: "Transition complete"}
		}
	}
}

// checkCurrentState checks the current session state.
func (m Model) checkCurrentState() tea.Cmd {
	return func() tea.Msg {
		running, err := m.sessionMgr.IsRunning(m.state.PolecatName)
		if err != nil {
			return transitionErrorMsg{err: err, message: "Failed to check session state"}
		}

		var currentIssue string
		if running {
			// Try to determine current issue from hook
			// This would require integration with beads/hook system
			currentIssue = "" // Placeholder
		}

		return currentStateMsg{
			running:      running,
			currentIssue: currentIssue,
		}
	}
}

// handleKeyPress handles keyboard input.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(msg, m.keys.Restart):
		if m.state.Phase == PhaseIdle || m.state.Phase == PhaseComplete {
			return m, m.StartCycle()
		}
	}
	return m, nil
}

// handleTransitionStep handles transition step messages.
func (m Model) handleTransitionStep(msg transitionStepMsg) (tea.Model, tea.Cmd) {
	m.state.Phase = msg.nextPhase
	m.state.Message = msg.message
	m.state.CurrentStep++
	m.state.Progress = float64(m.state.CurrentStep) / float64(m.state.TotalSteps)

	if msg.preservedData != nil {
		if m.state.PreservedData == nil {
			m.state.PreservedData = make(map[string]interface{})
		}
		for k, v := range msg.preservedData {
			m.state.PreservedData[k] = v
		}
	}

	if m.autoAdvance {
		return m, m.executePhase()
	}
	return m, nil
}

// Phase execution functions

func (m *Model) doPreShutdown() tea.Msg {
	// Fire pre-shutdown hooks via session manager
	// This is handled internally by SessionManager.Stop
	return transitionStepMsg{
		nextPhase: PhaseShuttingDown,
		message:   "Pre-shutdown checks complete",
	}
}

func (m *Model) doShutdown() tea.Msg {
	// Capture context before shutdown
	preserved := make(map[string]interface{})

	// Capture recent output for context
	output, err := m.sessionMgr.Capture(m.state.PolecatName, 50)
	if err == nil && output != "" {
		preserved["last_output"] = output
	}

	// Stop the session
	if err := m.sessionMgr.Stop(m.state.PolecatName, false); err != nil {
		return transitionErrorMsg{
			err:     err,
			message: "Failed to stop session",
		}
	}

	// Wait for shutdown to complete
	time.Sleep(500 * time.Millisecond)

	return transitionStepMsg{
		nextPhase:     PhaseShutdownHook,
		message:       "Session stopped",
		preservedData: preserved,
	}
}

func (m *Model) doPostShutdown() tea.Msg {
	// Post-shutdown hooks are handled by SessionManager.Stop
	// Brief delay to ensure cleanup
	time.Sleep(200 * time.Millisecond)

	// If this is a full cycle, move to pre-start
	if m.state.TotalSteps > 3 {
		return transitionStepMsg{
			nextPhase: PhasePreStart,
			message:   "Ready to start new session",
		}
	}

	// Otherwise we're done
	return transitionCompleteMsg{
		message: "Session shutdown complete",
	}
}

func (m *Model) doPreStart() tea.Msg {
	// Pre-start hooks are handled by SessionManager.Start
	return transitionStepMsg{
		nextPhase: PhaseStarting,
		message:   "Pre-start checks complete",
	}
}

func (m *Model) doStart() tea.Msg {
	opts := polecat.SessionStartOptions{
		Issue: m.state.NextIssue,
	}

	if err := m.sessionMgr.Start(m.state.PolecatName, opts); err != nil {
		return transitionErrorMsg{
			err:     err,
			message: "Failed to start session",
		}
	}

	return transitionStepMsg{
		nextPhase: PhaseStartupHook,
		message:   "Session started successfully",
	}
}

func (m *Model) doPostStart() tea.Msg {
	// Post-startup hooks are handled by SessionManager.Start
	// Wait for session to be fully ready
	time.Sleep(1 * time.Second)

	return transitionCompleteMsg{
		message: "Session ready",
	}
}

func (m *Model) forceShutdown() tea.Cmd {
	return func() tea.Msg {
		if err := m.sessionMgr.Stop(m.state.PolecatName, true); err != nil {
			return transitionErrorMsg{
				err:     err,
				message: "Force shutdown failed",
			}
		}
		return transitionCompleteMsg{
			message: "Session force stopped",
		}
	}
}

// Messages

type transitionStepMsg struct {
	nextPhase     TransitionPhase
	message       string
	preservedData map[string]interface{}
}

type transitionCompleteMsg struct {
	message string
}

type transitionErrorMsg struct {
	err     error
	message string
}

type currentStateMsg struct {
	running      bool
	currentIssue string
}

// StartCycleMsg triggers a session cycle.
type StartCycleMsg struct{}

// StartSessionMsg triggers a new session start.
type StartSessionMsg struct {
	IssueID string
}

// StopSessionMsg triggers a session stop.
type StopSessionMsg struct {
	Force bool
}

// SetNextIssue sets the issue ID for the next session.
func (m *Model) SetNextIssue(issueID string) {
	m.state.NextIssue = issueID
}

// HasError returns whether the model has an error.
func (m Model) HasError() bool {
	return m.state.Error != nil
}

// GetError returns the model's error.
func (m Model) GetError() error {
	return m.state.Error
}
