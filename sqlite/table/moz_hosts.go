package table

import "github.com/jinzhu/gorm"

/**
 * moz_hosts schema v39
 */
type MozHosts struct {
	ID       int    `gorm:"primary_key"`
	Host     string `gorm:"size:-1;not null;unique"`
	Frecency int64
	Typed    string `gorm:"size:-1;not null;default:0"`
	Prefix   string `gorm:"size:-1"`
}

func (table MozHosts) GetFirstID(db *gorm.DB) (maxID int) {
	db.Model(table).First(&table)
	return table.ID
}

func (table MozHosts) GetLastID(db *gorm.DB) (maxID int) {
	db.Model(table).Last(&table)
	return table.ID
}
