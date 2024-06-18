package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// Define a Level type to represent the severity level for a log entry.
type Level int8

const (
	LevelInfo Level = iota	// Has value 0.
	LevelError							// Has value 1.
	LevelFatal							// Has value 2.
	LevelOff								// Has value 3.
)

// Returns string representation for the severity level.
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Custom Logger type that holds the output destination that the log 
// entries will be written to, the minimum severity level that log entries will be written for,
// mutex for coordination the writes.
type Logger struct {
	out				io.Writer
	minLevel	Level
	mu				sync.Mutex
}

// Return a new Logger instance which writes log entries at or above a minumum severity
// level to a specific output destination.
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out: out,
		minLevel: minLevel,
	}
}

func (l *Logger) PrintInfo(message string, props map[string]string) {
	l.print(LevelInfo, message, props)
}

func (l *Logger) PrintError(err error, props map[string]string) {
	l.print(LevelError, err.Error(), props)
}

func (l *Logger) PrintFatal(err error, props map[string]string) {
	l.print(LevelFatal, err.Error(), props)
	os.Exit(1) // For entries at the FATAL level, we terminate the app.
}


func (l *Logger) print(level Level, message string, props map[string]string) (int, error) {
	// If sev level of the log entry is below the min sev for the logger, return with no action.
	if level < l.minLevel {
		return 0, nil
	}

	// Define an anonymous struct holding the data for the log entry.
	aux := struct {
		Level				string					`json:"level"`
		Time				string					`json:"time"`
		Message			string					`json:"message"`
		Properties  map[string]string `json:"properties,omitempty"`
		Trace				string					`json:"trace,omitempty"`
	}{
		Level:		level.String(),
		Time:			time.Now().UTC().Format(time.RFC3339),
		Message:  message,
		Properties: props,
	}

	// Include a stack trace for entries at the ERROR and FATAL level.
	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	// Declare a line variable for holding the actual log entry text.
	var line []byte

	// Encode the anonymous struct to JSON and store in the 'line' variable.
	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log message:" + err.Error())
	}

	// Lock the mutex so that no two writes to the output destination cannot happen concurrently.
	l.mu.Lock()

	defer l.mu.Unlock()

	return l.out.Write(append(line, '\n'))
}


// Implement Write() method on the Logger type so it satisfies the io.Writer interface.
func (l *Logger) Write(message []byte) (n int, err error) {
	return l.print(LevelError, string(message), nil)
}



