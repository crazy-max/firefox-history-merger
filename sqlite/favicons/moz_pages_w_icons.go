package favicons

import (
	"github.com/jinzhu/gorm"
)

/**
 * moz_pages_w_icons schema >= v39
 */
type MozPagesWIcons struct {
	ID          int    `gorm:"primary_key"`
	PageUrl     string `gorm:"size:-1;not null"`
	PageUrlHash int64  `gorm:"not null;index:moz_pages_w_icons_urlhashindex"`
}

func (table MozPagesWIcons) GetFirstID(db *gorm.DB) (maxID int) {
	db.Model(table).First(&table)
	return table.ID
}

func (table MozPagesWIcons) GetLastID(db *gorm.DB) (maxID int) {
	db.Model(table).Last(&table)
	return table.ID
}
