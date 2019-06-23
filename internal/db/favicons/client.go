package favicons

import (
	"fmt"

	"github.com/crazy-max/firefox-history-merger/internal/db"
	"github.com/crazy-max/firefox-history-merger/internal/utl"
	"github.com/rs/zerolog/log"
)

// Client represents an active favicons db object
type Client struct {
	Db         *db.Client
	IconsCount int
}

const (
	TbIcons = "moz_icons"
)

// New creates new favicons db instance
func New(filename string) (*Client, error) {
	var err error
	c := &Client{}

	if !utl.FileExists(filename) {
		return nil, fmt.Errorf("database not found in %s", filename)
	}

	c.Db, err = db.New(filename)
	if err != nil {
		return nil, err
	}

	if ok := c.Db.HasTable(&MozIcons{}); !ok {
		return nil, fmt.Errorf("table %s not found in database %s", TbIcons, filename)
	}
	c.Db.Model(&MozIcons{}).Count(&c.IconsCount)

	return c, nil
}

// Close closes favicons
func (c *Client) Close() {
	if err := c.Db.Close(); err != nil {
		log.Warn().Err(err).Msg("Cannot close database connection")
	}
}
