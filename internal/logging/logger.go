package logging

import (
	"os"
	"time"

	"github.com/crazy-max/firefox-history-merger/internal/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Configure configures logger
func Configure(fl model.Flags) {
	var err error

	ctx := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC1123,
	}).With().Timestamp()

	if fl.LogCaller {
		ctx = ctx.Caller()
	}

	log.Logger = ctx.Logger()

	logLevel, err := zerolog.ParseLevel(fl.LogLevel)
	if err != nil {
		log.Fatal().Err(err).Msgf("Unknown log level")
	} else {
		zerolog.SetGlobalLevel(logLevel)
	}
}
