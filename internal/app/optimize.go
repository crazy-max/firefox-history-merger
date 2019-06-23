package app

import (
	"github.com/crazy-max/firefox-history-merger/internal/db"
	"github.com/rs/zerolog/log"
)

// Optimize a database into a minimal amount of disk space
func (fhm *FirefoxHistoryMerger) Optimize() {
	log.Debug().Msgf("Opening %s database...", fhm.fl.DbFile)
	link, err := db.New(fhm.fl.DbFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open database %s", fhm.fl.DbFile)
	}
	fhm.dbs = append(fhm.dbs, link)
	defer link.Close()

	log.Info().Msg("Optimizing database...")
	if err = link.Vacuum(); err != nil {
		log.Fatal().Err(err).Msg("Cannot optimize database")
	}

	log.Info().Msg("Finished")
}
