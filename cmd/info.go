package cmd

import (
	"fmt"

	"github.com/crazy-max/firefox-history-merger/sqlite"
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:     "info",
	Short:   "Display info about places.sqlite file",
	Example: AppName + ` info "places.sqlite"`,
	Args:    cobra.ExactArgs(1),
	Run:     infoRun,
}

func init() {
	RootCmd.AddCommand(infoCmd)
}

func infoRun(cmd *cobra.Command, args []string) {
	sqliteDB := sqlite.OpenFile(args[0])
	fmt.Printf("\nFilename:         %s", sqliteDB.Info.Filename)
	fmt.Printf("\nHash:             %s", sqliteDB.Info.Filehash)
	fmt.Printf("\nSchema version:   v%d (Firefox >= %d)", sqliteDB.Info.Version, sqliteDB.Info.FirefoxVersion)
	fmt.Printf("\nHistory entries:  %d", sqliteDB.Info.HistoryCount)
	fmt.Printf("\nPlaces entries:   %d", sqliteDB.Info.PlacesCount)
	fmt.Printf("\nLast used on:     %s", sqliteDB.Info.LastUsedTime.Format("2006-01-02 15:04:05"))

	if DebugEnabled {
		fmt.Printf("\n\n")
		PrintPretty(sqliteDB.Info)
	}

	sqliteDB.Link.Close()
}
