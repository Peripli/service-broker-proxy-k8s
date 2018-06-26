package logger

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	errorSourceField = "errorSource"
)

type ErrorLocationHook struct {
}

func (h *ErrorLocationHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *ErrorLocationHook) Fire(entry *logrus.Entry) error {
	var (
		errObj interface{}
		exists bool
	)

	if errObj, exists = entry.Data[logrus.ErrorKey]; !exists {
		return nil
	}

	err, ok := errObj.(error)
	if !ok {
		return errors.New("object logged as error does not satisfy error interface")
	}

	stackErr := getInnermostTrace(err)

	if stackErr != nil {
		stackTrace := stackErr.StackTrace()
		errSource := fmt.Sprintf("%s:%n:%d", stackTrace[0], stackTrace[0], stackTrace[0])

		entry.Data[errorSourceField] = errSource
	}

	return nil
}

type stackTracer interface {
	error
	StackTrace() errors.StackTrace
}

type causer interface {
	Cause() error
}

func getInnermostTrace(err error) stackTracer {
	var tracer stackTracer

	for {
		t, isTracer := err.(stackTracer)
		if isTracer {
			tracer = t
		}

		c, isCauser := err.(causer)
		if isCauser {
			err = c.Cause()
		} else {
			return tracer
		}
	}
}
