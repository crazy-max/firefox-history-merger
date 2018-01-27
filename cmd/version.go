package cmd

import (
	"fmt"

	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show " + AppName + " version",
	Run:   versionRun,
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

func versionRun(cmd *cobra.Command, args []string) {
	fmt.Printf("%s %s\n%s\n", AppName, AppVersion, AppUrl)
}
