package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kingpin"
	"github.com/crazy-max/firefox-history-merger/internal/app"
	"github.com/crazy-max/firefox-history-merger/internal/logging"
	"github.com/crazy-max/firefox-history-merger/internal/model"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	fhm     *app.FirefoxHistoryMerger
	flags   model.Flags
	version = "dev"
)

const (
	helpPlacesPath   = "places.sqlite database file path."
	helpFaviconsPath = "favicons.sqlite database file path."
)

func main() {
	var err error

	cmd := kingpin.New("firefox-history-merger", "Merge Firefox history and repair missing favicons with ease.\nMoreinfo: https://github.com/crazy-max/firefox-history-merger")
	cmd.Flag("log-level", "Set log level.").Default(zerolog.InfoLevel.String()).StringVar(&flags.LogLevel)
	cmd.Flag("log-caller", "Enable to add file:line of the caller.").Default("false").BoolVar(&flags.LogCaller)
	cmd.UsageTemplate(kingpin.CompactUsageTemplate).Version(version).Author("CrazyMax")

	info := cmd.Command("info", "Display info about places database.")
	info.Arg("places-db", helpPlacesPath).Required().StringVar(&flags.PlacesFile)

	merge := cmd.Command("merge", "Merge a working places databases with another one.")
	merge.Arg("places-db", "File path to your working places database.").Required().StringVar(&flags.PlacesFile)
	merge.Arg("places-db-tomerge", "File path to a places databases to merge.").Required().StringVar(&flags.PlacesToMergeFile)
	merge.Flag("max-per-tx", "Number of records to be merged per transaction.").Default("1000").IntVar(&flags.MaxPerTx)

	repairFavicons := cmd.Command("repair-favicons", "Repair broken favicons")
	repairFavicons.Arg("places-db", helpPlacesPath).Required().StringVar(&flags.PlacesFile)
	repairFavicons.Arg("favicons-db", helpFaviconsPath).Required().StringVar(&flags.FaviconsFile)
	repairFavicons.Flag("workers", "Maximum number of workers.").Default("30").IntVar(&flags.Workers)

	optimize := cmd.Command("optimize", "Optimize a database into a minimal amount of disk space.")
	optimize.Arg("db", "Path to sqlite database").Required().StringVar(&flags.DbFile)

	_, _ = cmd.Parse(os.Args[1:])

	// Logger
	logging.Configure(flags)

	// Init
	if fhm, err = app.New(flags); err != nil {
		log.Fatal().Err(err).Msg("Cannot initialize")
	}

	// Handle os signals
	channel := make(chan os.Signal)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-channel
		fhm.Close()
		log.Warn().Msgf("Caught signal %v", sig)
		os.Exit(0)
	}()

	switch kingpin.MustParse(cmd.Parse(os.Args[1:])) {
	case info.FullCommand():
		fhm.Info()
	case merge.FullCommand():
		fhm.Merge()
	case optimize.FullCommand():
		fhm.Optimize()
	case repairFavicons.FullCommand():
		fhm.RepairFavicons()
	default:
		log.Fatal().Err(err).Msg("Unknown command")
	}
}
