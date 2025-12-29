package logger

import (
	"github.com/hashicorp/go-hclog"
)

var Log hclog.Logger

func init() {
	Log = hclog.NewNullLogger()
}

func Debug(msg string, args ...interface{}) {
	Log.Debug(msg, args...)
}

func Error(msg string, args ...interface{}) {
	Log.Error(msg, args...)
}

func Trace(msg string, args ...interface{}) {
	Log.Trace(msg, args...)
}

func Info(msg string, args ...interface{}) {
	Log.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	Log.Warn(msg, args...)
}
