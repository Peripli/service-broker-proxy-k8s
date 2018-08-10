package logging

import (
	"github.com/onrik/logrus/filename"
	"github.com/onrik/logrus/formatter"
	"github.com/sirupsen/logrus"
)

const (
	keyLogSource  = "logSource"
	logFormatJSON = "json"
)

// Setup sets up the logrus logging for the proxy based on the provided parameters.
func Setup(logLevel string, logFormat string) {
	logrus.AddHook(&ErrorLocationHook{})
	hook := filename.NewHook()
	hook.Field = keyLogSource
	logrus.AddHook(hook)
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.WithError(err).Error("Could not parse log level configuration")
	} else {
		logrus.SetLevel(level)
	}
	if logFormat == logFormatJSON {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		textFormatter := formatter.New()
		logrus.SetFormatter(textFormatter)
	}
}
