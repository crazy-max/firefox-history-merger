package table

import (
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/jinzhu/gorm"
)

/**
 * moz_places schema v39
 */
type MozPlaces struct {
	ID              int    `gorm:"primary_key"`
	Url             string `gorm:"size:-1"`
	Title           string `gorm:"size:-1"`
	RevHost         string `gorm:"size:-1;index:moz_places_hostindex"`
	VisitCount      int64  `gorm:"default:0;index:moz_places_visitcount"`
	Hidden          int64  `gorm:"not null;default:0"`
	Typed           int64  `gorm:"not null;default:0"`
	FaviconId       int    `gorm:"index:moz_places_faviconindex"`
	Frecency        int64  `gorm:"not null;default:-1;index:moz_places_frecencyindex"`
	LastVisitDate   int64  `gorm:"index:moz_places_lastvisitdateindex"`
	Guid            string `gorm:"size:-1;unique_index:moz_places_guid_uniqueindex"`
	ForeignCount    int64  `gorm:"not null;default:0"`
	UrlHash         int64  `gorm:"not null;default:0;index:moz_places_url_hashindex"`
	Description     string `gorm:"size:-1"`
	PreviewImageUrl string `gorm:"size:-1"`
}

func (table MozPlaces) GetFirstID(db *gorm.DB) (maxID int) {
	db.Model(table).First(&table)
	return table.ID
}

func (table MozPlaces) GetLastID(db *gorm.DB) (maxID int) {
	db.Model(table).Last(&table)
	return table.ID
}

func (table MozPlaces) GenerateGUID(db *gorm.DB) string {
	var found int
	guid := GenerateGUID()
	db.Model(table).Where("guid = ?", guid).Count(&found)
	if found > 0 {
		guid = table.GenerateGUID(db)
	}
	return guid
}
