package favicons

/**
 * moz_icons schema >= v39
 */
type MozIcons struct {
	ID               int    `gorm:"primary_key"`
	IconUrl          string `gorm:"size:-1;not null"`
	FixedIconUrlHash int64  `gorm:"not null;index:moz_icons_iconurlhashindex"`
	Width            int64  `gorm:"not null;default:0"`
	Root             int64  `gorm:"not null;default:0"`
	Color            int64
	ExpireMs         int64  `gorm:"not null;default:0"`
	Data             []byte `gorm:"size:-1"`
}

func (c *Client) IconsFirstID() (maxID int) {
	var table MozIcons
	c.Db.Model(table).First(&table)
	return table.ID
}

func (c *Client) IconsLastID() (maxID int) {
	var table MozIcons
	c.Db.Model(table).Last(&table)
	return table.ID
}
