package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/crazy-max/firefox-history-merger/logger"
	"github.com/crazy-max/firefox-history-merger/sqlite/favicons"
	"github.com/crazy-max/firefox-history-merger/sqlite/places"
	"github.com/crazy-max/firefox-history-merger/utils"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type (
	Info struct {
		Filepath       string
		Filename       string
		Filehash       string
		Version        int
		FirefoxVersion int
		HistoryCount   int
		PlacesCount    int
		IconsCount     int
		LastUsedUnix   int64
		LastUsedTime   time.Time
		Compatible     bool
		CompatibleStr  string
	}
	SqliteFiles struct {
		Places   string
		Favicons string
	}
	PlacesDb struct {
		Struct interface{}
		Link   *gorm.DB
		Info   Info
	}
	FaviconsDb struct {
		Struct interface{}
		Link   *gorm.DB
		Info   Info
	}
	PlacesDbs   []PlacesDb
	FaviconsDbs []FaviconsDb
)

var (
	SchemaVersion   = 39
	SchemaFFVersion = getFirefoxVersion(SchemaVersion)
)

func OpenPlacesDir(dirname string, workingPlacesDb PlacesDb) PlacesDbs {
	var placesDbs PlacesDbs

	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		logger.Crit(err)
	}

	for _, file := range files {
		if file.IsDir() || path.Ext(file.Name()) != ".sqlite" {
			continue
		}
		aPlacesDb, _, err := OpenDbs(SqliteFiles{Places: filepath.Join(dirname, file.Name())}, false)
		if err != nil {
			utils.Logger.Crit(err)
		} else if aPlacesDb.Info.Filehash != workingPlacesDb.Info.Filehash {
			placesDbs = append(placesDbs, aPlacesDb)
		}
	}

	sort.Slice(placesDbs, func(i, j int) bool {
		return placesDbs[i].Info.LastUsedUnix > placesDbs[j].Info.LastUsedUnix
	})

	return placesDbs
}

func Open(file string) (db *gorm.DB, err error) {
	if _, err = filepath.Abs(file); err != nil {
		return db, err
	}

	if _, err = os.Stat(file); err != nil {
		if err != nil {
			return db, err
		}
	}

	return gorm.Open("sqlite3", file)
}

func OpenDbs(sqliteFiles SqliteFiles, checkCompat bool) (placesDb PlacesDb, faviconsDb FaviconsDb, err error) {
	var (
		placesFileInfo   os.FileInfo
		placesHvRow      *sql.Row
		faviconsFileInfo os.FileInfo
	)

	if placesDb.Info.Filepath, err = filepath.Abs(sqliteFiles.Places); err != nil {
		return placesDb, faviconsDb, err
	}

	placesFileInfo, err = os.Stat(sqliteFiles.Places)
	if err != nil {
		return placesDb, faviconsDb, err
	}

	placesFilehash, err := utils.GetHash(sqliteFiles.Places)
	if err != nil {
		return placesDb, faviconsDb, err
	}

	placesDb.Link, err = gorm.Open("sqlite3", sqliteFiles.Places)
	if err != nil {
		return placesDb, faviconsDb, err
	}

	placesDb.Info.Version = getDbVersion(placesDb)
	placesDb.Info.FirefoxVersion = getFirefoxVersion(placesDb.Info.Version)
	placesDb.Info.Filename = placesFileInfo.Name()
	placesDb.Info.Filehash = placesFilehash
	placesDb.Info.Compatible = placesDb.Info.Version == SchemaVersion
	placesDb.Info.CompatibleStr = "YES"
	if !placesDb.Info.Compatible {
		placesDb.Info.CompatibleStr = "NO"
	}

	if checkCompat && placesDb.Info.Version != SchemaVersion {
		return placesDb, faviconsDb, errors.New(fmt.Sprintf("Your database schema v%d is not compatible with the current implementation (Firefox %d)", placesDb.Info.Version, SchemaFFVersion))
	}

	placesDb.Link.Model(&places.MozPlaces{}).Count(&placesDb.Info.PlacesCount)
	placesHvRow = placesDb.Link.Model(&places.MozHistoryvisits{}).Select("count(id), visit_date").Order("visit_date desc").Row()

	if placesHvRow != nil {
		placesHvRow.Scan(&placesDb.Info.HistoryCount, &placesDb.Info.LastUsedUnix)
		placesDb.Info.LastUsedTime = time.Unix(int64(math.Ceil(float64(placesDb.Info.LastUsedUnix/1000000))), 0)
	}

	//TODO: Display info about missing favicons DB ?
	if faviconsDb.Info.Filepath, err = filepath.Abs(sqliteFiles.Favicons); err != nil {
		return placesDb, faviconsDb, nil
	}

	faviconsFileInfo, err = os.Stat(sqliteFiles.Favicons)
	if err != nil {
		return placesDb, faviconsDb, nil
	}

	faviconsFilehash, err := utils.GetHash(sqliteFiles.Favicons)
	if err != nil {
		return placesDb, faviconsDb, nil
	}

	faviconsDb.Link, err = gorm.Open("sqlite3", sqliteFiles.Favicons)
	if err != nil {
		return placesDb, faviconsDb, nil
	}

	faviconsDb.Info.Filename = faviconsFileInfo.Name()
	faviconsDb.Info.Filehash = faviconsFilehash

	faviconsDb.Link.Model(&favicons.MozIcons{}).Count(&faviconsDb.Info.IconsCount)

	return placesDb, faviconsDb, nil
}

func Vacuum(db *gorm.DB) (err error) {
	if err = db.Exec("VACUUM").Error; err != nil {
		return errors.New(fmt.Sprintf("Cannot optimize database: %s", err.Error()))
	}
	return nil
}

// https://dxr.mozilla.org/mozilla-central/source/toolkit/modules/Sqlite.jsm#1175
func getDbVersion(placesDb PlacesDb) int {
	row := placesDb.Link.Raw("PRAGMA user_version").Row()
	if row == nil {
		placesDb.Link.Close()
		logger.Crit("PRAGMA user_version not found")
	}

	var dbVersion int
	if err := row.Scan(&dbVersion); err != nil {
		placesDb.Link.Close()
		logger.Crit(err)
	}

	return dbVersion
}

// https://dxr.mozilla.org/mozilla-central/source/toolkit/components/places/Database.cpp#974
func getFirefoxVersion(dbVersion int) int {
	if dbVersion < 11 {
		return 3
	}
	if dbVersion < 12 {
		return 4
	}
	if dbVersion < 13 {
		return 8
	}
	if dbVersion < 18 {
		return 12
	}
	if dbVersion < 20 {
		return 13
	}
	if dbVersion < 22 {
		return 14
	}
	if dbVersion < 23 {
		return 22
	}
	if dbVersion < 24 {
		return 24
	}
	if dbVersion < 25 {
		return 34
	}
	if dbVersion < 26 {
		return 36
	}
	if dbVersion < 27 {
		return 37
	}
	if dbVersion < 30 {
		return 39
	}
	if dbVersion < 31 {
		return 41
	}
	if dbVersion < 32 {
		return 48
	}
	if dbVersion < 33 {
		return 49
	}
	if dbVersion < 34 {
		return 50
	}
	if dbVersion < 35 {
		return 51
	}
	if dbVersion < 37 {
		return 52
	}
	if dbVersion < 38 {
		return 55
	}
	if dbVersion < 39 {
		return 56
	}
	if dbVersion < 41 {
		return 57
	}
	if dbVersion == 41 {
		return 58
	}
	return -1
}

func BackupDb(dbInfo Info) error {
	dbFileBackup := fmt.Sprintf("%s.%s", dbInfo.Filepath, time.Now().Format("20060102150405"))
	utils.Logger.Printf("Backing up '%s' to '%s'", dbInfo.Filename, filepath.Base(dbFileBackup))
	return utils.CopyFile(dbInfo.Filepath, dbFileBackup)
}

// https://dxr.mozilla.org/mozilla-central/source/toolkit/components/places/SQLFunctions.cpp#868
func FixupUrl(ustr string) (host string, prefix string, err error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return "", "", err
	}

	schemes := []string{"https", "ftp"}
	for _, scheme := range schemes {
		if u.Scheme == scheme {
			prefix = u.Scheme + "://"
			break
		}
	}

	if strings.HasPrefix(u.Host, "www.") {
		prefix += "www."
		u.Host = strings.TrimPrefix(u.Host, "www.")
	}

	return u.Host, prefix, nil
}

// https://dxr.mozilla.org/mozilla-beta/source/toolkit/components/places/SQLFunctions.cpp#1014
func Hash(text string) {
	maxCharsToHash := 1500
	if len(text) > maxCharsToHash {
		text = text[0 : maxCharsToHash-1]
	}
	//TODO: continue...
}
