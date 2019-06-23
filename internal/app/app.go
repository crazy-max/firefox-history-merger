package app

import (
	"github.com/crazy-max/firefox-history-merger/internal/db"
	"github.com/crazy-max/firefox-history-merger/internal/model"
	"github.com/rs/zerolog/log"
)

// FirefoxHistoryMerger represents an active firefox-history-merger object
type FirefoxHistoryMerger struct {
	fl  model.Flags
	dbs []*db.Client
}

// New creates new firefox-history-merger instance
func New(fl model.Flags) (*FirefoxHistoryMerger, error) {
	return &FirefoxHistoryMerger{
		fl:  fl,
		dbs: []*db.Client{},
	}, nil
}

// Close closes firefox-history-merger
func (fhm *FirefoxHistoryMerger) Close() {
	for _, dbc := range fhm.dbs {
		if err := dbc.Close(); err != nil {
			log.Warn().Err(err).Msg("Cannot close database")
		}
	}
}
