package teelogger

import (
	"fmt"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/Cloud-Foundations/golib/pkg/log/debuglogger"
)

type flusher interface {
	Flush() error
}

type Logger struct {
	one log.DebugLogger
	two log.DebugLogger
}

// New creates a Logger which will send each log message to the logger specified
// by one and then to the logger specified by two.
// Debug log message will be sent to logger which implement a DebugLogger.
func New(one, two log.Logger) *Logger {
	return &Logger{debuglogger.Upgrade(one), debuglogger.Upgrade(two)}
}

func (l *Logger) Debug(level uint8, v ...interface{}) {
	l.one.Debug(level, v...)
	l.two.Debug(level, v...)
}

func (l *Logger) Debugf(level uint8, format string, v ...interface{}) {
	l.one.Debugf(level, format, v...)
	l.two.Debugf(level, format, v...)
}

func (l *Logger) Debugln(level uint8, v ...interface{}) {
	l.one.Debugln(level, v...)
	l.two.Debugln(level, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.one.Print(msg)
	if fl, ok := l.one.(flusher); ok {
		fl.Flush()
	}
	l.two.Fatal(msg)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.one.Print(msg)
	if fl, ok := l.one.(flusher); ok {
		fl.Flush()
	}
	l.two.Fatal(msg)
}

func (l *Logger) Fatalln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	l.one.Print(msg)
	if fl, ok := l.one.(flusher); ok {
		fl.Flush()
	}
	l.two.Fatal(msg)
}

func (l *Logger) Panic(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.one.Print(msg)
	l.two.Panic(msg)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.one.Print(msg)
	l.two.Panic(msg)
}

func (l *Logger) Panicln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	l.one.Print(msg)
	l.two.Panic(msg)
}

func (l *Logger) Print(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.one.Print(msg)
	l.two.Print(msg)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.one.Print(msg)
	l.two.Print(msg)
}

func (l *Logger) Println(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	l.one.Print(msg)
	l.two.Print(msg)
}
