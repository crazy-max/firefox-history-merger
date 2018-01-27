// fork of https://github.com/happierall/l
package logger

import (
	"fmt"
	"os"
	"sync"
)

const (
	// LevelCrit show error and os.Exit(1)
	LevelCrit Level = iota
	// LevelError Error conditions(Ex: An application has exceeded its file storage limit and attempts to write are failing)
	LevelError
	// LevelWarning May indicate that an error will occur if action is not taken (Ex: A non-root file system has only 2GB remaining)
	LevelWarning
	// LevelInfo Normal operation events that require no action (Ex: An application has started, paused or ended successfully.
	LevelInfo
	// LevelDebug Information useful to developers for debugging an application
	LevelDebug
)

var levels = [...]string{
	"CRIT ",
	"ERROR ",
	"WARNING ",
	"INFO ",
	"DEBUG ",
}

// Level log
type Level int

func (l Level) String() string { return levels[l] }

// Default instance (global)
var Default = New()

func init() {
	Default.Depth = 4
}

// Logger instance
type Logger struct {
	mu           sync.Mutex
	Prefix       string
	DisabledInfo bool
	Depth        int
	Level        Level
}

// Debug with date and file info
func (t *Logger) Debug(v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelDebug {
		return
	}

	t.log(LevelDebug, true, v...)
}

// Log with date and file info
func (t *Logger) Log(v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelInfo {
		return
	}
	t.log(LevelInfo, true, v...)
}

// Print without date and file info
func (t *Logger) Print(v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelInfo {
		return
	}

	t.log(LevelInfo, false, v...)
}

// Warn with date and file info
func (t *Logger) Warn(v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelWarning {
		return
	}

	t.log(LevelWarning, true, v...)
}

// Error log
func (t *Logger) Error(v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelError {
		return
	}

	t.log(LevelError, true, v...)
}

// Crit log and os.Exit(1)
func (t *Logger) Crit(v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelCrit {
		return
	}

	t.log(LevelCrit, true, v...)

	os.Exit(1)
}

// Debugf format with date and file info
func (t *Logger) Debugf(format string, v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelDebug {
		return
	}

	t.log(LevelDebug, true, fmt.Sprintf(format, v...))
}

// Logf format with date and file info
func (t *Logger) Logf(format string, v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelInfo {
		return
	}
	t.log(LevelInfo, true, fmt.Sprintf(format, v...))
}

// Printf format without date and file info
func (t *Logger) Printf(format string, v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelInfo {
		return
	}

	t.log(LevelInfo, false, fmt.Sprintf(format, v...))
}

// Warnf format with date and file info
func (t *Logger) Warnf(format string, v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelWarning {
		return
	}

	t.log(LevelWarning, true, fmt.Sprintf(format, v...))
}

// Errorf format log
func (t *Logger) Errorf(format string, v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelError {
		return
	}

	t.log(LevelError, true, fmt.Sprintf(format, v...))
}

// Critf format log and os.Exit(1)
func (t *Logger) Critf(format string, v ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Level < LevelCrit {
		return
	}

	t.log(LevelCrit, true, fmt.Sprintf(format, v...))

	os.Exit(1)
}

// Debug from default instance
func Debug(v ...interface{}) {
	Default.Debug(v...)
}

// Log from default instance
func Log(v ...interface{}) {
	Default.Log(v...)
}

// Print from default instance
func Print(v ...interface{}) {
	Default.Print(v...)
}

// Warn from default instance
func Warn(v ...interface{}) {
	Default.Warn(v...)
}

// Error from default instance
func Error(v ...interface{}) {
	Default.Error(v...)
}

// Crit from default instance
func Crit(v ...interface{}) {
	Default.Crit(v...)
}

// Debugf from default instance
func Debugf(format string, v ...interface{}) {
	Default.Debugf(format, v...)
}

// Logf from default instance
func Logf(format string, v ...interface{}) {
	Default.Logf(format, v...)
}

// Printf from default instance
func Printf(format string, v ...interface{}) {
	Default.Printf(format, v...)
}

// Warnf from default instance
func Warnf(format string, v ...interface{}) {
	Default.Warnf(format, v...)
}

// Errorf from default instance
func Errorf(format string, v ...interface{}) {
	Default.Errorf(format, v...)
}

// Critf from default instance
func Critf(format string, v ...interface{}) {
	Default.Critf(format, v...)
}

func (t *Logger) log(lvl Level, enabledHeader bool, v ...interface{}) {
	out := ""

	if lvl <= LevelWarning {
		out += lvl.String()
	}

	if t.Prefix != "" {
		out += t.Prefix
	}

	formated := stringifyErrors(v)

	args := formatArgs(formated...)

	funcName, file, line, _ := getCallerInfo(t.Depth)

	if enabledHeader && !t.DisabledInfo {
		header := header(funcName, file, line)
		if header != "" {
			out = out + header + " "
		}
	}

	names, _ := argNames(file, line)

	// Convert the arguments to name=value strings.
	if len(names) == len(args) {
		args = prependArgName(names, args)
	}

	out = out + output(args...)

	if out != "" {
		fmt.Println(out)
	}
}

// New Logger
func New() *Logger {
	return &Logger{
		DisabledInfo: false,
		Depth:        3,
		Level:        LevelDebug,
	}
}
