package logwatcher

import (
	"io"
	"fmt"
)

// Simple logger interface to avoid importing another dependency.

type SimpleLogger interface {
	Errorf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
}

type WriterLogger struct {
	s io.Writer
}

func (wl *WriterLogger) Errorf(s string, v ...interface{}) {
	wl.lfprintf("ERROR", s, v...)
}
func (wl *WriterLogger) Infof(s string, v ...interface{}) {
	wl.lfprintf("INFO", s, v...)
}
func (wl *WriterLogger) Debugf(s string, v ...interface{}) {
	wl.lfprintf("DEBUG", s, v...)
}

func (wl *WriterLogger) lfprintf(lvl, template string, v ...interface{}) {
	fmt.Fprintf(wl.s, "[" + lvl + "] " + template + "\n", v...)
}
