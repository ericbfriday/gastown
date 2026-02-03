// Package prompt provides interactive prompts for destructive operations.
package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/style"
)

// Config controls prompt behavior
type Config struct {
	// Force skips all prompts and returns true
	Force bool
	// Yes skips all prompts and returns true
	Yes bool
	// NonInteractive causes prompts to fail instead of blocking
	NonInteractive bool
	// Timeout for automation (0 = no timeout)
	Timeout time.Duration
	// DefaultResponse when timeout expires
	DefaultResponse bool
}

// GlobalConfig is the default configuration used by package-level functions
var GlobalConfig = Config{}

// Option configures a prompt
type Option func(*Config)

// WithForce sets the force flag
func WithForce(force bool) Option {
	return func(c *Config) {
		c.Force = force
	}
}

// WithYes sets the yes flag
func WithYes(yes bool) Option {
	return func(c *Config) {
		c.Yes = yes
	}
}

// WithNonInteractive sets non-interactive mode
func WithNonInteractive(ni bool) Option {
	return func(c *Config) {
		c.NonInteractive = ni
	}
}

// WithTimeout sets a timeout for the prompt
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithDefaultResponse sets the default response when timeout expires
func WithDefaultResponse(defaultResp bool) Option {
	return func(c *Config) {
		c.DefaultResponse = defaultResp
	}
}

// Confirm prompts for yes/no confirmation.
// Returns true if user confirms, false otherwise.
// Bypassed with --force, --yes, or GT_YES environment variable.
func Confirm(message string, opts ...Option) bool {
	cfg := GlobalConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	// Check environment variable bypass
	if os.Getenv("GT_YES") == "1" {
		return true
	}

	// Check config bypasses
	if cfg.Force || cfg.Yes {
		return true
	}

	// Non-interactive mode fails by default
	if cfg.NonInteractive || !isTerminal() {
		return cfg.DefaultResponse
	}

	// Print prompt
	fmt.Printf("%s [y/N] ", message)

	// Handle timeout
	if cfg.Timeout > 0 {
		return readWithTimeout(cfg.Timeout, cfg.DefaultResponse)
	}

	// Read response
	var response string
	_, _ = fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}

// ConfirmDanger prompts for confirmation of a dangerous operation.
// Uses red color coding and requires explicit "yes" (not just "y").
func ConfirmDanger(message string, opts ...Option) bool {
	cfg := GlobalConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	// Check environment variable bypass
	if os.Getenv("GT_YES") == "1" {
		return true
	}

	// Check config bypasses
	if cfg.Force || cfg.Yes {
		return true
	}

	// Non-interactive mode fails by default
	if cfg.NonInteractive || !isTerminal() {
		return cfg.DefaultResponse
	}

	// Print danger prompt with red styling
	fmt.Printf("%s %s [yes/NO] ", style.Error.Render("⚠"), style.Bold.Render(message))

	// Handle timeout
	if cfg.Timeout > 0 {
		return readWithTimeout(cfg.Timeout, cfg.DefaultResponse)
	}

	// Read response - require full "yes"
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "yes"
}

// ConfirmBatch prompts for batch operation confirmation.
// Shows count of items affected and requires explicit confirmation.
func ConfirmBatch(operation string, count int, opts ...Option) bool {
	cfg := GlobalConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	// Check environment variable bypass
	if os.Getenv("GT_YES") == "1" {
		return true
	}

	// Check config bypasses
	if cfg.Force || cfg.Yes {
		return true
	}

	// Non-interactive mode fails by default
	if cfg.NonInteractive || !isTerminal() {
		return cfg.DefaultResponse
	}

	// Print batch prompt
	fmt.Printf("%s %d item(s)? [y/N] ", operation, count)

	// Handle timeout
	if cfg.Timeout > 0 {
		return readWithTimeout(cfg.Timeout, cfg.DefaultResponse)
	}

	// Read response
	var response string
	_, _ = fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}

// Choice prompts for a choice from multiple options.
// Returns the index of the selected option, or -1 if canceled.
func Choice(message string, options []string, opts ...Option) int {
	cfg := GlobalConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	// Non-interactive mode returns default (first option or -1)
	if cfg.NonInteractive || !isTerminal() {
		if cfg.DefaultResponse && len(options) > 0 {
			return 0
		}
		return -1
	}

	// Print prompt
	fmt.Println(style.Bold.Render(message))
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Printf("  0) Cancel\n\n")
	fmt.Printf("Choice [0-%d]: ", len(options))

	// Handle timeout
	if cfg.Timeout > 0 {
		// For choice, timeout returns -1 (cancel)
		response := readWithTimeout(cfg.Timeout, false)
		if !response {
			return -1
		}
	}

	// Read response
	var choice int
	_, err := fmt.Scanln(&choice)
	if err != nil || choice < 0 || choice > len(options) {
		return -1
	}

	if choice == 0 {
		return -1
	}

	return choice - 1
}

// Input prompts for text input with validation.
// Returns the input string and true if valid, empty string and false if canceled.
func Input(message string, validator func(string) error, opts ...Option) (string, bool) {
	cfg := GlobalConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	// Non-interactive mode fails
	if cfg.NonInteractive || !isTerminal() {
		return "", false
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		// Print prompt
		fmt.Printf("%s: ", message)

		// Handle timeout
		if cfg.Timeout > 0 {
			response := readWithTimeout(cfg.Timeout, cfg.DefaultResponse)
			if !response {
				return "", false
			}
		}

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", false
		}

		input = strings.TrimSpace(input)

		// Allow empty to cancel
		if input == "" {
			return "", false
		}

		// Validate
		if validator != nil {
			if err := validator(input); err != nil {
				fmt.Printf("%s %s\n", style.Error.Render("✗"), err)
				continue
			}
		}

		return input, true
	}
}

// isTerminal checks if stdin is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// readWithTimeout reads user input with a timeout.
// Returns defaultResponse if timeout expires.
func readWithTimeout(timeout time.Duration, defaultResponse bool) bool {
	type result struct {
		response string
		err      error
	}

	resultCh := make(chan result, 1)

	// Read in goroutine
	go func() {
		var response string
		_, err := fmt.Scanln(&response)
		resultCh <- result{response: response, err: err}
	}()

	// Wait for input or timeout
	select {
	case res := <-resultCh:
		if res.err != nil {
			return defaultResponse
		}
		response := strings.ToLower(strings.TrimSpace(res.response))
		return response == "y" || response == "yes"
	case <-time.After(timeout):
		fmt.Printf("\n%s Timeout - using default (%v)\n", style.Warning.Render("⏱"), defaultResponse)
		return defaultResponse
	}
}
