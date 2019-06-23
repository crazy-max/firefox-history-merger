package places

/**
 * moz_historyvisits schema >= v39
 */
type MozHistoryvisits struct {
	ID        int   `gorm:"primary_key"`
	FromVisit int   `gorm:"index:moz_historyvisits_fromindex"`
	PlaceId   int   `gorm:"index:moz_historyvisits_placedateindex"`
	VisitDate int64 `gorm:"index:moz_historyvisits_dateindex;index:moz_historyvisits_placedateindex"`
	VisitType int64
	Session   int64
}

func (c *Client) HistoryvisitsFirstID() (maxID int) {
	var table MozHistoryvisits
	c.Db.Model(table).First(&table)
	return table.ID
}

func (c *Client) HistoryvisitsLastID() (maxID int) {
	var table MozHistoryvisits
	c.Db.Model(table).Last(&table)
	return table.ID
}
