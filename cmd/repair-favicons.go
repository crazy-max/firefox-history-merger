package cmd

import (
	"io/ioutil"
	"log"
	"math"
	"strings"

	"github.com/crazy-max/firefox-history-merger/sqlite"
	"github.com/crazy-max/firefox-history-merger/sqlite/favicons"
	"github.com/crazy-max/firefox-history-merger/sqlite/places"
	. "github.com/crazy-max/firefox-history-merger/utils"
	"github.com/mat/besticon/besticon"
	"github.com/spf13/cobra"
	"gopkg.in/cheggaaa/pb.v1"
)

var (
	repairFaviconsCmd = &cobra.Command{
		Use:     "repair-favicons",
		Short:   "Repair favicons in places.sqlite",
		Example: AppName + ` repair-favicons "/home/user/places.sqlite" "/home/user/favicons.sqlite"`,
		Run:     repairFavicons,
	}
	enableIconCache bool
)

const (
	repairIconsPages = 100
)

type repairFaviconsResult struct {
	lastIconsID       int
	lastPagesWIconsID int
	countValid        int
	countRepaired     int
	countSkipped      int
	countExist        int
	countErrors       int
}

func init() {
	repairFaviconsCmd.PersistentFlags().BoolVar(&enableIconCache, "enable-cache", false, "Enable icon cache")
	RootCmd.AddCommand(repairFaviconsCmd)
}

func repairFavicons(cmd *cobra.Command, args []string) {
	var err error
	var result repairFaviconsResult

	// check args
	if len(args) > 2 {
		Logger.Crit("repair-favicons has too many arguments")
	}
	if len(args) != 2 {
		Logger.Crit("repair-favicons requires places.sqlite and favicons.sqlite files")
	}
	if !FileExists(args[0]) {
		Logger.Critf("%s not found", args[0])
	}
	if !FileExists(args[1]) {
		Logger.Critf("%s not found", args[1])
	}

	// check and open dbs
	Logger.Printf("Checking and opening DBs...")
	masterPlacesDb, masterFaviconsDb, err = sqlite.OpenDbs(sqlite.SqliteFiles{
		Places: args[0], Favicons: args[1],
	}, true)
	if err != nil {
		Logger.Crit(err)
	}

	// besticon settings
	besticon.SetLogOutput(ioutil.Discard)
	if enableIconCache {
		besticon.SetCacheMaxSize(128)
	}

	// https://github.com/golang/go/issues/19895
	log.SetOutput(ioutil.Discard)

	// backup dbs
	sqlite.BackupDb(masterPlacesDb.Info)
	sqlite.BackupDb(masterFaviconsDb.Info)

	// get moz_places count
	var mozPlacesCount int
	masterPlacesDb.Link.Model(places.MozPlaces{}).Count(&mozPlacesCount)
	Logger.Printf("\nPlaces to check:           %d", mozPlacesCount)

	// get last moz_icons.id
	result.lastIconsID = favicons.MozIcons{}.GetLastID(masterFaviconsDb.Link)
	Logger.Printf("Last moz_icons.id:         %d", result.lastIconsID)

	// get last moz_pages_w_icons.id
	result.lastPagesWIconsID = favicons.MozPagesWIcons{}.GetLastID(masterFaviconsDb.Link)
	Logger.Printf("Last moz_pages_w_icons.id: %d", result.lastPagesWIconsID)

	// get first moz_places.id
	firstPlacesID := places.MozPlaces{}.GetFirstID(masterPlacesDb.Link)

	// start repair
	pageSize := int(math.Ceil(float64(mozPlacesCount / repairIconsPages)))

	Logger.Debugf("\nPaginate moz_places:")
	Logger.Debugf("- total rows: %s", mozPlacesCount)
	Logger.Debugf("- first id:   %s", firstPlacesID)
	Logger.Debugf("- pages:      %s", repairIconsPages)
	Logger.Debugf("- page size:  %s", pageSize)

	Logger.Printf("\n## Repairing favicons...")
	progBar := pb.StartNew(mozPlacesCount)
	progBar.Prefix("moz_icons")

	lastPlacesID := 0
	for i := 0; i <= repairIconsPages; i++ {
		var mozPlaces []places.MozPlaces
		if lastPlacesID == 0 {
			masterPlacesDb.Link.Order("id ASC").Limit(pageSize).Find(&mozPlaces)
		} else {
			masterPlacesDb.Link.Where("id > ?", lastPlacesID).Order("id ASC").Limit(pageSize).Find(&mozPlaces)
		}
		if len(mozPlaces) == 0 {
			continue
		}
		lastPlacesID = mozPlaces[len(mozPlaces)-1].ID

		for _, mozPlace := range mozPlaces {
			var (
				mozIcon              favicons.MozIcons
				foundExistingMozIcon favicons.MozIcons
				mozPageWIcon         favicons.MozPagesWIcons
				mozIconToPage        favicons.MozIconsToPages
			)

			// seek icon
			masterFaviconsDb.Link.First(&mozIcon, mozPlace.FaviconId)

			// valid
			if mozIcon.IconUrl != "" {
				result.countValid++
				progBar.Increment()
				continue
			}

			// skip invalid URL
			host, _, err := sqlite.FixupUrl(mozPlace.Url)
			if err != nil || Contains([]string{"localhost", "127.0.0.1"}, host) || strings.HasPrefix(mozPlace.Url, "moz-extension://") || !strings.HasPrefix(mozPlace.Url, "http://") && !strings.HasPrefix(mozPlace.Url, "https://") {
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

			mozIcon.IconUrl = ico.URL
			mozIcon.FixedIconUrlHash = 0 //TODO: Fills with 0 temporarily
			mozIcon.Width = int64(ico.Width)
			mozIcon.Root = 0 //TODO: Fills with 0 temporarily
			mozIcon.ExpireMs = 0
			mozIcon.Data = ico.ImageData

			// check if found existing favicon
			masterFaviconsDb.Link.Where("icon_url = ?", mozIcon.IconUrl).First(&foundExistingMozIcon)
			if foundExistingMozIcon.IconUrl != "" {
				mozPlace.FaviconId = foundExistingMozIcon.ID
				result.countExist++
			} else {
				result.lastIconsID++
				mozIcon.ID = result.lastIconsID
				if err := masterFaviconsDb.Link.Create(&mozIcon).Error; err != nil {
					Logger.Errorf("Creating moz_icons row with id=%d : %s", mozIcon.ID, err)
					result.countErrors++
					progBar.Increment()
					continue
				}

				mozIconToPage.PageId = mozPlace.ID
				mozIconToPage.IconId = mozIcon.ID
				if err := masterFaviconsDb.Link.Create(&mozIconToPage).Error; err != nil {
					Logger.Errorf("Creating moz_icons_to_pages row with page_id=%d and icon_id=%d : %s", mozIconToPage.PageId, mozIconToPage.IconId, err)
					result.countErrors++
					progBar.Increment()
					continue
				}

				result.lastPagesWIconsID++
				mozPageWIcon.ID = result.lastPagesWIconsID
				mozPageWIcon.PageUrl = mozPlace.Url
				mozPageWIcon.PageUrlHash = 0 //TODO: Fills with 0 temporarily
				if err := masterFaviconsDb.Link.Create(&mozPageWIcon).Error; err != nil {
					Logger.Errorf("Creating moz_pages_w_icons row with id=%d : %s", mozPageWIcon.ID, err)
					result.countErrors++
					progBar.Increment()
					continue
				}

				mozPlace.FaviconId = result.lastIconsID
				result.countRepaired++
			}

			// update places
			if err := masterPlacesDb.Link.Save(&mozPlace).Error; err != nil {
				result.countRepaired++
				Logger.Errorf("Updating moz_places row with id=%d : %s", mozPlace.ID, err)
			}

			progBar.Increment()
		}
	}

	progBar.Finish()

	Logger.Printf("\nResult")
	Logger.Printf("  valid    = %d", result.countValid)
	Logger.Printf("  repaired = %d", result.countRepaired)
	Logger.Printf("  skipped  = %d", result.countSkipped)
	Logger.Printf("  exist    = %d", result.countExist)
	Logger.Printf("  errors   = %d", result.countErrors)

	Logger.Printf("\nOptimizing %s database...", masterPlacesDb.Info.Filename)
	if err = sqlite.Vacuum(masterPlacesDb.Link); err != nil {
		Logger.Warn(err)
	}

	Logger.Printf("Optimizing %s database...", masterFaviconsDb.Info.Filename)
	if err = sqlite.Vacuum(masterFaviconsDb.Link); err != nil {
		Logger.Warn(err)
	}

	masterPlacesDb.Link.Close()
	masterFaviconsDb.Link.Close()
}
