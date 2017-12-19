package table

import (
	"github.com/jinzhu/gorm"
)

/**
 * moz_favicons schema v39
 */
type MozFavicons struct {
	ID         int    `gorm:"primary_key"`
	Url        string `gorm:"size:-1;unique"`
	Data       []byte `gorm:"size:-1"`
	MimeType   string `gorm:"size:32"`
	Expiration int64
	Guid       string `gorm:"size:-1"`
}

func (table MozFavicons) GetFirstID(db *gorm.DB) (maxID int) {
	db.Model(table).First(&table)
	return table.ID
}

func (table MozFavicons) GetLastID(db *gorm.DB) (maxID int) {
	db.Model(table).Last(&table)
	return table.ID
}
