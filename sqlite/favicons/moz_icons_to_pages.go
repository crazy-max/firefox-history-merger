package favicons

/**
 * moz_icons_to_pages schema v39
 */
type MozIconsToPages struct {
	PageId int `gorm:"primary_key"`
	IconId int `gorm:"primary_key"`
}
