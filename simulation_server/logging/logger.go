package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type MultiError struct {
	errors []error
}

func (m *MultiError) Error() string {
	report := make([]string, 0, len(m.errors)+1)
	report = append(report, fmt.Sprintf("%d errors occurred", len(m.errors)))
	for _, err := range m.errors {
		report = append(report, err.Error())
	}
	return strings.Join(report, "; ")
}

type Config struct {
	BaseDir        string // e.g. "logs"
	AlsoToStderr   bool
	EnableDebugLog bool
}

type RunLogs struct {
	RunID  string
	RunDir string

	Log   *slog.Logger // use everywhere
	Sync  func()       // best-effort flush for crash paths
	Close func() error
}

// Create a run directory and configure a single logger that fans out to multiple files.
func NewRunLogs(cfg Config) (*RunLogs, error) {
	if cfg.BaseDir == "" {
		cfg.BaseDir = "logs"
	}

	ts := time.Now().Format("2006-01-02_15-04-05")
	suffix, err := randomHex(4) // 8 hex chars
	if err != nil {
		return nil, err
	}
	runID := fmt.Sprintf("%s_%s", ts, suffix)
	runDir := filepath.Join(cfg.BaseDir, runID)

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}

	eventsF, err := os.OpenFile(filepath.Join(runDir, "events.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	errorsF, err := os.OpenFile(filepath.Join(runDir, "errors.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		_ = eventsF.Close()
		return nil, err
	}

	var debugF *os.File
	if cfg.EnableDebugLog {
		debugF, err = os.OpenFile(filepath.Join(runDir, "debug.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			_ = eventsF.Close()
			_ = errorsF.Close()
			return nil, err
		}
	}

	// Handlers (levels decide what goes where)
	eventH := slog.NewJSONHandler(eventsF, &slog.HandlerOptions{Level: slog.LevelInfo})
	errorH := slog.NewJSONHandler(errorsF, &slog.HandlerOptions{Level: slog.LevelWarn})

	var hs []slog.Handler
	hs = append(hs, eventH, errorH)

	if cfg.EnableDebugLog {
		debugH := slog.NewJSONHandler(debugF, &slog.HandlerOptions{Level: slog.LevelDebug})
		hs = append(hs, debugH)
	}

	if cfg.AlsoToStderr {
		stderrH := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})
		hs = append(hs, stderrH)
	}

	mh := NewMultiHandler(hs...)
	base := slog.New(mh).With(
		slog.String("run_id", runID),
		slog.String("run_dir", runDir),
	)

	// Sync function: best-effort fsync on the files we own.
	syncFn := func() {
		_ = eventsF.Sync()
		_ = errorsF.Sync()
		if debugF != nil {
			_ = debugF.Sync()
		}
		_ = os.Stdout.Sync() // harmless best-effort
		_ = os.Stderr.Sync()
	}

	closeFn := func() error {
		var errs []error
		if err := eventsF.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := errorsF.Close(); err != nil {
			errs = append(errs, err)
		}
		if debugF != nil {
			if err := debugF.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		if errs != nil {
			return &MultiError{errs}
		}
		return nil
	}

	// Write a minimal meta record.
	base.Info("run_start",
		slog.String("type", "run_start"),
		slog.String("ts", time.Now().Format(time.RFC3339Nano)),
		slog.Bool("debug_enabled", cfg.EnableDebugLog),
	)

	return &RunLogs{
		RunID:  runID,
		RunDir: runDir,
		Log:    base,
		Sync:   syncFn,
		Close:  closeFn,
	}, nil
}

// Panic guard you put at the top of main/run entrypoint.
func RecoverAndLog(log *slog.Logger, syncFn func()) {
	if r := recover(); r != nil {
		log.Error("panic",
			slog.String("type", "panic"),
			slog.Any("panic", r),
			slog.String("stack", string(debug.Stack())),
		)
		if syncFn != nil {
			syncFn()
		}
		// Re-panic so CI/crash behavior stays the same,
		// or os.Exit(1) if you prefer.
		panic(r)
	}
}

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

/******** MultiHandler ********/

type MultiHandler struct {
	mu       sync.Mutex
	handlers []slog.Handler
}

func NewMultiHandler(h ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: h}
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	// r is a struct with references; some handlers may consume attrs,
	// so we clone per handler.
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, h := range m.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		rc := r.Clone()

		err := h.Handle(ctx, rc)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return &MultiError{errs}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: hs}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: hs}
}
