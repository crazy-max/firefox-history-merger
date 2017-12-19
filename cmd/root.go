package cmd

import (
	"fmt"
	"os"

	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   AppName,
	Short: AppDescription,
	Long: AppDescription + `.
More info on ` + AppUrl,
}

var Dir string
var DebugEnabled bool

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&DebugEnabled, "debug", "x", false, "Debug")
}
