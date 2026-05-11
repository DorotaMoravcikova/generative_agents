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
	BaseDir      string
	FileLevel    slog.Level
	ConsoleLevel slog.Level
	LogToFile    bool
}

type RunLogs struct {
	RunID  string
	RunDir string

	Log   *slog.Logger
	Sync  func()
	Close func() error
}

func NewRunLogs(cfg Config) (*RunLogs, error) {
	ts := time.Now().Format("2006-01-02_15-04-05")
	suffix, err := randomHex(4)
	if err != nil {
		return nil, err
	}
	runID := fmt.Sprintf("%s_%s", ts, suffix)

	var (
		logF   *os.File
		runDir string
	)

	if cfg.LogToFile {
		if cfg.BaseDir == "" {
			cfg.BaseDir = "logs"
		}
		runDir = filepath.Join(cfg.BaseDir, runID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return nil, err
		}
		logF, err = os.OpenFile(filepath.Join(runDir, "run.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
	}

	var hs []slog.Handler

	if logF != nil {
		hs = append(hs, slog.NewJSONHandler(logF, &slog.HandlerOptions{Level: cfg.FileLevel}))
	}
	hs = append(hs, slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.ConsoleLevel}))

	base := slog.New(NewMultiHandler(hs...)).With(slog.String("run_id", runID))
	if runDir != "" {
		base = base.With(slog.String("run_dir", runDir))
	}

	syncFn := func() {
		if logF != nil {
			_ = logF.Sync()
		}
		_ = os.Stderr.Sync()
	}

	closeFn := func() error {
		if logF != nil {
			return logF.Close()
		}
		return nil
	}

	base.Info("run_start",
		slog.String("type", "run_start"),
		slog.String("ts", time.Now().Format(time.RFC3339Nano)),
	)

	return &RunLogs{
		RunID:  runID,
		RunDir: runDir,
		Log:    base,
		Sync:   syncFn,
		Close:  closeFn,
	}, nil
}

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
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, h := range m.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r.Clone()); err != nil {
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
