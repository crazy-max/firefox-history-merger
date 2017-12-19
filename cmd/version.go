package cmd

import (
	"fmt"

	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show " + AppName + " version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(RootCmd.Use + " " + AppVersion + "\n" + AppUrl + "\n")
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
