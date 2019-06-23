package favicons

/**
 * moz_pages_w_icons schema >= v39
 */
type MozPagesWIcons struct {
	ID          int    `gorm:"primary_key"`
	PageUrl     string `gorm:"size:-1;not null"`
	PageUrlHash int64  `gorm:"not null;index:moz_pages_w_icons_urlhashindex"`
}

func (c *Client) PagesWIconsFirstID() (maxID int) {
	var table MozPagesWIcons
	c.Db.Model(table).First(&table)
	return table.ID
}

func (c *Client) PagesWIconsLastID() (maxID int) {
	var table MozPagesWIcons
	c.Db.Model(table).Last(&table)
	return table.ID
}
