package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/crazy-max/firefox-history-merger/internal/utl"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/rs/zerolog/log"
)

// Client represents an active db object
type Client struct {
	*gorm.DB
	Filename string
	Fileinfo os.FileInfo
	Filehash string
}

// New creates new db instance
func New(filename string) (*Client, error) {
	var err error
	c := &Client{}

	if c.Filename, err = filepath.Abs(filename); err != nil {
		return nil, err
	}

	if c.Fileinfo, err = os.Stat(c.Filename); err != nil {
		return nil, err
	}

	if c.Filehash, err = utl.FileHash(c.Filename); err != nil {
		return nil, err
	}

	if c.DB, err = gorm.Open("sqlite3", c.Filename); err != nil {
		return nil, err
	}
	c.DB.LogMode(false)
	c.DB.DB().SetMaxIdleConns(0)

	return c, nil
}

func (c *Client) Vacuum() (err error) {
	return c.Exec("VACUUM").Error
}

func (c *Client) Backup() error {
	fileBackup := fmt.Sprintf("%s.%s", c.Filename, time.Now().Format("20060102150405"))
	log.Debug().Msgf("Backing up %s to %s", c.Filename, filepath.Base(fileBackup))
	return utl.CopyFile(c.Filename, fileBackup)
}
