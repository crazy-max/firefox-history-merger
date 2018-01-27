package utils

import (
	"github.com/crazy-max/firefox-history-merger/logger"
)

var Logger *logger.Logger

func InitLogger(debug bool) {
	Logger = logger.New()
	Logger.DisabledInfo = true
	Logger.Level = logger.LevelInfo
	if debug {
		Logger.Level = logger.LevelDebug
	}
}
