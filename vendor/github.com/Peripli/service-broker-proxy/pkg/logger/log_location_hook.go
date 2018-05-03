package logger

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	logSourceField = "logSource"
	logrusPackage  = "github.com/sirupsen/logrus"
	prefix         = "github.com/Peripli/service-broker-proxy"
)

type LogLocationHook struct {
}

func (h *LogLocationHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *LogLocationHook) Fire(entry *logrus.Entry) error {
	pcs := make([]uintptr, 64)

	// skip 2 frames:
	//   runtime.Callers
	//   github.com/Peripli/service-broker-proxy/pkg/logger/(*LogLocationHook).Fire
	n := runtime.Callers(2, pcs)

	// re-slice pcs based on the number of entries written
	frames := runtime.CallersFrames(pcs[:n])

	// now traverse up the call stack looking for the first non-logrus
	// func, which will be the logrus invoker
	var (
		frame runtime.Frame
		more  = true
	)

	for more {
		frame, more = frames.Next()
		if strings.Contains(frame.File, logrusPackage) {
			continue
		}

		file := removePackagePrefix(frame.File)
		entry.Data[logSourceField] = fmt.Sprintf("%s:%d", file, frame.Line)
		break
	}

	return nil
}

func removePackagePrefix(file string) string {
	if index := strings.Index(file, prefix); index != -1 {
		// strip off .../github.com/Peripli/service-broker-proxy/ so we just have pkg/...
		return file[index+len(prefix):]
	}

	return file
}
