package main

import (
	"fmt"
	"os"

	"github.com/crazy-max/firefox-history-merger/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
