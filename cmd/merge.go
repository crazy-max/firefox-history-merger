package cmd

import (
	"os"
	"strconv"
	"strings"

	"github.com/crazy-max/firefox-history-merger/sqlite"
	"github.com/crazy-max/firefox-history-merger/sqlite/places"
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/cheggaaa/pb.v1"
)

type MergeResult struct {
	Created int
	Updated int
	Errors  int
}

var (
	mergeCmd = &cobra.Command{
		Use:     "merge",
		Short:   "Merge a working places.sqlite with other ones",
		Example: AppName + ` merge "/home/user/places.sqlite" "/home/user/folder/containing/places_sqlite/"`,
		Run:     mergeRun,
	}

	doMergeFull             bool
	doMergeMinimal          bool
	doMergeAddOrigins       bool
	doMergeAddHistoryvisits bool
	limit                   int

	lastID struct {
		MozPlaces        int
		MozHistoryvisits int
		MozOrigins       int
	}

	mergeLogFile *os.File
	mergeLog     = logrus.New()
)

func init() {
	mergeCmd.PersistentFlags().BoolVar(&doMergeMinimal, "merge-minimal", true, "Merge moz_places")
	mergeCmd.PersistentFlags().BoolVar(&doMergeFull, "merge-full", false, "Merge moz_places, moz_origins and moz_historyvisits")
	mergeCmd.PersistentFlags().BoolVar(&doMergeAddHistoryvisits, "merge-add-historyvisits", false, "Add moz_historyvisits to merge")
	mergeCmd.PersistentFlags().BoolVar(&doMergeAddOrigins, "merge-add-origins", false, "Add moz_origins to merge (formerly moz_hosts)")
	mergeCmd.PersistentFlags().IntVar(&limit, "limit", 0, "Limit the number of entries")
	RootCmd.AddCommand(mergeCmd)
}

func mergeRun(cmd *cobra.Command, args []string) {
	// check args
	if len(args) > 2 {
		Logger.Crit("merge has too many arguments")
	}
	if len(args) != 2 {
		Logger.Crit("merge requires your master places.sqlite and folder with places.sqlite files to merge")
	}
	if !FileExists(args[0]) {
		Logger.Critf("%s not found", args[0])
	}
	placesFolder, err := os.Stat(args[1])
	if err != nil {
		Logger.Critf("%s not found", args[1])
	}
	if !placesFolder.Mode().IsDir() {
		Logger.Critf("%s is not a directory", args[1])
	}

	// check and open db
	Logger.Printf("Checking and opening DBs...")
	masterPlacesDb, _, err = sqlite.OpenDbs(sqlite.SqliteFiles{
		Places: args[0], Favicons: "",
	}, true)
	if err != nil {
		Logger.Crit(err)
	}

	// working db infos
	Logger.Printf("\nMaster DB is '%s'", masterPlacesDb.Info.Filename)
	Logger.Debugf("Hash: %s:", masterPlacesDb.Info.Filehash)

	// backup db
	sqlite.BackupDb(masterPlacesDb.Info)

	// log
	mergeLogFile, err = os.OpenFile(PathJoin(appPath, "merge.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		Logger.Errorf("Merge log file: %s", err)
	}
	defer mergeLogFile.Close()
	mergeLog.Out = mergeLogFile
	mergeLog.Info("Starting merge...")

	// check merge flags
	mergeList := []string{"moz_places"}
	if doMergeFull {
		mergeList = append(mergeList, "moz_historyvisits (inc. in moz_places)", "moz_origins")
	} else {
		if doMergeAddHistoryvisits {
			mergeList = append(mergeList, "moz_historyvisits (inc. in moz_places)")
		}
		if doMergeAddOrigins {
			mergeList = append(mergeList, "moz_origins")
		}
	}
	Logger.Printf("\nThe following tables will be merged:\n- %s", strings.Join(mergeList, "\n- "))

	// open folder of places files to merge
	Logger.Printf("\nLooking for *.sqlite DBs in '%s'", args[1])
	slavePlacesDbs := sqlite.OpenPlacesDir(args[1], masterPlacesDb)
	Logger.Printf("%s valid DB(s) found:", strconv.Itoa(len(slavePlacesDbs)))
	for _, slavePlacesDb := range slavePlacesDbs {
		Logger.Printf("- %s (firefox %d v%d ; %d entries ; last used on %s)",
			slavePlacesDb.Info.Filename,
			slavePlacesDb.Info.FirefoxVersion,
			slavePlacesDb.Info.Version,
			slavePlacesDb.Info.PlacesCount,
			slavePlacesDb.Info.LastUsedTime.Format("2006-01-02 15:04:05"),
		)
	}

	// find max moz_places, moz_origins and moz_historyvisits ids
	lastID.MozPlaces = places.MozPlaces{}.GetLastID(masterPlacesDb.Link)
	lastID.MozOrigins = places.MozOrigins{}.GetLastID(masterPlacesDb.Link)
	lastID.MozHistoryvisits = places.MozHistoryvisits{}.GetLastID(masterPlacesDb.Link)
	Logger.Debugf("\nLast IDs found in working DB:")
	Logger.Debugf("- moz_places.id:        %d", lastID.MozPlaces)
	Logger.Debugf("- moz_origins.id:       %d", lastID.MozOrigins)
	Logger.Debugf("- moz_historyvisits.id: %d", lastID.MozHistoryvisits)

	// merge dbs to working db
	for _, slavePlacesDb := range slavePlacesDbs {
		Logger.Printf("\n## Merging DB '%s'...", slavePlacesDb.Info.Filename)
		merge(slavePlacesDb)
		slavePlacesDb.Link.Close()
	}

	Logger.Printf("\nOptimizing %s database...", masterPlacesDb.Info.Filename)
	if err = sqlite.Vacuum(masterPlacesDb.Link); err != nil {
		Logger.Warn(err)
	}

	masterPlacesDb.Link.Close()
}

func merge(slavePlacesDb sqlite.PlacesDb) {
	var placesResult MergeResult
	var historyvisitsResult MergeResult
	var originsResult MergeResult
	var hostsResult MergeResult

	// moz_places
	placesResult, historyvisitsResult = mergePlaces(slavePlacesDb)

	// moz_hosts or moz_origins
	if slavePlacesDb.Info.Version < 52 {
		hostsResult = mergeHosts(slavePlacesDb)
	} else {
		// TODO: check if places repaired with right origin_id
		originsResult = mergeOrigins(slavePlacesDb)
	}

	Logger.Debugf("\n[moz_places]")
	Logger.Debugf("  created = %d", placesResult.Created)
	Logger.Debugf("  updated = %d", placesResult.Updated)
	Logger.Debugf("  errors  = %d", placesResult.Errors)

	if historyvisitsResult.Created > 0 || historyvisitsResult.Errors > 0 {
		Logger.Debugf("\n[moz_historyvisits]")
		Logger.Debugf("  created = %d", historyvisitsResult.Created)
		Logger.Debugf("  errors  = %d", historyvisitsResult.Errors)
	}

	if originsResult.Created > 0 || originsResult.Updated > 0 || originsResult.Errors > 0 {
		Logger.Debugf("\n[moz_origins]")
		Logger.Debugf("  created = %d", originsResult.Created)
		Logger.Debugf("  updated = %d", originsResult.Updated)
		Logger.Debugf("  errors  = %d", originsResult.Errors)
	}

	if hostsResult.Created > 0 || hostsResult.Updated > 0 || hostsResult.Errors > 0 {
		Logger.Debugf("\n[moz_origins] from moz_hosts")
		Logger.Debugf("  created = %d", hostsResult.Created)
		Logger.Debugf("  updated = %d", hostsResult.Updated)
		Logger.Debugf("  errors  = %d", hostsResult.Errors)
	}

	if placesResult.Errors > 0 || historyvisitsResult.Errors > 0 || originsResult.Errors > 0 || hostsResult.Errors > 0 {
		cntErrors := placesResult.Errors + historyvisitsResult.Errors + originsResult.Errors + hostsResult.Errors
		Logger.Printf("\n%d error(s) occurred. Check your merge.log file", cntErrors)
	}
}

func mergePlaces(slavePlacesDb sqlite.PlacesDb) (placesResult MergeResult, historyvisitsResult MergeResult) {
	var slaveMozPlaces []places.MozPlaces
	if limit > 0 {
		slavePlacesDb.Link.Limit(limit).Find(&slaveMozPlaces)
	} else {
		slavePlacesDb.Link.Find(&slaveMozPlaces)
	}

	progBar := pb.StartNew(len(slaveMozPlaces))
	progBar.Prefix("moz_places")
	for _, slaveMozPlace := range slaveMozPlaces {
		var (
			masterMozPlace places.MozPlaces
			newMozPlace    places.MozPlaces
			updateMozPlace places.MozPlaces
		)
		masterPlacesDb.Link.Where("url = ?", slaveMozPlace.Url).Find(&masterMozPlace)

		// New moz_place entry
		if masterMozPlace.ID == 0 {
			lastID.MozPlaces++
			newMozPlace = slaveMozPlace
			newMozPlace.ID = lastID.MozPlaces
			newMozPlace.Guid = places.MozPlaces{}.GenerateGUID(slavePlacesDb.Link)
			if err := masterPlacesDb.Link.Create(&newMozPlace).Error; err != nil {
				mergeLog.Errorf("Creating moz_places row with id=%d : %s", newMozPlace.ID, err)
				placesResult.Errors++
			} else {
				placesResult.Created++
				mergeHistoryvisits(newMozPlace, slaveMozPlace, slavePlacesDb, &historyvisitsResult)
			}

			// Update moz_place entry
		} else {
			updateMozPlace = masterMozPlace
			updateMozPlace.VisitCount += slaveMozPlace.VisitCount
			updateMozPlace.LastVisitDate = MaxInt64(updateMozPlace.LastVisitDate, slaveMozPlace.LastVisitDate)
			updateMozPlace.Frecency = (updateMozPlace.Frecency + slaveMozPlace.Frecency) / 2
			if err := masterPlacesDb.Link.Save(&updateMozPlace).Error; err != nil {
				mergeLog.Errorf("Updating moz_places row with id=%d : %s", updateMozPlace.ID, err)
				placesResult.Errors++
			} else {
				placesResult.Updated++
				mergeHistoryvisits(updateMozPlace, slaveMozPlace, slavePlacesDb, &historyvisitsResult)
			}
		}
		progBar.Increment()
	}
	progBar.Finish()

	return placesResult, historyvisitsResult
}

func mergeHistoryvisits(masterMozPlaces places.MozPlaces, slaveMozPlaces places.MozPlaces, slavePlacesDb sqlite.PlacesDb, result *MergeResult) {
	if !doMergeFull && !doMergeAddHistoryvisits {
		return
	}

	var slaveMozHistoryvisits []places.MozHistoryvisits
	slavePlacesDb.Link.Where("place_id = ?", slaveMozPlaces.ID).Find(&slaveMozHistoryvisits)
	for _, slaveMozHistoryvisit := range slaveMozHistoryvisits {
		// Places already exists
		if masterMozPlaces.ID != lastID.MozPlaces {
			// Check history visit collisions before create
			var masterMozHistoryvisits places.MozHistoryvisits
			//FIXME: No unique columns so fingerprint with from_visit, place_id and visit_date
			masterPlacesDb.Link.Where("from_visit = ? AND place_id = ? AND visit_date = ?",
				slaveMozHistoryvisit.FromVisit,
				masterMozPlaces.ID,
				slaveMozHistoryvisit.VisitDate,
			).First(&masterMozHistoryvisits)

			if masterMozHistoryvisits.ID > 0 {
				return
			}
		}

		lastID.MozHistoryvisits++
		slaveMozHistoryvisit.ID = lastID.MozHistoryvisits
		slaveMozHistoryvisit.PlaceId = masterMozPlaces.ID
		slaveMozHistoryvisit.FromVisit = 0 //TODO: Find a way to retrieve ancestors. Fills from_visit with 0 temporarily.

		if err := masterPlacesDb.Link.Create(&slaveMozHistoryvisit).Error; err != nil {
			mergeLog.Errorf("Creating moz_historyvisits row with id=%d : %s", slaveMozHistoryvisit.ID, err)
			result.Errors++
		} else {
			result.Created++
		}
	}
}

func mergeHosts(slavePlacesDb sqlite.PlacesDb) MergeResult {
	var result MergeResult

	if !doMergeFull && !doMergeAddOrigins {
		return result
	}

	var slaveMozHosts []places.MozHosts
	slavePlacesDb.Link.Find(&slaveMozHosts)

	progBar := pb.StartNew(len(slaveMozHosts))
	progBar.Prefix("moz_hosts")
	for _, slaveMozHost := range slaveMozHosts {
		var masterMozOrigin places.MozOrigins
		masterPlacesDb.Link.Where("host = ?", slaveMozHost.Host).First(&masterMozOrigin)
		if masterMozOrigin.Host == "" {
			lastID.MozOrigins++
			masterMozOrigin.ID = lastID.MozOrigins
			masterMozOrigin.Prefix = slaveMozHost.Prefix
			masterMozOrigin.Host = slaveMozHost.Host
			masterMozOrigin.Frecency = slaveMozHost.Frecency
			if err := masterPlacesDb.Link.Create(&masterMozOrigin).Error; err != nil {
				mergeLog.Errorf("Creating moz_origins row with id=%d : %s", masterMozOrigin.ID, err)
				result.Errors++
			} else {
				result.Created++
			}
		} else {
			masterMozOrigin.Frecency = (masterMozOrigin.Frecency + slaveMozHost.Frecency) / 2
			if err := masterPlacesDb.Link.Save(&masterMozOrigin).Error; err != nil {
				mergeLog.Errorf("Updating moz_origins row with id=%d : %s", masterMozOrigin.ID, err)
				result.Errors++
			} else {
				result.Updated++
			}
		}
		progBar.Increment()
	}
	progBar.Finish()

	return result
}

func mergeOrigins(slavePlacesDb sqlite.PlacesDb) MergeResult {
	var result MergeResult

	if !doMergeFull && !doMergeAddOrigins {
		return result
	}

	var slaveMozOrigins []places.MozOrigins
	slavePlacesDb.Link.Find(&slaveMozOrigins)

	progBar := pb.StartNew(len(slaveMozOrigins))
	progBar.Prefix("moz_origins")
	for _, slaveMozOrigin := range slaveMozOrigins {
		var masterMozOrigin places.MozOrigins
		masterPlacesDb.Link.Where("host = ?", slaveMozOrigin.Host).First(&masterMozOrigin)
		if masterMozOrigin.Host == "" {
			lastID.MozOrigins++
			slaveMozOrigin.ID = lastID.MozOrigins
			if err := masterPlacesDb.Link.Create(&slaveMozOrigin).Error; err != nil {
				mergeLog.Errorf("Creating moz_origins row with id=%d : %s", slaveMozOrigin.ID, err)
				result.Errors++
			} else {
				result.Created++
			}
		} else {
			masterMozOrigin.Frecency = (masterMozOrigin.Frecency + slaveMozOrigin.Frecency) / 2
			if err := masterPlacesDb.Link.Save(&masterMozOrigin).Error; err != nil {
				mergeLog.Errorf("Updating moz_origins row with id=%d : %s", masterMozOrigin.ID, err)
				result.Errors++
			} else {
				result.Updated++
			}
		}
		progBar.Increment()
	}
	progBar.Finish()

	return result
}
