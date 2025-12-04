package logger

import (
    "encoding/json"
    "fmt"
    "os"
    "time"
)

type Level string

const (
    INFO  Level = "INFO"
    WARN  Level = "WARN"
    ERROR Level = "ERROR"
    DEBUG Level = "DEBUG"
)

type Logger struct {
    level Level
}

type LogEntry struct {
    Time    string                 `json:"time"`
    Level   string                 `json:"level"`
    Message string                 `json:"message"`
    Data    map[string]interface{} `json:"data,omitempty"`
}

func New() *Logger {
    return &Logger{level: INFO}
}

func (l *Logger) log(level Level, msg string, args ...interface{}) {
    entry := LogEntry{
        Time:    time.Now().Format(time.RFC3339),
        Level:   string(level),
        Message: msg,
    }

    if len(args) > 0 {
        entry.Data = make(map[string]interface{})
        for i := 0; i < len(args)-1; i += 2 {
            if key, ok := args[i].(string); ok {
                entry.Data[key] = args[i+1]
            }
        }
    }

    output, _ := json.Marshal(entry)
    fmt.Fprintln(os.Stdout, string(output))
}

func (l *Logger) Info(msg string, args ...interface{}) {
    l.log(INFO, msg, args...)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
    l.log(WARN, msg, args...)
}

func (l *Logger) Error(msg string, args ...interface{}) {
    l.log(ERROR, msg, args...)
}

func (l *Logger) Debug(msg string, args ...interface{}) {
    l.log(DEBUG, msg, args...)
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
    l.log(ERROR, msg, args...)
    os.Exit(1)
}
