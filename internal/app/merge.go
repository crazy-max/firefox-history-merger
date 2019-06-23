package app

import (
	"github.com/crazy-max/firefox-history-merger/internal/db/places"
	"github.com/crazy-max/firefox-history-merger/internal/utl"
	"github.com/jinzhu/gorm"
	"github.com/rs/zerolog/log"
	"github.com/vcraescu/go-paginator"
	"github.com/vcraescu/go-paginator/adapter"
)

type mergeClient struct {
	pCli         *places.Client
	pTx          *gorm.DB
	ptmCli       *places.Client
	mpLeft       int
	countCreated int
	countUpdated int
	countErrors  int
}

// Merge a working places databases with other ones
func (fhm *FirefoxHistoryMerger) Merge() {
	var err error

	log.Debug().Msgf("Opening working places database: %s...", fhm.fl.PlacesFile)
	pCli, err := places.New(fhm.fl.PlacesFile, true)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open database %s", fhm.fl.PlacesFile)
	}
	fhm.dbs = append(fhm.dbs, pCli.Db)
	defer pCli.Db.Close()
	if err := pCli.Db.Backup(); err != nil {
		log.Warn().Err(err).Msgf("Cannot backup database %s", pCli.Db.Filename)
	}

	log.Debug().Msgf("Opening places database to merge: %s...", fhm.fl.PlacesToMergeFile)
	ptmCli, err := places.New(fhm.fl.PlacesToMergeFile, true)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open database %s", fhm.fl.PlacesToMergeFile)
	}
	fhm.dbs = append(fhm.dbs, ptmCli.Db)
	defer ptmCli.Db.Close()

	// Create merge client
	mc := &mergeClient{
		pCli:   pCli,
		ptmCli: ptmCli,
	}

	// Paginate result
	var ptmPlacesList []places.MozPlaces
	ptmPlacesPage := paginator.New(adapter.NewGORMAdapter(ptmCli.Db.Model(places.MozPlaces{})), fhm.fl.MaxPerTx)
	if err = ptmPlacesPage.Results(&ptmPlacesList); err != nil {
		log.Fatal().Err(err).Msg("Cannot get pagination places")
	}

	// Count places to merge
	mc.mpLeft = ptmPlacesPage.Nums()
	log.Info().Msgf("%d places will be merged", ptmPlacesPage.Nums())

	// Iterate
	for {
		tx := mc.pCli.Db.Begin()
		log.Info().Msgf("Merging %d places (%d/%d)...", len(ptmPlacesList), ptmPlacesPage.Page(), ptmPlacesPage.PageNums())
		for _, ptmPlaces := range ptmPlacesList {
			pPlaces := mc.mergePlaces(ptmPlaces, tx)
			if pPlaces.ID == 0 {
				continue
			}
			mc.mergeHistoryvisits(pPlaces, ptmPlaces, tx)
		}
		tx.Commit()
		log.Debug().Msg("Transaction committed")

		if !ptmPlacesPage.HasNext() {
			break
		}
		nextPage, err := ptmPlacesPage.NextPage()
		if err != nil {
			log.Fatal().Err(err).Msg("Cannot reached places next page")
		}
		ptmPlacesPage.SetPage(nextPage)
		if err := ptmPlacesPage.Results(&ptmPlacesList); err != nil {
			log.Fatal().Err(err).Msg("Cannot get pagination places")
		}
	}

	log.Info().Msg("Optimizing database...")
	if err := mc.pCli.Db.Vacuum(); err != nil {
		log.Warn().Err(err).Msg("Cannot optimize database")
	}

	log.Info().
		Int("total", ptmPlacesPage.Nums()).
		Int("created", mc.countCreated).
		Int("updated", mc.countUpdated).
		Int("errors", mc.countErrors).
		Msg("Finished")
}

func (mc *mergeClient) mergePlaces(ptmPlaces places.MozPlaces, pTx *gorm.DB) places.MozPlaces {
	defer func() {
		mc.mpLeft--
	}()

	var pPlaces places.MozPlaces
	pTx.Where("url = ?", ptmPlaces.Url).Find(&pPlaces)

	// New entry
	if pPlaces.ID == 0 {
		pPlaces = ptmPlaces
		pPlaces.ID = 0
		pPlaces.Guid = placesNewGUID(pTx)
		pTx.NewRecord(pPlaces)
		if err := pTx.Create(&pPlaces).Error; err != nil {
			mc.countErrors++
			log.Error().Err(err).Int("places_id", pPlaces.ID).
				Int("mplaces_id", ptmPlaces.ID).
				Int("left", mc.mpLeft).
				Str("url", ptmPlaces.Url).Msg("Creating moz_places row")
			return places.MozPlaces{}
		}

		mc.countCreated++
		log.Debug().Int("places_id", pPlaces.ID).
			Int("mplaces_id", ptmPlaces.ID).
			Int("left", mc.mpLeft).
			Str("url", ptmPlaces.Url).Msg("Creating moz_places row")
		return pPlaces
	}

	// Update entry
	pPlaces.VisitCount += ptmPlaces.VisitCount
	pPlaces.LastVisitDate = utl.MaxInt64(pPlaces.LastVisitDate, ptmPlaces.LastVisitDate)
	pPlaces.Frecency = (pPlaces.Frecency + ptmPlaces.Frecency) / 2
	if err := pTx.Save(&pPlaces).Error; err != nil {
		mc.countErrors++
		log.Error().Err(err).Int("places_id", pPlaces.ID).
			Int("mplaces_id", ptmPlaces.ID).
			Int("left", mc.mpLeft).
			Str("url", ptmPlaces.Url).Msg("Updating moz_places row")
		return places.MozPlaces{}
	}

	mc.countUpdated++
	log.Debug().Int("places_id", pPlaces.ID).
		Int("mplaces_id", ptmPlaces.ID).
		Int("left", mc.mpLeft).
		Str("url", ptmPlaces.Url).Msg("Updating moz_places row")
	return pPlaces
}

func placesNewGUID(pTx *gorm.DB) string {
	var found int
	var table places.MozPlaces
	guid := utl.GenerateGUID()
	pTx.Model(table).Where("guid = ?", guid).Count(&found)
	if found > 0 {
		guid = placesNewGUID(pTx)
	}
	return guid
}

func (mc *mergeClient) mergeHistoryvisits(pPlaces places.MozPlaces, ptmPlaces places.MozPlaces, pTx *gorm.DB) {
	var ptmHistoryvisitsList []places.MozHistoryvisits

	mc.ptmCli.Db.Where("place_id = ?", ptmPlaces.ID).Find(&ptmHistoryvisitsList)
	for _, ptmHistoryvisits := range ptmHistoryvisitsList {
		// Find if already exists
		var pHistoryvisitsList places.MozHistoryvisits
		pTx.Where("from_visit = ? AND place_id = ? AND visit_date = ?",
			ptmHistoryvisits.FromVisit,
			pPlaces.ID,
			ptmHistoryvisits.VisitDate,
		).First(&pHistoryvisitsList)

		// Leave if found
		if pHistoryvisitsList.ID > 0 {
			continue
		}

		ptmHistoryvisits.ID = 0
		ptmHistoryvisits.PlaceId = pPlaces.ID
		ptmHistoryvisits.FromVisit = 0 //TODO: Find a way to retrieve ancestors. Fills from_visit with 0 temporarily.
		pTx.NewRecord(ptmHistoryvisits)
		if err := pTx.Create(&ptmHistoryvisits).Error; err != nil {
			log.Error().Err(err).Int("places_id", pPlaces.ID).
				Int("mplaces_id", ptmPlaces.ID).
				Int("historyvisits_id", ptmHistoryvisits.ID).
				Msg("Creating moz_places row")
			continue
		}

		log.Debug().Int("places_id", pPlaces.ID).
			Int("mplaces_id", ptmPlaces.ID).
			Int("historyvisits_id", ptmHistoryvisits.ID).
			Msg("Creating moz_historyvisits row")
	}
}
