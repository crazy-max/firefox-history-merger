package places

import (
	"github.com/jinzhu/gorm"
)

/**
 * moz_origins schema >= v52
 */
type MozOrigins struct {
	ID       int    `gorm:"primary_key"`
	Prefix   string `gorm:"size:-1;not null"`
	Host     string `gorm:"size:-1;not null"`
	Frecency int64  `gorm:"not null"`
}

func (table MozOrigins) GetFirstID(db *gorm.DB) (maxID int) {
	db.Model(table).First(&table)
	return table.ID
}

func (table MozOrigins) GetLastID(db *gorm.DB) (maxID int) {
	db.Model(table).Last(&table)
	return table.ID
}
