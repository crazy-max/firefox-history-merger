package places

/**
 * moz_places schema >= v52
 */
type MozPlaces struct {
	ID              int    `gorm:"primary_key"`
	Url             string `gorm:"size:-1"`
	Title           string `gorm:"size:-1"`
	RevHost         string `gorm:"size:-1;index:moz_places_hostindex"`
	VisitCount      int64  `gorm:"default:0;index:moz_places_visitcount"`
	Hidden          int64  `gorm:"not null;default:0"`
	Typed           int64  `gorm:"not null;default:0"`
	Frecency        int64  `gorm:"not null;default:-1;index:moz_places_frecencyindex"`
	LastVisitDate   int64  `gorm:"index:moz_places_lastvisitdateindex"`
	Guid            string `gorm:"size:-1;unique_index:moz_places_guid_uniqueindex"`
	ForeignCount    int64  `gorm:"not null;default:0"`
	UrlHash         int64  `gorm:"not null;default:0;index:moz_places_url_hashindex"`
	Description     string `gorm:"size:-1"`
	PreviewImageUrl string `gorm:"size:-1"`
	OriginId        int    `gorm:"index:moz_places_originidindex"`
}

// PlacesFirstID returns first moz_places ID
func (c *Client) PlacesFirstID() (maxID int) {
	var table MozPlaces
	c.Db.Model(table).First(&table)
	return table.ID
}

// PlacesLastID returns last moz_places ID
func (c *Client) PlacesLastID() (maxID int) {
	var table MozPlaces
	c.Db.Model(table).Last(&table)
	return table.ID
}
