package logger

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

type Config struct {
	Level  string
	Format string
}

func Init(cfg Config) error {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	stdoutCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	ringBufCore := newRingBufferCore(2048)

	core := zapcore.NewTee(stdoutCore, ringBufCore)
	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return nil
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

func WithRequestID(requestID string) *zap.Logger {
	if requestID == "" {
		return Log
	}
	return Log.With(zap.String("request_id", requestID))
}

// --- ring buffer core ---

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Caller    string    `json:"caller,omitempty"`
}

var (
	ringMu     sync.Mutex
	ringData   []LogEntry
	ringSize   int
	ringPos    int
)

func newRingBufferCore(size int) zapcore.Core {
	ringSize = size
	ringData = make([]LogEntry, size)
	return &ringBufCore{
		LevelEnabler: zapcore.DebugLevel,
	}
}

type ringBufCore struct {
	zapcore.LevelEnabler
}

func (c *ringBufCore) Enabled(level zapcore.Level) bool {
	return c.LevelEnabler.Enabled(level)
}

func (c *ringBufCore) With(fields []zapcore.Field) zapcore.Core {
	return c
}

func (c *ringBufCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

func (c *ringBufCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	ringMu.Lock()
	ringData[ringPos] = LogEntry{
		Timestamp: entry.Time,
		Level:     entry.Level.String(),
		Message:   entry.Message,
		Caller:    entry.Caller.TrimmedPath(),
	}
	ringPos = (ringPos + 1) % ringSize
	ringMu.Unlock()
	return nil
}

func (c *ringBufCore) Sync() error { return nil }

func QueryLogs(level string, limit int) []LogEntry {
	ringMu.Lock()
	defer ringMu.Unlock()

	var result []LogEntry
	for i := 0; i < ringSize; i++ {
		idx := (ringPos - 1 - i + ringSize) % ringSize
		e := ringData[idx]
		if e.Timestamp.IsZero() {
			continue
		}
		if level != "" && level != e.Level {
			continue
		}
		result = append(result, e)
		if len(result) >= limit {
			break
		}
	}
	return result
}
