package places

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/crazy-max/firefox-history-merger/internal/db"
	"github.com/crazy-max/firefox-history-merger/internal/utl"
	"github.com/rs/zerolog/log"
)

// Client represents an active places db object
type Client struct {
	Db                 *db.Client
	DbVersion          int
	FirefoxVersion     int
	PlacesCount        int
	HistoryvisitsCount int
	LastUsed           time.Time
}

const (
	MinSchemaVersion = 39
	TbPlaces         = "moz_places"
	TbHistoryVisits  = "moz_history_visits"
)

// New creates new places db instance
func New(filename string, checkcompat bool) (*Client, error) {
	var err error
	c := &Client{}

	if !utl.FileExists(filename) {
		return nil, fmt.Errorf("database not found in %s", filename)
	}

	c.Db, err = db.New(filename)
	if err != nil {
		return nil, err
	}

	if c.DbVersion, err = c.dbVersion(); err != nil {
		return nil, err
	}

	c.FirefoxVersion = c.firefoxVersion()

	if checkcompat && !c.Compatible() {
		return nil, fmt.Errorf("Schema v%d (Firefox %d) is not compatible (v%d or higher required). ",
			c.DbVersion,
			c.FirefoxVersion,
			MinSchemaVersion,
		)
	}

	if ok := c.Db.HasTable(&MozPlaces{}); !ok {
		return nil, fmt.Errorf("table %s not found in database %s", TbPlaces, filename)
	}
	c.Db.Model(&MozPlaces{}).Count(&c.PlacesCount)

	if ok := c.Db.HasTable(&MozHistoryvisits{}); !ok {
		return nil, fmt.Errorf("table %s not found in database %s", TbHistoryVisits, filename)
	}
	if err := c.Db.Model(&MozHistoryvisits{}).Select("count(id)").Row().Scan(&c.HistoryvisitsCount); err != nil {
		return nil, err
	}

	var lastUsed int64
	if err := c.Db.Model(&MozPlaces{}).Select("last_visit_date").Order("last_visit_date desc").Row().Scan(&lastUsed); err != nil {
		return nil, err
	}
	c.LastUsed = time.Unix(int64(math.Ceil(float64(lastUsed/1000000))), 0)

	return c, nil
}

// Compatible checks if the schema version is compatible with the app
func (c *Client) Compatible() bool {
	return c.DbVersion >= MinSchemaVersion
}

// Close closes favicons
func (c *Client) Close() {
	if err := c.Db.Close(); err != nil {
		log.Warn().Err(err).Msg("Cannot close database connection")
	}
}

// https://dxr.mozilla.org/mozilla-central/source/toolkit/modules/Sqlite.jsm#1175
func (c *Client) dbVersion() (int, error) {
	row := c.Db.Raw("PRAGMA user_version").Row()
	if row == nil {
		return 0, errors.New("PRAGMA user_version not found")
	}

	var version int
	if err := row.Scan(&version); err != nil {
		return 0, err
	}

	return version, nil
}

// https://dxr.mozilla.org/mozilla-central/source/toolkit/components/places/Database.cpp#974
func (c *Client) firefoxVersion() int {
	if c.DbVersion < 11 {
		return 3
	}
	if c.DbVersion < 12 {
		return 4
	}
	if c.DbVersion < 13 {
		return 8
	}
	if c.DbVersion < 18 {
		return 12
	}
	if c.DbVersion < 20 {
		return 13
	}
	if c.DbVersion < 22 {
		return 14
	}
	if c.DbVersion < 23 {
		return 22
	}
	if c.DbVersion < 24 {
		return 24
	}
	if c.DbVersion < 25 {
		return 34
	}
	if c.DbVersion < 26 {
		return 36
	}
	if c.DbVersion < 27 {
		return 37
	}
	if c.DbVersion < 30 {
		return 39
	}
	if c.DbVersion < 31 {
		return 41
	}
	if c.DbVersion < 32 {
		return 48
	}
	if c.DbVersion < 33 {
		return 49
	}
	if c.DbVersion < 34 {
		return 50
	}
	if c.DbVersion < 35 {
		return 51
	}
	if c.DbVersion < 37 {
		return 52
	}
	if c.DbVersion < 38 {
		return 55
	}
	if c.DbVersion < 39 {
		return 56
	}
	if c.DbVersion < 41 {
		return 57
	}
	if c.DbVersion < 42 {
		return 58
	}
	if c.DbVersion < 43 {
		return 59
	}
	if c.DbVersion < 47 {
		return 60
	}
	if c.DbVersion < 52 {
		return 61
	}
	if c.DbVersion < 53 {
		return 62
	}
	return 74
}
