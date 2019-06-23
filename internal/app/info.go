package app

import (
	"github.com/crazy-max/firefox-history-merger/internal/db/places"
	"github.com/rs/zerolog/log"
)

// Info displays info about places database
func (fhm *FirefoxHistoryMerger) Info() {
	log.Debug().Msgf("Opening %s places database...", fhm.fl.PlacesFile)
	pCli, err := places.New(fhm.fl.PlacesFile, false)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open database %s", fhm.fl.PlacesFile)
	}

	log.Info().Msgf("Schema version:         v%d (Firefox >= %d)", pCli.DbVersion, pCli.FirefoxVersion)
	log.Info().Msgf("Compatible:             %t", pCli.Compatible())
	log.Info().Msgf("Places entries:         %d", pCli.PlacesCount)
	log.Info().Msgf("Historyvisits entries:  %d", pCli.HistoryvisitsCount)
	log.Info().Msgf("Last used on:           %s", pCli.LastUsed.Format("2006-01-02 15:04:05"))
}
