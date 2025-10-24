package termexec

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/logctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessTerminationHandling(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	// Test with a simple command that exits quickly
	config := StartProcessConfig{
		Program:        "echo",
		Args:           []string{"hello world"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)
	defer process.Close(logger, 5*time.Second)

	// Wait for process to terminate naturally
	select {
	case <-process.TerminationChannel():
		// Process terminated
	case <-time.After(10 * time.Second):
		t.Fatal("Process did not terminate within timeout")
	}

	// Verify process is marked as terminated
	assert.True(t, process.IsTerminated())
	assert.NoError(t, process.ProcessError()) // Should be nil for normal exit

	// Verify ReadScreen still works after termination
	screen := process.ReadScreen()
	assert.Contains(t, screen, "hello world")
}

func TestProcessErrorHandling(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	// Test with a command that will fail
	config := StartProcessConfig{
		Program:        "nonexistentcommand12345",
		Args:           []string{},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)
	defer process.Close(logger, 5*time.Second)

	// Wait for process to terminate with error
	select {
	case <-process.TerminationChannel():
		// Process terminated
	case <-time.After(10 * time.Second):
		t.Fatal("Process did not terminate within timeout")
	}

	// Verify process is marked as terminated
	assert.True(t, process.IsTerminated())
	assert.Error(t, process.ProcessError()) // Should have an error
}

func TestWriteToTerminatedProcess(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "echo",
		Args:           []string{"test"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)
	defer process.Close(logger, 5*time.Second)

	// Wait for process to terminate
	select {
	case <-process.TerminationChannel():
		// Process terminated
	case <-time.After(10 * time.Second):
		t.Fatal("Process did not terminate within timeout")
	}

	// Verify we can't write to terminated process
	_, err = process.Write([]byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terminated process")
}

func TestReadScreenAfterTermination(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "echo",
		Args:           []string{"final output"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)
	defer process.Close(logger, 5*time.Second)

	// Wait for process to terminate
	select {
	case <-process.TerminationChannel():
		// Process terminated
	case <-time.After(10 * time.Second):
		t.Fatal("Process did not terminate within timeout")
	}

	// ReadScreen should work immediately after termination
	start := time.Now()
	screen := process.ReadScreen()
	duration := time.Since(start)

	// Should return immediately (not wait for 48ms)
	assert.Less(t, duration, 50*time.Millisecond)
	assert.Contains(t, screen, "final output")
}

func TestConcurrentReadScreen(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "bash",
		Args:           []string{"-c", "for i in {1..5}; do echo 'line' $i; sleep 0.1; done"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)
	defer process.Close(logger, 5*time.Second)

	// Test concurrent ReadScreen calls
	var wg sync.WaitGroup
	results := make(chan string, 10)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			screen := process.ReadScreen()
			results <- screen
		}()
	}

	wg.Wait()
	close(results)

	// All results should contain some output
	for screen := range results {
		assert.NotEmpty(t, screen)
	}
}

func TestProcessStateConsistency(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "sleep",
		Args:           []string{"1"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)
	defer process.Close(logger, 5*time.Second)

	// Initially should not be terminated
	assert.False(t, process.IsTerminated())
	assert.NoError(t, process.ProcessError())

	// Wait for termination
	select {
	case <-process.TerminationChannel():
		// Process terminated
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not terminate within timeout")
	}

	// Should now be terminated
	assert.True(t, process.IsTerminated())
	assert.NoError(t, process.ProcessError()) // Normal exit, no error
}

func TestCloseTerminatedProcess(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "echo",
		Args:           []string{"test"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)

	// Wait for process to terminate naturally
	select {
	case <-process.TerminationChannel():
		// Process terminated
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not terminate within timeout")
	}

	// Close should handle already terminated process gracefully
	err = process.Close(logger, 1*time.Second)
	assert.NoError(t, err)
}

func TestLongRunningProcessTermination(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "bash",
		Args:           []string{"-c", "while true; do echo 'running'; sleep 0.5; done"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Should not be terminated yet
	assert.False(t, process.IsTerminated())

	// Close the process
	err = process.Close(logger, 2*time.Second)
	assert.NoError(t, err)

	// Should now be terminated
	assert.True(t, process.IsTerminated())
}

func TestTerminalStateAfterError(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	// Create a process that will fail
	config := StartProcessConfig{
		Program:        "invalidcommand12345",
		Args:           []string{},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)
	defer process.Close(logger, 5*time.Second)

	// Wait for termination
	select {
	case <-process.TerminationChannel():
		// Process terminated
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not terminate within timeout")
	}

	// Should be able to read screen even after error
	screen := process.ReadScreen()
	assert.NotNil(t, screen) // Should not panic or hang
}

// Test that verifies the fix for the original issue: ReadRune errors don't freeze the PTY
func TestReadRuneErrorHandling(t *testing.T) {
	ctx := context.Background()
	logger := logctx.From(ctx)

	config := StartProcessConfig{
		Program:        "echo",
		Args:           []string{"test output"},
		TerminalWidth:  80,
		TerminalHeight: 24,
	}

	process, err := StartProcess(ctx, config)
	require.NoError(t, err)
	defer process.Close(logger, 5*time.Second)

	// Wait for process to complete
	select {
	case <-process.TerminationChannel():
		// Process terminated
	case <-time.After(5 * time.Second):
		t.Fatal("Process did not terminate within timeout")
	}

	// Verify that ReadScreen still works after the reader goroutine encountered EOF
	// This tests the core fix: that ReadRune errors don't leave the PTY frozen
	screen := process.ReadScreen()
	assert.Contains(t, screen, "test output")

	// Verify process state is consistent
	assert.True(t, process.IsTerminated())
	assert.NoError(t, process.ProcessError())
}
