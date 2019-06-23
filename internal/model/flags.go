package model

// Flags holds flags from command line
type Flags struct {
	LogLevel          string
	LogCaller         bool
	DbFile            string
	PlacesFile        string
	PlacesToMergeFile string
	MaxPerTx          int
	FaviconsFile      string
	Workers           int
}
