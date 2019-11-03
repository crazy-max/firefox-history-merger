package app

import (
	"io/ioutil"
	golog "log"
	"strings"
	"sync"

	"github.com/crazy-max/firefox-history-merger/internal/db/favicons"
	"github.com/crazy-max/firefox-history-merger/internal/db/places"
	"github.com/crazy-max/firefox-history-merger/internal/utl"
	"github.com/mat/besticon/besticon"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

type repairFaviconsResult struct {
	countValid    int
	countRepaired int
	countSkipped  int
	countErrors   int
}

type repairFaviconsJob struct {
	placesCli   *places.Client
	faviconsCli *favicons.Client
	places      places.MozPlaces
	result      *repairFaviconsResult
	left        *int
}

// RepairFavicons repairs broken favicons
func (fhm *FirefoxHistoryMerger) RepairFavicons() {
	result := repairFaviconsResult{}
	besticon.SetLogOutput(ioutil.Discard)
	besticon.SetCacheMaxSize(128)

	// https://github.com/golang/go/issues/19895
	golog.SetOutput(ioutil.Discard)

	log.Debug().Msgf("Opening %s places database...", fhm.fl.PlacesFile)
	placesCli, err := places.New(fhm.fl.PlacesFile, true)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open database %s", fhm.fl.PlacesFile)
	}
	fhm.dbs = append(fhm.dbs, placesCli.Db)
	defer placesCli.Db.Close()

	log.Debug().Msgf("Opening %s favicons database...", fhm.fl.FaviconsFile)
	faviconsCli, err := favicons.New(fhm.fl.FaviconsFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open database %s", fhm.fl.FaviconsFile)
	}
	fhm.dbs = append(fhm.dbs, faviconsCli.Db)
	defer faviconsCli.Db.Close()
	if err := faviconsCli.Db.Backup(); err != nil {
		log.Warn().Err(err).Msgf("Cannot backup database %s", faviconsCli.Db.Filename)
	}

	var placesl []places.MozPlaces
	placesCli.Db.Order("id ASC").Find(&placesl)

	left := len(placesl)
	log.Info().Msgf("Checking %d places...", left)

	var wg sync.WaitGroup
	pool, _ := ants.NewPoolWithFunc(fhm.fl.Workers, func(i interface{}) {
		fhm.jobRepairFavicons(i)
		wg.Done()
	})
	defer pool.Release()

	for _, place := range placesl {
		wg.Add(1)
		err := pool.Invoke(repairFaviconsJob{
			placesCli:   placesCli,
			faviconsCli: faviconsCli,
			places:      place,
			left:        &left,
			result:      &result,
		})
		if err != nil {
			log.Error().Err(err).Msgf("Invoking job")
		}
	}
	wg.Wait()

	log.Info().
		Int("total", len(placesl)).
		Int("valid", result.countValid).
		Int("repaired", result.countRepaired).
		Int("skipped", result.countSkipped).
		Int("errors", result.countErrors).
		Msg("Finished")
}

func (fhm *FirefoxHistoryMerger) jobRepairFavicons(i interface{}) {
	job := i.(repairFaviconsJob)
	var icon favicons.MozIcons
	defer func() {
		*job.left--
	}()

	// Skip invalid URL
	host, _, err := utl.FixupUrl(job.places.Url)
	if err != nil || utl.Contains([]string{"localhost", "127.0.0.1"}, host) ||
		strings.HasPrefix(job.places.Url, "moz-extension://") ||
		!strings.HasPrefix(job.places.Url, "http://") &&
			!strings.HasPrefix(job.places.Url, "https://") {
		job.result.countSkipped++
		log.Debug().
			Int("places_id", job.places.ID).
			Int("left", *job.left).
			Str("url", job.places.Url).
			Msg("Favicon skipped due to invalid places url")
		return
	}

	// Seek icon
	job.faviconsCli.Db.Table("moz_icons").
		Joins("JOIN moz_icons_to_pages ON moz_icons_to_pages.icon_id = moz_icons.id").
		Where("moz_icons_to_pages.page_id = ?", job.places.ID).
		First(&icon)

	// Valid favicon
	if icon.IconUrl != "" {
		log.Debug().
			Int("places_id", job.places.ID).
			Int("favicon_id", icon.ID).
			Int("left", *job.left).
			Msg("Favicon valid")
		job.result.countValid++
		return
	}

	// Retrieve favicon
	ico, err := utl.GetFavicon(job.places.Url)
	if err != nil {
		job.result.countErrors++
		log.Error().Err(err).
			Int("places_id", job.places.ID).
			Int("favicon_id", icon.ID).
			Int("left", *job.left).
			Str("url", job.places.Url).
			Msg("Cannot get favicon")
		return
	}

	icon.IconUrl = ico.URL
	icon.FixedIconUrlHash = 0 //TODO: Fills with 0 temporarily
	icon.Width = int64(ico.Width)
	icon.Root = 0 //TODO: Fills with 0 temporarily
	icon.ExpireMs = 0
	icon.Data = ico.ImageData

	// Otherwise create favicon entry
	job.faviconsCli.Db.NewRecord(icon)
	if err := job.faviconsCli.Db.Create(&icon).Error; err != nil {
		job.result.countErrors++
		log.Error().Err(err).
			Int("places_id", job.places.ID).
			Int("favicon_id", icon.ID).
			Int("left", *job.left).
			Str("url", job.places.Url).
			Msg("Creating moz_icons row")
		return
	}

	var iconToPage favicons.MozIconsToPages
	iconToPage.PageId = job.places.ID
	iconToPage.IconId = icon.ID
	job.faviconsCli.Db.NewRecord(iconToPage)
	if err := job.faviconsCli.Db.Create(&iconToPage).Error; err != nil {
		job.result.countErrors++
		log.Error().Err(err).
			Int("places_id", job.places.ID).
			Int("favicon_id", icon.ID).
			Int("left", *job.left).
			Str("url", job.places.Url).
			Msg("Creating moz_icons_to_pages row")
		return
	}

	var pageWIcon favicons.MozPagesWIcons
	pageWIcon.PageUrl = job.places.Url
	pageWIcon.PageUrlHash = 0 //TODO: Fills with 0 temporarily
	job.faviconsCli.Db.NewRecord(pageWIcon)
	if err := job.faviconsCli.Db.Create(&pageWIcon).Error; err != nil {
		job.result.countErrors++
		log.Error().Err(err).
			Int("places_id", job.places.ID).
			Int("favicon_id", icon.ID).
			Int("left", *job.left).
			Str("url", job.places.Url).
			Msg("Creating moz_pages_w_icon row")
		return
	}

	job.result.countRepaired++
	log.Info().
		Int("places_id", job.places.ID).
		Int("favicon_id", icon.ID).
		Int("left", *job.left).
		Str("url", job.places.Url).
		Msg("Favicon repaired")
}
