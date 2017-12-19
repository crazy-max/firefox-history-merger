package table

import (
	"github.com/jinzhu/gorm"
)

/**
 * moz_historyvisits schema v39
 */
type MozHistoryvisits struct {
	ID        int   `gorm:"primary_key"`
	FromVisit int   `gorm:"index:moz_historyvisits_fromindex"`
	PlaceId   int   `gorm:"index:moz_historyvisits_placedateindex"`
	VisitDate int64 `gorm:"index:moz_historyvisits_dateindex;index:moz_historyvisits_placedateindex"`
	VisitType int64
	Session   int64
}

func (table MozHistoryvisits) GetFirstID(db *gorm.DB) (maxID int) {
	db.Model(table).First(&table)
	return table.ID
}

func (table MozHistoryvisits) GetLastID(db *gorm.DB) (maxID int) {
	db.Model(table).Last(&table)
	return table.ID
}
