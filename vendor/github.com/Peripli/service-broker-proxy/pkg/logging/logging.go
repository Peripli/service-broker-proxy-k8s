package logging

import (
	"github.com/onrik/logrus/filename"
	"github.com/onrik/logrus/formatter"
	"github.com/sirupsen/logrus"
	"github.com/Peripli/service-manager/pkg/log"
)

const (
	keyLogSource  = "logSource"
	logFormatJSON = "json"
)

// Setup sets up the logrus logging for the proxy based on the provided parameters.
func Setup(settings *log.Settings) {
	logrus.AddHook(&ErrorLocationHook{})
	hook := filename.NewHook()
	hook.Field = keyLogSource
	logrus.AddHook(hook)
	level, err := logrus.ParseLevel(settings.Level)
	if err != nil {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.WithError(err).Error("Could not parse log level configuration")
	} else {
		logrus.SetLevel(level)
	}
	if settings.Format == logFormatJSON {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		textFormatter := formatter.New()
		logrus.SetFormatter(textFormatter)
	}
}
