package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	level  string
	prefix string
	out    io.Writer
}

var std = &Logger{level: "info", prefix: "photography", out: os.Stdout}

func Init(level string) {
	std.level = level
}

func SetOutput(w io.Writer) {
	std.out = w
}

func Output() io.Writer {
	return std.out
}

func log(level, format string, args ...interface{}) {
	now := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(std.out, "[%s] %s %-5s %s\n", now, std.prefix, level, msg)
}

func Debugf(format string, args ...interface{}) {
	if std.level == "debug" {
		log("DEBUG", format, args...)
	}
}

func Infof(format string, args ...interface{}) {
	log("INFO", format, args...)
}

func Warnf(format string, args ...interface{}) {
	log("WARN", format, args...)
}

func Errorf(format string, args ...interface{}) {
	log("ERROR", format, args...)
}

// RotateFile returns a file writer under dir/name-yyyyMMdd.log
func RotateFile(dir, name string) (io.Writer, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.log", name, time.Now().Format("20060102")))
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}
