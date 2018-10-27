package cmd

import (
	"os"
	"path/filepath"

	"github.com/crazy-max/firefox-history-merger/sqlite"
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/spf13/cobra"
)

var (
	debugEnabled     bool
	appPath          string
	masterPlacesDb   sqlite.PlacesDb
	masterFaviconsDb sqlite.FaviconsDb
)

var RootCmd = &cobra.Command{
	Use:   AppName,
	Short: AppDescription,
	Long: AppDescription + `.
More info on ` + AppUrl,
}

func init() {
	appPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	cobra.OnInitialize(initRoot)
	RootCmd.PersistentFlags().BoolVarP(&debugEnabled, "debug", "x", false, "Debug")
}

func initRoot() {
	InitLogger(debugEnabled)
}
