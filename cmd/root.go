package cmd

import (
	"github.com/crazy-max/firefox-history-merger/sqlite"
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/spf13/cobra"
)

var (
	debugEnabled bool
	placesDb     sqlite.PlacesDb
	faviconsDb   sqlite.FaviconsDb
)

var RootCmd = &cobra.Command{
	Use:   AppName,
	Short: AppDescription,
	Long: AppDescription + `.
More info on ` + AppUrl,
}

func init() {
	cobra.OnInitialize(initRoot)
	RootCmd.PersistentFlags().BoolVarP(&debugEnabled, "debug", "x", false, "Debug")
}

func initRoot() {
	InitLogger(debugEnabled)
}
