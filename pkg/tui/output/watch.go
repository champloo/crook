package output

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/term"
)

// MinWatchInterval is the minimum allowed watch interval to prevent excessive CPU usage.
const MinWatchInterval = 100 * time.Millisecond

// WatchOptions configures watch mode behavior
type WatchOptions struct {
	// Interval is the refresh interval
	Interval time.Duration

	// Format is the output format
	Format Format

	// FetchFunc fetches the data
	FetchFunc func(ctx context.Context) (*Data, error)

	// Writer is where to write output
	Writer io.Writer

	// Command is the command string to display in header
	Command string
}

// RunWatch runs the watch loop, refreshing output at the specified interval.
// Returns an error if opts.Interval is <= 0.
func RunWatch(ctx context.Context, opts WatchOptions) error {
	// Validate interval to prevent ticker panic
	if opts.Interval <= 0 {
		return fmt.Errorf("watch interval must be positive, got %v", opts.Interval)
	}
	if opts.Interval < MinWatchInterval {
		return fmt.Errorf("watch interval must be at least %v, got %v", MinWatchInterval, opts.Interval)
	}

	// Set up signal handling
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
			// Context cancelled, exit goroutine cleanly
		}
	}()

	// Get terminal info
	isTermOut := isTerminalWriter(opts.Writer)

	// Initial render
	if err := renderWatchIteration(ctx, opts, isTermOut); err != nil {
		return err
	}

	// Set up ticker
	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Clean exit
			if isTermOut {
				// Print newline so next command starts on fresh line
				_, _ = fmt.Fprintln(opts.Writer)
			}
			return nil

		case <-ticker.C:
			if err := renderWatchIteration(ctx, opts, isTermOut); err != nil {
				// Log error but continue
				if isTermOut {
					_, _ = fmt.Fprintf(opts.Writer, "\nError: %v\n", err)
				}
			}
		}
	}
}

// renderWatchIteration performs a single watch iteration
func renderWatchIteration(ctx context.Context, opts WatchOptions, isTerm bool) error {
	// Clear screen if terminal
	if isTerm {
		clearScreen(opts.Writer)
	}

	// Print header
	printWatchHeader(opts.Writer, opts.Interval, opts.Command)

	// Fetch data
	data, fetchErr := opts.FetchFunc(ctx)
	if fetchErr != nil {
		return fmt.Errorf("failed to fetch data: %w", fetchErr)
	}

	// Render output
	if renderErr := Render(opts.Writer, data, opts.Format); renderErr != nil {
		return fmt.Errorf("failed to render: %w", renderErr)
	}

	return nil
}

// printWatchHeader prints the watch header similar to the Unix watch command
func printWatchHeader(w io.Writer, interval time.Duration, command string) {
	now := time.Now()
	timeStr := now.Format("Mon Jan 2 15:04:05 2006")

	// Format: "Every 2.0s: crook ls    Sun Jan 4 12:00:00 2026"
	header := fmt.Sprintf("Every %.1fs: %s    %s\n\n",
		interval.Seconds(),
		command,
		timeStr,
	)

	_, _ = fmt.Fprint(w, header)
}

// clearScreen clears the terminal screen
func clearScreen(w io.Writer) {
	// ANSI escape codes
	// \033[H - move cursor to home position (top-left)
	// \033[2J - clear entire screen
	_, _ = fmt.Fprint(w, "\033[H\033[2J")
}

// isTerminalWriter checks if the writer is a terminal
func isTerminalWriter(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// WatchRunner is a wrapper that manages watch mode execution
type WatchRunner struct {
	opts    WatchOptions
	running atomic.Bool
}

// NewWatchRunner creates a new watch runner
func NewWatchRunner(opts WatchOptions) *WatchRunner {
	return &WatchRunner{
		opts: opts,
	}
}

// Run starts the watch loop
func (wr *WatchRunner) Run(ctx context.Context) error {
	if !wr.running.CompareAndSwap(false, true) {
		return fmt.Errorf("watch already running")
	}
	defer wr.running.Store(false)

	return RunWatch(ctx, wr.opts)
}

// IsRunning returns true if the watch loop is running
func (wr *WatchRunner) IsRunning() bool {
	return wr.running.Load()
}
