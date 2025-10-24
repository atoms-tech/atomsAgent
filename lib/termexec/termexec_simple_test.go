package termexec

import (
	"context"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/logctx"
)

// Simple test to verify the basic functionality works
func TestProcessBasicFunctionality(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "echo",
		Args:           []string{"hello"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	defer process.Close(logger, 5*time.Second)

	// Wait a bit for process to complete
	time.Sleep(100 * time.Millisecond)

	// Test basic functionality
	screen := process.ReadScreen()
	if screen == "" {
		t.Error("ReadScreen returned empty string")
	}

	// Test process state
	if process.IsTerminated() {
		t.Log("Process terminated as expected")
	} else {
		t.Log("Process still running")
	}
}

// Test that verifies the core fix: ReadRune errors don't freeze the PTY
func TestReadRuneErrorHandling(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "echo",
		Args:           []string{"test"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	defer process.Close(logger, 5*time.Second)

	// Wait for process to complete
	time.Sleep(200 * time.Millisecond)

	// This should not hang or panic - this tests the core fix
	screen := process.ReadScreen()
	if screen == "" {
		t.Error("ReadScreen returned empty string after process termination")
	}

	// Verify process state is consistent
	if !process.IsTerminated() {
		t.Error("Process should be marked as terminated")
	}
}