package beads

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

// Executor defines the interface for executing bd commands.
type Executor interface {
	Execute(ctx context.Context, workDir string, args ...string) ([]byte, error)
}

// DefaultExecutor implements Executor by shelling out to the bd binary.
type DefaultExecutor struct{}

// Execute runs bd with the given arguments and returns stdout.
func (e *DefaultExecutor) Execute(ctx context.Context, workDir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "bd", args...)

	if workDir != "" {
		cmd.Dir = workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check for specific error conditions
		stderrStr := stderr.String()

		if strings.Contains(stderrStr, "not initialized") ||
			strings.Contains(stderrStr, "no .beads") ||
			strings.Contains(stderrStr, "no beads database") ||
			strings.Contains(stderrStr, "no active beads workspace") {
			return nil, &NotInitializedError{Message: stderrStr}
		}

		if strings.Contains(stderrStr, "not found") ||
			strings.Contains(stderrStr, "does not exist") {
			return nil, &NotFoundError{ID: extractIDFromArgs(args)}
		}

		// Check if bd itself is not found
		if execErr, ok := err.(*exec.Error); ok {
			if execErr.Err == exec.ErrNotFound {
				return nil, &BDNotFoundError{}
			}
		}

		return nil, &ExecutionError{
			Command: strings.Join(args, " "),
			Stderr:  stderrStr,
			Err:     err,
		}
	}

	return stdout.Bytes(), nil
}

// extractIDFromArgs attempts to extract an issue ID from command args.
func extractIDFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "show" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// MockExecutor is a test double for Executor.
type MockExecutor struct {
	Responses map[string][]byte
	Errors    map[string]error
}

// NewMockExecutor creates a mock executor for testing.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
}

// Execute returns pre-configured responses based on the first argument.
func (m *MockExecutor) Execute(ctx context.Context, workDir string, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")

	if err, ok := m.Errors[key]; ok {
		return nil, err
	}

	if resp, ok := m.Responses[key]; ok {
		return resp, nil
	}

	// Try matching just the command
	if len(args) > 0 {
		if err, ok := m.Errors[args[0]]; ok {
			return nil, err
		}
		if resp, ok := m.Responses[args[0]]; ok {
			return resp, nil
		}
	}

	return nil, &ExecutionError{Command: key, Stderr: "mock: no response configured"}
}

// SetResponse sets a mock response for a command pattern.
func (m *MockExecutor) SetResponse(pattern string, response []byte) {
	m.Responses[pattern] = response
}

// SetError sets a mock error for a command pattern.
func (m *MockExecutor) SetError(pattern string, err error) {
	m.Errors[pattern] = err
}
