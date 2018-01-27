package cmd

import (
	"os"
	"strconv"
	"strings"

	"github.com/crazy-max/firefox-history-merger/sqlite"
	"github.com/crazy-max/firefox-history-merger/sqlite/places"
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/spf13/cobra"
	pb "gopkg.in/cheggaaa/pb.v1"
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
	doMergeAddHosts         bool
	doMergeAddHistoryvisits bool

	lastID struct {
		MozPlaces        int
		MozHistoryvisits int
		MozHosts         int
	}
)

func init() {
	mergeCmd.PersistentFlags().BoolVar(&doMergeMinimal, "merge-minimal", true, "Merge moz_places")
	mergeCmd.PersistentFlags().BoolVar(&doMergeFull, "merge-full", false, "Merge moz_places, moz_hosts and moz_historyvisits")
	mergeCmd.PersistentFlags().BoolVar(&doMergeAddHistoryvisits, "merge-add-historyvisits", false, "Add moz_historyvisits to merge")
	mergeCmd.PersistentFlags().BoolVar(&doMergeAddHosts, "merge-add-hosts", false, "Add moz_hosts to merge")
	RootCmd.AddCommand(mergeCmd)
}

func mergeRun(cmd *cobra.Command, args []string) {
	// check args
	if len(args) > 2 {
		Logger.Crit("merge has too many arguments")
	}
	if len(args) != 2 {
		Logger.Crit("merge requires your current places.sqlite and folder with places.sqlite files to merge")
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
	placesDb, _, err = sqlite.OpenDbs(sqlite.SqliteFiles{
		Places: args[0], Favicons: "",
	}, true)
	if err != nil {
		Logger.Crit(err)
	}

	// working db infos
	Logger.Printf("\nWorking DB is '%s'", placesDb.Info.Filename)
	Logger.Debugf("Hash: %s:", placesDb.Info.Filehash)

	// backup db
	sqlite.BackupDb(placesDb.Info)

	// check merge flags
	mergeList := []string{"moz_places"}
	if doMergeFull {
		mergeList = append(mergeList, "moz_historyvisits (inc. in moz_places)", "moz_hosts")
	} else {
		if doMergeAddHistoryvisits {
			mergeList = append(mergeList, "moz_historyvisits (inc. in moz_places)")
		}
		if doMergeAddHosts {
			mergeList = append(mergeList, "moz_hosts")
		}
	}
	Logger.Printf("\nThe following tables will be merged:\n- %s", strings.Join(mergeList, "\n- "))

	// open folder of places files to merge
	Logger.Printf("\nLooking for *.sqlite DBs in '%s'", args[1])
	sqliteDBs := sqlite.OpenPlacesDir(args[1], placesDb)
	Logger.Printf("%s valid DB(s) found:", strconv.Itoa(len(sqliteDBs)))
	for _, sqliteDB := range sqliteDBs {
		Logger.Printf("- %s (%d entries ; last used on %s)",
			sqliteDB.Info.Filename,
			sqliteDB.Info.PlacesCount,
			sqliteDB.Info.LastUsedTime.Format("2006-01-02 15:04:05"),
		)
	}

	// find max moz_places, moz_hosts and moz_historyvisits ids
	lastID.MozPlaces = places.MozPlaces{}.GetLastID(placesDb.Link)
	lastID.MozHosts = places.MozHosts{}.GetLastID(placesDb.Link)
	lastID.MozHistoryvisits = places.MozHistoryvisits{}.GetLastID(placesDb.Link)
	Logger.Debugf("\nLast IDs found in working DB:")
	Logger.Debugf("- moz_places.id:        %d", lastID.MozPlaces)
	Logger.Debugf("- moz_hosts.id:         %d", lastID.MozHosts)
	Logger.Debugf("- moz_historyvisits.id: %d", lastID.MozHistoryvisits)

	// merge dbs to working db
	for _, currentPlacesDb := range sqliteDBs {
		Logger.Printf("\n## Merging DB '%s'...", currentPlacesDb.Info.Filename)
		merge(currentPlacesDb)
		currentPlacesDb.Link.Close()
	}

	Logger.Printf("\nOptimizing %s database...", placesDb.Info.Filename)
	if err = sqlite.Vacuum(placesDb.Link); err != nil {
		Logger.Warn(err)
	}

	placesDb.Link.Close()
}

func merge(currentPlacesDb sqlite.PlacesDb) {
	var placesResult MergeResult
	var historyvisitsResult MergeResult
	var hostsResult MergeResult

	// moz_places
	placesResult, historyvisitsResult = mergePlaces(currentPlacesDb)

	// moz_hosts
	if doMergeFull || doMergeAddHosts {
		hostsResult = mergeHosts(currentPlacesDb)
	}

	Logger.Debugf("\n[moz_places]")
	Logger.Debugf("  created = %d", placesResult.Created)
	Logger.Debugf("  updated = %d", placesResult.Updated)
	Logger.Debugf("  errors  = %d", placesResult.Errors)

	if doMergeFull || doMergeAddHistoryvisits {
		Logger.Debugf("\n[moz_historyvisits]")
		Logger.Debugf("  created = %d", historyvisitsResult.Created)
		Logger.Debugf("  errors  = %d", historyvisitsResult.Errors)
	}

	if doMergeFull || doMergeAddHosts {
		Logger.Debugf("\n[moz_hosts]")
		Logger.Debugf("  created = %d", hostsResult.Created)
		Logger.Debugf("  updated = %d", hostsResult.Updated)
		Logger.Debugf("  errors  = %d", hostsResult.Errors)
	}
}

func mergePlaces(currentPlacesDb sqlite.PlacesDb) (placesResult MergeResult, historyvisitsResult MergeResult) {
	var currentMozPlaces []places.MozPlaces
	currentPlacesDb.Link.Find(&currentMozPlaces)
	//currentPlacesDb.Link.Limit(500).Find(&currentMozPlaces)

	progBar := pb.StartNew(len(currentMozPlaces))
	progBar.Prefix("moz_places")
	for _, currentMozPlace := range currentMozPlaces {
		var (
			workingMozPlace places.MozPlaces
			newMozPlace     places.MozPlaces
			updateMozPlace  places.MozPlaces
		)
		placesDb.Link.Where("url = ?", currentMozPlace.Url).Find(&workingMozPlace)
		if workingMozPlace.ID == 0 {
			lastID.MozPlaces++
			newMozPlace = currentMozPlace
			newMozPlace.ID = lastID.MozPlaces
			newMozPlace.Guid = places.MozPlaces{}.GenerateGUID(currentPlacesDb.Link)
			if err := placesDb.Link.Create(&newMozPlace).Error; err != nil {
				Logger.Errorf("Creating moz_places row with id=%d : %s", newMozPlace.ID, err)
				placesResult.Errors++
			} else {
				placesResult.Created++
				if doMergeFull || doMergeAddHistoryvisits {
					mergeHistoryvisits(newMozPlace, currentMozPlace, currentPlacesDb, &historyvisitsResult)
				}
			}
		} else {
			updateMozPlace = workingMozPlace
			updateMozPlace.VisitCount += currentMozPlace.VisitCount
			updateMozPlace.LastVisitDate = MaxInt64(updateMozPlace.LastVisitDate, currentMozPlace.LastVisitDate)
			updateMozPlace.Frecency = (updateMozPlace.Frecency + currentMozPlace.Frecency) / 2
			if err := placesDb.Link.Save(&updateMozPlace).Error; err != nil {
				Logger.Errorf("Updating moz_places row with id=%d : %s", updateMozPlace.ID, err)
				placesResult.Errors++
			} else {
				placesResult.Updated++
				if doMergeFull || doMergeAddHistoryvisits {
					mergeHistoryvisits(updateMozPlace, currentMozPlace, currentPlacesDb, &historyvisitsResult)
				}
			}
		}
		progBar.Increment()
	}
	progBar.Finish()

	return placesResult, historyvisitsResult
}

func mergeHistoryvisits(workingMozPlaces places.MozPlaces, currentMozPlaces places.MozPlaces, currentPlacesDb sqlite.PlacesDb, result *MergeResult) {
	var currentMozHistoryvisits []places.MozHistoryvisits
	currentPlacesDb.Link.Where("place_id = ?", currentMozPlaces.ID).Find(&currentMozHistoryvisits)
	for _, currentMozHistoryvisit := range currentMozHistoryvisits {
		// Places already exists
		if workingMozPlaces.ID != lastID.MozPlaces {
			// Check history visit collisions before create
			var workingMozHistoryvisits places.MozHistoryvisits
			//FIXME: No unique columns so fingerprint with from_visit, place_id and visit_date
			placesDb.Link.Where("from_visit = ? AND place_id = ? AND visit_date = ?",
				currentMozHistoryvisit.FromVisit,
				workingMozPlaces.ID,
				currentMozHistoryvisit.VisitDate,
			).First(&workingMozHistoryvisits)

			if workingMozHistoryvisits.ID > 0 {
				return
			}
		}

		lastID.MozHistoryvisits++
		currentMozHistoryvisit.ID = lastID.MozHistoryvisits
		currentMozHistoryvisit.PlaceId = workingMozPlaces.ID
		currentMozHistoryvisit.FromVisit = 0 //TODO: Find a way to retrieve ancestors. Fills from_visit with 0 temporarily.

		if err := placesDb.Link.Create(&currentMozHistoryvisit).Error; err != nil {
			Logger.Errorf("Creating moz_historyvisits row with id=%d : %s", currentMozHistoryvisit.ID, err)
			result.Errors++
		} else {
			result.Created++
		}
	}
}

func mergeHosts(currentPlacesDb sqlite.PlacesDb) MergeResult {
	var result MergeResult

	var currentMozHosts []places.MozHosts
	currentPlacesDb.Link.Find(&currentMozHosts)

	progBar := pb.StartNew(len(currentMozHosts))
	progBar.Prefix("moz_hosts")
	for _, currentMozHost := range currentMozHosts {
		var workingMozHost places.MozHosts
		placesDb.Link.Where("host = ?", currentMozHost.Host).First(&workingMozHost)
		if workingMozHost.Host == "" {
			lastID.MozHosts++
			currentMozHost.ID = lastID.MozHosts
			if err := placesDb.Link.Create(&currentMozHost).Error; err != nil {
				Logger.Errorf("Creating moz_hosts row with id=%d : %s", currentMozHost.ID, err)
				result.Errors++
			} else {
				result.Created++
			}
		} else {
			workingMozHost.Frecency = (workingMozHost.Frecency + currentMozHost.Frecency) / 2
			if err := placesDb.Link.Save(&workingMozHost).Error; err != nil {
				Logger.Errorf("Updating moz_hosts row with id=%d : %s", workingMozHost.ID, err)
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
