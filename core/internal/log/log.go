package log

import (
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	cblog "github.com/charmbracelet/log"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

type Logger struct{ *cblog.Logger }

func (l *Logger) Printf(format string, v ...any) { l.Infof(format, v...) }

func (l *Logger) Fatalf(format string, v ...any) { l.Logger.Fatalf(format, v...) }

var (
	logger     *Logger
	initLogger sync.Once

	logMu     sync.Mutex
	logFile   *os.File
	logStderr io.Writer = os.Stderr

	ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

type ansiStripWriter struct{ w io.Writer }

func (a *ansiStripWriter) Write(p []byte) (int, error) {
	stripped := ansiRe.ReplaceAll(p, nil)
	if _, err := a.w.Write(stripped); err != nil {
		return 0, err
	}
	return len(p), nil
}

func parseLevel(level string) cblog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return cblog.DebugLevel
	case "info":
		return cblog.InfoLevel
	case "warn", "warning":
		return cblog.WarnLevel
	case "error":
		return cblog.ErrorLevel
	case "fatal":
		return cblog.FatalLevel
	default:
		return cblog.InfoLevel
	}
}

// GetQtLoggingRules returns a QT_LOGGING_RULES string that mirrors the
// configured DANKCAL_LOG_LEVEL so Qt-side logs respect the same threshold.
func GetQtLoggingRules() string {
	level := os.Getenv("DANKCAL_LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	var rules []string
	switch strings.ToLower(level) {
	case "fatal":
		rules = []string{"*.debug=false", "*.info=false", "*.warning=false", "*.critical=false"}
	case "error":
		rules = []string{"*.debug=false", "*.info=false", "*.warning=false"}
	case "warn", "warning":
		rules = []string{"*.debug=false", "*.info=false"}
	case "info":
		rules = []string{"*.debug=false"}
	case "debug":
		return ""
	default:
		rules = []string{"*.debug=false"}
	}

	return strings.Join(rules, ";")
}

func GetLogger() *Logger {
	initLogger.Do(func() {
		styles := cblog.DefaultStyles()
		// Match the colors used by the QML Log service.
		styles.Levels[cblog.FatalLevel] = lipgloss.NewStyle().
			SetString(" FATAL").
			Foreground(lipgloss.Color("1"))
		styles.Levels[cblog.ErrorLevel] = lipgloss.NewStyle().
			SetString(" ERROR").
			Foreground(lipgloss.Color("9"))
		styles.Levels[cblog.WarnLevel] = lipgloss.NewStyle().
			SetString("  WARN").
			Foreground(lipgloss.Color("3"))
		styles.Levels[cblog.InfoLevel] = lipgloss.NewStyle().
			SetString("  INFO").
			Foreground(lipgloss.Color("2"))
		styles.Levels[cblog.DebugLevel] = lipgloss.NewStyle().
			SetString(" DEBUG").
			Foreground(lipgloss.Color("4"))

		base := cblog.New(logStderr)
		base.SetStyles(styles)
		base.SetReportTimestamp(false)

		level := cblog.InfoLevel
		if envLevel := os.Getenv("DANKCAL_LOG_LEVEL"); envLevel != "" {
			level = parseLevel(envLevel)
		}
		base.SetLevel(level)
		base.SetPrefix(" go")

		logger = &Logger{base}

		if path := os.Getenv("DANKCAL_LOG_FILE"); path != "" {
			_ = SetLogFile(path)
		}
	})
	return logger
}

func SetLevel(level string) {
	GetLogger().SetLevel(parseLevel(level))
}

// SetLogFile makes the logger append to path in addition to stderr. Passing an
// empty string detaches the file sink. Atomic per-line writes (≤PIPE_BUF) on
// O_APPEND keep concurrent Go and QML writers from corrupting each other.
//
// charmbracelet/log auto-detects color support from its io.Writer, and
// io.MultiWriter doesn't pass that through, so we force the ANSI profile when
// stderr is a TTY and route the file through ansiStripWriter so the file stays
// plain while stderr keeps its colors.
func SetLogFile(path string) error {
	logMu.Lock()
	defer logMu.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}

	l := GetLogger()
	if path == "" {
		l.SetOutput(logStderr)
		applyColorProfile(l, logStderr)
		return nil
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	logFile = f
	out := io.MultiWriter(logStderr, &ansiStripWriter{w: f})
	l.SetOutput(out)
	applyColorProfile(l, logStderr)
	return nil
}

func applyColorProfile(l *Logger, stderr io.Writer) {
	f, ok := stderr.(*os.File)
	if !ok {
		l.SetColorProfile(termenv.Ascii)
		return
	}
	if isatty.IsTerminal(f.Fd()) {
		l.SetColorProfile(termenv.ANSI)
		return
	}
	l.SetColorProfile(termenv.Ascii)
}

// ApplyEnvOverrides re-reads DANKCAL_LOG_LEVEL and DANKCAL_LOG_FILE and
// reconfigures the singleton. Safe to call after CLI flags have rewritten the
// environment.
func ApplyEnvOverrides() {
	GetLogger()
	if level := os.Getenv("DANKCAL_LOG_LEVEL"); level != "" {
		SetLevel(level)
	}
	if path := os.Getenv("DANKCAL_LOG_FILE"); path != "" {
		if err := SetLogFile(path); err != nil {
			Warnf("Failed to open log file %q: %v", path, err)
		}
	}
}

func Debug(msg any, kv ...any)       { GetLogger().Logger.Debug(msg, kv...) }
func Debugf(format string, v ...any) { GetLogger().Logger.Debugf(format, v...) }
func Info(msg any, kv ...any)        { GetLogger().Logger.Info(msg, kv...) }
func Infof(format string, v ...any)  { GetLogger().Logger.Infof(format, v...) }
func Warn(msg any, kv ...any)        { GetLogger().Logger.Warn(msg, kv...) }
func Warnf(format string, v ...any)  { GetLogger().Logger.Warnf(format, v...) }
func Error(msg any, kv ...any)       { GetLogger().Logger.Error(msg, kv...) }
func Errorf(format string, v ...any) { GetLogger().Logger.Errorf(format, v...) }
func Fatal(msg any, kv ...any)       { GetLogger().Logger.Fatal(msg, kv...) }
func Fatalf(format string, v ...any) { GetLogger().Logger.Fatalf(format, v...) }
