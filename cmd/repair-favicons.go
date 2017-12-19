package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/crazy-max/firefox-history-merger/sqlite"
	"github.com/crazy-max/firefox-history-merger/sqlite/table"
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/mat/besticon/besticon"
	"github.com/spf13/cobra"
	"gopkg.in/cheggaaa/pb.v1"
)

var (
	repairFaviconsCmd = &cobra.Command{
		Use:     "repair-favicons",
		Short:   "Repair favicons in places.sqlite",
		Example: AppName + ` repair-favicons "/home/user/places.sqlite"`,
		Args:    cobra.ExactArgs(1),
		Run:     repairFavicons,
	}
)

const (
	RepairFaviconsPages = 100
)

type RepairFaviconsResult struct {
	lastFaviconsID int
	countValid     int
	countRepaired  int
	countSkipped   int
	countExist     int
	countErrors    int
}

func init() {
	RootCmd.AddCommand(repairFaviconsCmd)
}

func repairFavicons(cmd *cobra.Command, args []string) {
	var result RepairFaviconsResult

	// besticon settings
	besticon.SetLogOutput(ioutil.Discard)
	besticon.SetCacheMaxSize(128)

	// https://github.com/golang/go/issues/19895
	log.SetOutput(ioutil.Discard)

	// check and open DB
	sqliteDBFile, err := filepath.Abs(args[0])
	CheckIfError(err)
	sqliteDB := sqlite.OpenFile(sqliteDBFile)

	// backup db
	sqliteDBFileBackup := fmt.Sprintf("%s.%s", sqliteDBFile, time.Now().Format("20060102150405"))
	fmt.Printf("Backing up '%s' to '%s'\n", sqliteDB.Info.Filename, filepath.Base(sqliteDBFileBackup))
	CopyFile(args[0], sqliteDBFileBackup)
	CheckIfError(err)

	// get moz_places count
	var mozPlacesCount int
	sqliteDB.Link.Model(table.MozPlaces{}).Count(&mozPlacesCount)
	fmt.Printf("Places to check: %d\n", mozPlacesCount)

	// get last moz_favicons.id
	result.lastFaviconsID = table.MozFavicons{}.GetLastID(sqliteDB.Link)
	fmt.Printf("Last moz_favicons.id: %d\n", result.lastFaviconsID)

	// get first moz_places.id
	firstPlacesID := table.MozPlaces{}.GetFirstID(sqliteDB.Link)

	// start repair
	pageSize := int(math.Ceil(float64(mozPlacesCount / RepairFaviconsPages)))

	if DebugEnabled {
		fmt.Printf("\nPaginate moz_places:\n")
		fmt.Printf("- total rows: %d\n", mozPlacesCount)
		fmt.Printf("- first id:   %d\n", firstPlacesID)
		fmt.Printf("- pages:      %d\n", RepairFaviconsPages)
		fmt.Printf("- page size:  %d\n", pageSize)
	}

	fmt.Printf("\n## Repairing favicons...\n")
	progBar := pb.StartNew(mozPlacesCount)
	progBar.Prefix("repair moz_favicons")

	lastPlacesID := 0
	runtime.GOMAXPROCS(2)
	wg := new(sync.WaitGroup)
	for i := 0; i <= RepairFaviconsPages; i++ {
		var mozPlaces []table.MozPlaces
		if lastPlacesID == 0 {
			sqliteDB.Link.Order("id ASC").Limit(pageSize).Find(&mozPlaces)
		} else {
			sqliteDB.Link.Where("id > ?", lastPlacesID).Order("id ASC").Limit(pageSize).Find(&mozPlaces)
		}
		if len(mozPlaces) == 0 {
			continue
		}
		lastPlacesID = mozPlaces[len(mozPlaces)-1].ID

		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, mozPlace := range mozPlaces {
				var (
					mozFavicon               table.MozFavicons
					foundExistingMozFavicons table.MozFavicons
				)

				// seek favicon
				sqliteDB.Link.First(&mozFavicon, mozPlace.FaviconId)

				// valid
				if mozFavicon.Url != "" {
					result.countValid++
					progBar.Increment()
					continue
				}

				// skip unvalid URL
				host, _, err := sqlite.FixupUrl(mozPlace.Url)
				if err != nil || Contains([]string{"localhost", "127.0.0.1"}, host) || !strings.HasPrefix(mozPlace.Url, "http://") && !strings.HasPrefix(mozPlace.Url, "https://") {
					result.countSkipped++
					progBar.Increment()
					continue
				}

				// get favicon
				ico, err := GetFavicon(mozPlace.Url)
				if err != nil {
					result.countErrors++
					progBar.Increment()
					continue
				}

				result.lastFaviconsID++
				mozFavicon.ID = result.lastFaviconsID
				mozFavicon.Url = ico.URL
				mozFavicon.Data = ico.ImageData
				if ico.Format == "png" {
					mozFavicon.MimeType = "image/png"
				} else if ico.Format == "gif" {
					mozFavicon.MimeType = "image/gif"
				} else if ico.Format == "ico" {
					mozFavicon.MimeType = "image/x-icon"
				}
				mozFavicon.Expiration = 0

				// check if found existing favicon
				sqliteDB.Link.Where("url = ?", ico.URL).First(&foundExistingMozFavicons)
				if foundExistingMozFavicons.Url != "" {
					mozPlace.FaviconId = foundExistingMozFavicons.ID
					result.countExist++
				} else {
					if err := sqliteDB.Link.Create(mozFavicon).Error; err != nil {
						Error("Creating moz_favicons row with id=%d : %s", mozFavicon.ID, err)
						result.countErrors++
						progBar.Increment()
						continue
					}
					mozPlace.FaviconId = result.lastFaviconsID
					result.countRepaired++
				}

				// update places
				if err := sqliteDB.Link.Save(mozPlace).Error; err != nil {
					result.countRepaired++
					Error("Updating moz_places row with id=%d : %s", mozPlace.ID, err)
				}

				progBar.Increment()
			}
		}()
	}

	wg.Wait()
	progBar.Finish()

	fmt.Printf("\nResult\n")
	fmt.Printf("  valid    = %d\n", result.countValid)
	fmt.Printf("  repaired = %d\n", result.countRepaired)
	fmt.Printf("  skipped  = %d\n", result.countSkipped)
	fmt.Printf("  exist    = %d\n", result.countExist)
	fmt.Printf("  errors   = %d\n", result.countErrors)

	sqliteDB.Link.Close()
}
