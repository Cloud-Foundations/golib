package main

import (
	"bytes"
	"log"

	"github.com/Cloud-Foundations/golib/pkg/log/debuglogger"
)

type memoryLoggerType struct {
	*debuglogger.Logger
	buffer *bytes.Buffer
}

func newMemoryLogger() *memoryLoggerType {
	buffer := &bytes.Buffer{}
	return &memoryLoggerType{
		Logger: debuglogger.New(log.New(buffer, "", 0)),
		buffer: buffer,
	}
}
