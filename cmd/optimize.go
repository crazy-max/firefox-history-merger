package cmd

import (
	"github.com/crazy-max/firefox-history-merger/sqlite"
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/spf13/cobra"
)

var (
	optimizeCmd = &cobra.Command{
		Use:     "optimize",
		Short:   "Optimize a database into a minimal amount of disk space",
		Example: AppName + ` optimize "/home/user/places.sqlite"`,
		Run:     optimizeRun,
	}
)

func init() {
	RootCmd.AddCommand(optimizeCmd)
}

func optimizeRun(cmd *cobra.Command, args []string) {
	var err error

	// check args
	if len(args) < 1 {
		Logger.Crit("info requires at least one sqlite file")
	}
	if len(args) > 2 {
		Logger.Crit("has too many arguments")
	}
	if !FileExists(args[0]) {
		Logger.Critf("%s not found", args[0])
	}

	// optimizing
	Logger.Print("Optimizing DB...")
	db, err := sqlite.Open(args[0])
	if err != nil {
		Logger.Crit(err)
	}
	if err = sqlite.Vacuum(db); err != nil {
		Logger.Warn(err)
	} else {
		Logger.Print("Done!")
	}

	db.Close()
}
