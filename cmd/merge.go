package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/crazy-max/firefox-history-merger/sqlite"
	"github.com/crazy-max/firefox-history-merger/sqlite/table"
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
		Example: AppName + ` merge "/home/user/places.sqlite" "/home/user/folder/containing/*.sqlite"`,
		Args:    cobra.ExactArgs(2),
		Run:     mergeRun,
	}

	doMergeFull             bool
	doMergeMinimal          bool
	doMergeAddHosts         bool
	doMergeAddHistoryvisits bool

	lastID struct {
		MozPlaces        int
		MozFavicons      int
		MozHistoryvisits int
		MozHosts         int
	}
)

func init() {
	mergeCmd.PersistentFlags().BoolVar(&doMergeMinimal, "merge-minimal", true, "Merge moz_places and moz_favicons")
	mergeCmd.PersistentFlags().BoolVar(&doMergeFull, "merge-full", false, "Merge moz_places, moz_favicons, moz_hosts and moz_historyvisits")
	mergeCmd.PersistentFlags().BoolVar(&doMergeAddHistoryvisits, "merge-add-historyvisits", false, "Add moz_historyvisits to merge")
	mergeCmd.PersistentFlags().BoolVar(&doMergeAddHosts, "merge-add-hosts", false, "Add moz_hosts to merge")
	RootCmd.AddCommand(mergeCmd)
}

func mergeRun(cmd *cobra.Command, args []string) {
	// Check and open working DB
	workingSqliteDBFile, err := filepath.Abs(args[0])
	CheckIfError(err)
	workingSqliteDB := sqlite.OpenFile(workingSqliteDBFile)
	fmt.Printf("\nWorking DB is '%s'", workingSqliteDB.Info.Filename)
	if DebugEnabled {
		fmt.Printf("\nHash: %s:", workingSqliteDB.Info.Filehash)
	}

	// Backup working db
	workingDBFileBackup := fmt.Sprintf("%s.%s", workingSqliteDBFile, time.Now().Format("20060102150405"))
	fmt.Printf("\nBacking up working DB to '%s'...", filepath.Base(workingDBFileBackup))
	CopyFile(args[0], workingDBFileBackup)
	CheckIfError(err)

	// Check merge flags
	mergeList := []string{"moz_places", "moz_favicons (inc. in moz_places)"}
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
	fmt.Printf("\n\nThe following tables will be merged:\n- %s", strings.Join(mergeList, "\n- "))

	// Open folder of places files to merge
	fmt.Printf("\n\nLooking for *.sqlite DBs in '%s'", args[1])
	sqliteDBs := sqlite.OpenDir(args[1], workingSqliteDB)
	fmt.Printf("\n%s valid DB(s) found:", strconv.Itoa(len(sqliteDBs)))
	for _, sqliteDB := range sqliteDBs {
		fmt.Printf("\n- %s (%d entries ; last used on %s)",
			sqliteDB.Info.Filename,
			sqliteDB.Info.PlacesCount,
			sqliteDB.Info.LastUsedTime.Format("2006-01-02 15:04:05"),
		)
	}

	// Migrate working db
	fmt.Printf("\n\nMigrating '%s' to schema v%d...", workingSqliteDB.Info.Filename, sqlite.DbSchema)
	if err := workingSqliteDB.Link.AutoMigrate(&table.MozPlaces{}, table.MozFavicons{}, table.MozHistoryvisits{}, table.MozHosts{}).Error; err != nil {
		Warning(err.Error())
	}

	// Find max moz_places id, moz_historyvisits id and moz_favicons id
	lastID.MozPlaces = table.MozPlaces{}.GetLastID(workingSqliteDB.Link)
	lastID.MozFavicons = table.MozFavicons{}.GetLastID(workingSqliteDB.Link)
	lastID.MozHosts = table.MozHosts{}.GetLastID(workingSqliteDB.Link)
	lastID.MozHistoryvisits = table.MozHistoryvisits{}.GetLastID(workingSqliteDB.Link)
	if DebugEnabled {
		fmt.Printf("\n\nLast IDs found in working DB:")
		fmt.Printf("\n- moz_places.id:        %d", lastID.MozPlaces)
		fmt.Printf("\n- moz_favicons.id:      %d", lastID.MozFavicons)
		fmt.Printf("\n- moz_hosts.id:         %d", lastID.MozHosts)
		fmt.Printf("\n- moz_historyvisits.id: %d", lastID.MozHistoryvisits)
	}

	// Merge DBs to working DB
	for _, currentSqliteDB := range sqliteDBs {
		fmt.Printf("\n\n## Merging DB '%s'...\n", currentSqliteDB.Info.Filename)
		merge(currentSqliteDB, workingSqliteDB)
		currentSqliteDB.Link.Close()
	}

	// Close working DB
	workingSqliteDB.Link.Close()
}

func merge(currentSqliteDB sqlite.Db, workingSqliteDB sqlite.Db) {
	var placesResult MergeResult
	var faviconsResult MergeResult
	var historyvisitsResult MergeResult
	var hostsResult MergeResult

	// moz_places
	placesResult, faviconsResult, historyvisitsResult = mergePlaces(currentSqliteDB, workingSqliteDB)

	// moz_hosts
	if doMergeFull || doMergeAddHosts {
		hostsResult = mergeHosts(currentSqliteDB, workingSqliteDB)
	}

	if DebugEnabled {
		fmt.Printf("\n\n[moz_places]")
		fmt.Printf("\n  created = %d", placesResult.Created)
		fmt.Printf("\n  updated = %d", placesResult.Updated)
		fmt.Printf("\n  errors  = %d", placesResult.Errors)

		fmt.Printf("\n\n[moz_favicons]")
		fmt.Printf("\n  created = %d\n", faviconsResult.Created)
		fmt.Printf("\n  errors  = %d\n", faviconsResult.Errors)

		if doMergeFull || doMergeAddHistoryvisits {
			fmt.Printf("\n\n[moz_historyvisits]")
			fmt.Printf("\n  created = %d", historyvisitsResult.Created)
			fmt.Printf("\n  errors  = %d", historyvisitsResult.Errors)
		}

		if doMergeFull || doMergeAddHosts {
			fmt.Printf("\n\n[moz_hosts]\n")
			fmt.Printf("\n  created = %d", hostsResult.Created)
			fmt.Printf("\n  updated = %d", hostsResult.Updated)
			fmt.Printf("\n  errors  = %d", hostsResult.Errors)
		}
	}
}

func mergePlaces(currentSqliteDB sqlite.Db, workingSqliteDB sqlite.Db) (placesResult MergeResult, faviconsResult MergeResult, historyvisitsResult MergeResult) {
	var currentMozPlaces []table.MozPlaces
	currentSqliteDB.Link.Find(&currentMozPlaces)
	//currentSqliteDB.Link.Limit(500).Find(&currentMozPlaces)

	progBar := pb.StartNew(len(currentMozPlaces))
	progBar.Prefix("moz_places")
	for _, currentMozPlace := range currentMozPlaces {
		var (
			workingMozPlace table.MozPlaces
			newMozPlace     table.MozPlaces
			updateMozPlace  table.MozPlaces
		)
		workingSqliteDB.Link.Where("url = ?", currentMozPlace.Url).Find(&workingMozPlace)
		if workingMozPlace.ID == 0 {
			lastID.MozPlaces++
			newMozPlace = currentMozPlace
			newMozPlace.ID = lastID.MozPlaces
			newMozPlace.Guid = table.MozPlaces{}.GenerateGUID(currentSqliteDB.Link)
			if err := workingSqliteDB.Link.Create(&newMozPlace).Error; err != nil {
				Error("Creating moz_places row with id=%d : %s", newMozPlace.ID, err)
				placesResult.Errors++
			} else {
				placesResult.Created++
				mergeFavicon(newMozPlace, currentMozPlace, currentSqliteDB, workingSqliteDB, &faviconsResult)
				if doMergeFull || doMergeAddHistoryvisits {
					mergeHistoryvisits(newMozPlace, currentMozPlace, currentSqliteDB, workingSqliteDB, &historyvisitsResult)
				}
			}
		} else {
			updateMozPlace = workingMozPlace
			updateMozPlace.VisitCount += currentMozPlace.VisitCount
			updateMozPlace.LastVisitDate = MaxInt64(updateMozPlace.LastVisitDate, currentMozPlace.LastVisitDate)
			updateMozPlace.Frecency = (updateMozPlace.Frecency + currentMozPlace.Frecency) / 2
			if err := workingSqliteDB.Link.Save(updateMozPlace).Error; err != nil {
				Error("Updating moz_places row with id=%d : %s", updateMozPlace.ID, err)
				placesResult.Errors++
			} else {
				placesResult.Updated++
				mergeFavicon(updateMozPlace, currentMozPlace, currentSqliteDB, workingSqliteDB, &faviconsResult)
				if doMergeFull || doMergeAddHistoryvisits {
					mergeHistoryvisits(updateMozPlace, currentMozPlace, currentSqliteDB, workingSqliteDB, &historyvisitsResult)
				}
			}
		}
		progBar.Increment()
	}
	progBar.Finish()

	return placesResult, faviconsResult, historyvisitsResult
}

func mergeFavicon(workingMozPlaces table.MozPlaces, currentMozPlaces table.MozPlaces, currentSqliteDB sqlite.Db, workingSqliteDB sqlite.Db, result *MergeResult) {
	var currentMozFavicons table.MozFavicons
	var workingMozFavicons table.MozFavicons

	currentSqliteDB.Link.First(&currentMozFavicons, currentMozPlaces.FaviconId)
	if currentMozFavicons.Url == "" {
		return
	}

	workingSqliteDB.Link.Where("url = ?", currentMozFavicons.Url).First(&workingMozFavicons)
	if workingMozFavicons.Url == "" {
		lastID.MozFavicons++
		currentMozFavicons.ID = lastID.MozFavicons
		if err := workingSqliteDB.Link.Create(currentMozFavicons).Error; err != nil {
			result.Errors++
			Error("Creating moz_favicons row with id=%d : %s", currentMozFavicons.ID, err)
		}
		workingMozPlaces.FaviconId = lastID.MozFavicons
		result.Created++
	} else {
		workingMozPlaces.FaviconId = workingMozFavicons.ID
	}

	if err := workingSqliteDB.Link.Save(workingMozPlaces).Error; err != nil {
		result.Errors++
		Error("Updating moz_places row with id=%d : %s", workingMozPlaces.ID, err)
	}
}

func mergeHistoryvisits(workingMozPlaces table.MozPlaces, currentMozPlaces table.MozPlaces, currentSqliteDB sqlite.Db, workingSqliteDB sqlite.Db, result *MergeResult) {
	var currentMozHistoryvisits []table.MozHistoryvisits
	currentSqliteDB.Link.Where("place_id = ?", currentMozPlaces.ID).Find(&currentMozHistoryvisits)
	for _, currentMozHistoryvisit := range currentMozHistoryvisits {
		// Places already exists
		if workingMozPlaces.ID != lastID.MozPlaces {
			// Check history visit collisions before create
			var workingMozHistoryvisits table.MozHistoryvisits
			//FIXME: No unique columns so fingerprint with from_visit, place_id and visit_date
			workingSqliteDB.Link.Where("from_visit = ? AND place_id = ? AND visit_date = ?",
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

		if err := workingSqliteDB.Link.Create(currentMozHistoryvisit).Error; err != nil {
			result.Errors++
			Error("Creating moz_historyvisits row with id=%d : %s", currentMozHistoryvisit.ID, err)
		}
		result.Created++
	}
}

func mergeHosts(currentSqliteDB sqlite.Db, workingSqliteDB sqlite.Db) MergeResult {
	var result MergeResult

	var currentMozHosts []table.MozHosts
	currentSqliteDB.Link.Find(&currentMozHosts)

	progBar := pb.StartNew(len(currentMozHosts))
	progBar.Prefix("moz_hosts")
	for _, currentMozHost := range currentMozHosts {
		var workingMozHost table.MozHosts
		workingSqliteDB.Link.Where("host = ?", currentMozHost.Host).First(&workingMozHost)
		if workingMozHost.Host == "" {
			lastID.MozHosts++
			currentMozHost.ID = lastID.MozHosts
			if err := workingSqliteDB.Link.Create(&currentMozHost).Error; err != nil {
				Error("Creating moz_hosts row with id=%d : %s", currentMozHost.ID, err)
				result.Errors++
			} else {
				result.Created++
			}
		} else {
			workingMozHost.Frecency = (workingMozHost.Frecency + currentMozHost.Frecency) / 2
			if err := workingSqliteDB.Link.Save(&workingMozHost).Error; err != nil {
				Error("Updating moz_hosts row with id=%d : %s", workingMozHost.ID, err)
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
