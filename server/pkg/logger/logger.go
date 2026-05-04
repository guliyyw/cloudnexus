package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger
var logDir string

type Config struct {
	Level   string
	Format  string
	Service string
	LogDir  string
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
		StacktraceKey:  "stack",
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

	ringBufCore := newRingBufferCore(2048, cfg.Service)
	cores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level),
		ringBufCore,
	}

	if cfg.LogDir != "" && cfg.Service != "" {
		logDir = cfg.LogDir
		dw := newDailyWriter(cfg.LogDir, cfg.Service)
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(dw),
			level,
		)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))

	return nil
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

func GetLogDir() string {
	return logDir
}

// FromContext returns a logger with request_id and user_id from the Gin context.
func FromContext(c *gin.Context) *zap.Logger {
	fields := []zap.Field{}
	if rid, ok := c.Get("request_id"); ok {
		if s, ok := rid.(string); ok && s != "" {
			fields = append(fields, zap.String("request_id", s))
		}
	}
	if uid, ok := c.Get("user_id"); ok {
		fields = append(fields, zap.Uint64("user_id", uid.(uint64)))
	}
	if len(fields) == 0 {
		return Log
	}
	return Log.With(fields...)
}

// --- ring buffer core ---

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Caller    string    `json:"caller,omitempty"`
	Service   string    `json:"service,omitempty"`
	Method    string    `json:"method,omitempty"`
	Path      string    `json:"path,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Stack     string    `json:"stack,omitempty"`
}

var (
	ringMu   sync.Mutex
	ringData []LogEntry
	ringSize int
	ringPos  int
)

func newRingBufferCore(size int, service string) zapcore.Core {
	ringSize = size
	ringData = make([]LogEntry, size)
	return &ringBufCore{
		LevelEnabler: zapcore.DebugLevel,
		service:      service,
	}
}

type ringBufCore struct {
	zapcore.LevelEnabler
	service string
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
	le := LogEntry{
		Timestamp: entry.Time,
		Level:     entry.Level.String(),
		Message:   entry.Message,
		Caller:    entry.Caller.TrimmedPath(),
		Service:   c.service,
		Stack:     entry.Stack,
	}

	for _, f := range fields {
		switch f.Key {
		case "request_id":
			le.RequestID = f.String
		case "user_id":
			if f.Integer >= 0 {
				le.UserID = fmt.Sprintf("%d", f.Integer)
			}
		case "method":
			le.Method = f.String
		case "path":
			le.Path = f.String
		}
	}

	ringMu.Lock()
	ringData[ringPos] = le
	ringPos = (ringPos + 1) % ringSize
	ringMu.Unlock()
	return nil
}

func (c *ringBufCore) Sync() error { return nil }

func QueryLogs(level, requestID, userID string, limit int) []LogEntry {
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
		if requestID != "" && requestID != e.RequestID {
			continue
		}
		if userID != "" && userID != e.UserID {
			continue
		}
		result = append(result, e)
		if len(result) >= limit {
			break
		}
	}
	return result
}

// --- daily rotating file writer ---

const maxFileSize = 10 * 1024 * 1024 // 10 MB per file

type dailyWriter struct {
	mu          sync.Mutex
	logDir      string
	service     string
	currentDay  string
	file        *os.File
	currentSize int64
}

func newDailyWriter(logDir, service string) *dailyWriter {
	return &dailyWriter{logDir: logDir, service: service}
}

func (w *dailyWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if today != w.currentDay || w.file == nil || w.currentSize >= maxFileSize {
		if w.file != nil {
			w.file.Close()
		}
		dir := filepath.Join(w.logDir, today)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return 0, err
		}

		base := filepath.Join(dir, w.service)
		ext := ".log"
		idx := 0
		for {
			var fname string
			if idx == 0 {
				fname = base + ext
			} else {
				fname = fmt.Sprintf("%s.%d%s", base, idx, ext)
			}
			fi, err := os.Stat(fname)
			if os.IsNotExist(err) {
				w.file, err = os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return 0, err
				}
				w.currentSize = 0
				break
			}
			if err != nil {
				return 0, err
			}
			if fi.Size() < maxFileSize {
				w.file, err = os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return 0, err
				}
				w.currentSize = fi.Size()
				break
			}
			idx++
		}
		w.currentDay = today
	}

	n, err = w.file.Write(p)
	w.currentSize += int64(n)
	return n, err
}

func (w *dailyWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Sync()
	}
	return nil
}

// --- log cleanup ---

func StartLogCleanup() {
	go func() {
		cleanOldLogs()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cleanOldLogs()
		}
	}()
}

func cleanOldLogs() {
	if logDir == "" {
		return
	}
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -30)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t, err := time.Parse("2006-01-02", entry.Name())
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			os.RemoveAll(filepath.Join(logDir, entry.Name()))
		}
	}
}

type LogFileInfo struct {
	Date string `json:"date"`
	Size int64  `json:"size"`
}

func ListLogFiles() []LogFileInfo {
	if logDir == "" {
		return []LogFileInfo{}
	}
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return []LogFileInfo{}
	}
	result := make([]LogFileInfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dateDir := filepath.Join(logDir, entry.Name())
		var totalSize int64
		filepath.Walk(dateDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			totalSize += info.Size()
			return nil
		})
		result = append(result, LogFileInfo{Date: entry.Name(), Size: totalSize})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date > result[j].Date })
	return result
}

func GetLogFilePath(date, service string) string {
	if logDir == "" {
		return ""
	}
	return filepath.Join(logDir, date, service+".log")
}

// ReadLogFile reads up to `limit` most recent log entries from the daily log file
// for the given service and date, returning entries in reverse chronological order.
func ReadLogFile(service, date string, limit int) []LogEntry {
	if logDir == "" || service == "" {
		return []LogEntry{}
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	filePath := filepath.Join(logDir, date, service+".log")
	f, err := os.Open(filePath)
	if err != nil {
		return []LogEntry{}
	}
	defer f.Close()

	var allLines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		allLines = append(allLines, sc.Text())
	}

	// Take last `limit` lines only
	if len(allLines) > limit {
		allLines = allLines[len(allLines)-limit:]
	}

	result := make([]LogEntry, 0, limit)
	for i := len(allLines) - 1; i >= 0; i-- {
		var entry LogEntry
		if err := json.Unmarshal([]byte(allLines[i]), &entry); err == nil {
			result = append(result, entry)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

// ListLogServices returns distinct service names found in today's log directory.
func ListLogServices() []string {
	if logDir == "" {
		return []string{}
	}
	today := time.Now().Format("2006-01-02")
	dir := filepath.Join(logDir, today)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	seen := make(map[string]bool)
	uniq := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		svc := strings.TrimSuffix(name, ".log")
		if idx := strings.LastIndex(svc, "."); idx != -1 {
			// Strip numbered suffix like ".1" from split files
			base := svc[:idx]
			if _, err := fmt.Sscanf(svc[idx:], ".%d", new(int)); err == nil {
				svc = base
			}
		}
		if !seen[svc] {
			seen[svc] = true
			uniq = append(uniq, svc)
		}
	}
	sort.Strings(uniq)
	return uniq
}
