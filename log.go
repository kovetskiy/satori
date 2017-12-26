package main

import (
	"encoding/json"
	"fmt"

	"github.com/kovetskiy/lorg"
	"github.com/reconquest/karma-go"
)

func tracef(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	doLog(lorg.LevelTrace, context, message, args...)
}

func debugf(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	doLog(lorg.LevelDebug, context, message, args...)
}

func infof(
	context *karma.Context,
	message string,
	args ...interface{},
) {
	doLog(lorg.LevelInfo, context, message, args...)
}

func warningf(
	err error,
	message string,
	args ...interface{},
) {
	doLog(lorg.LevelWarning, err, message, args...)
}

func errorf(
	err error,
	message string,
	args ...interface{},
) {
	doLog(lorg.LevelError, err, message, args...)
}

func fatalf(
	err error,
	message string,
	args ...interface{},
) {
	doLog(lorg.LevelFatal, err, message, args...)
}

func doLog(
	level lorg.Level,
	reason interface{},
	message string,
	args ...interface{},
) {
	if logger == nil {
		return
	}

	var hierarchy karma.Karma

	switch reason := reason.(type) {
	case karma.Hierarchical:
		hierarchy = karma.Format(reason, message, args...)

	case *karma.Context:
		hierarchy = karma.Format(nil, message, args...)
		hierarchy.Context = reason

	default:
		hierarchy = karma.Format(reason, message, args...)
	}

	loggers := map[lorg.Level]func(...interface{}){
		lorg.LevelTrace:   logger.Trace,
		lorg.LevelDebug:   logger.Debug,
		lorg.LevelInfo:    logger.Info,
		lorg.LevelWarning: logger.Warning,
		lorg.LevelError:   logger.Error,
		lorg.LevelFatal:   logger.Fatal,
	}

	loggers[level](hierarchy.String())
}

func traceJSON(obj interface{}) (encoded string) {
	if !tracing {
		return ""
	}

	defer func() {
		err := recover()
		if err != nil {
			encoded = fmt.Sprintf(
				"%#v (unable to encode to json: %s)",
				obj, err,
			)
		}
	}()

	contents, err := json.MarshalIndent(obj, "", " ")
	if err != nil {
		return fmt.Sprintf(
			"%#v (unable to encode to json: %s)",
			obj, err,
		)
	}

	return string(contents)
}
