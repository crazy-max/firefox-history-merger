package sqlite

import (
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/crazy-max/firefox-history-merger/sqlite/table"
	"github.com/crazy-max/firefox-history-merger/utils"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Info struct {
	Filename       string
	Filehash       string
	Version        int
	FirefoxVersion int
	HistoryCount   int
	PlacesCount    int
	LastUsedUnix   int64
	LastUsedTime   time.Time
}

type Db struct {
	Link *gorm.DB
	Info Info
}

type Dbs []Db

var DbSchema = 39

func OpenDir(dirname string, workingDb Db) Dbs {
	var dbs Dbs

	files, err := ioutil.ReadDir(dirname)
	utils.CheckIfError(err)

	for _, file := range files {
		if file.IsDir() || path.Ext(file.Name()) != ".sqlite" {
			continue
		}
		db := OpenFile(filepath.Join(dirname, file.Name()))
		if db.Info.Filehash != workingDb.Info.Filehash {
			dbs = append(dbs, db)
		}
	}

	sort.Slice(dbs, func(i, j int) bool {
		return dbs[i].Info.LastUsedUnix > dbs[j].Info.LastUsedUnix
	})

	return dbs
}

func OpenFile(file string) Db {
	utils.CheckFileExists(file)

	fileInfo, err := os.Stat(file)
	utils.CheckIfError(err)

	filehash, err := utils.GetHash(file)
	utils.CheckIfError(err)

	link, err := gorm.Open("sqlite3", file)
	utils.CheckIfError(err)

	var dbInfo Info
	dbInfo.Version = getDbVersion(link)
	dbInfo.FirefoxVersion = getFirefoxVersion(dbInfo.Version)
	dbInfo.Filename = fileInfo.Name()
	dbInfo.Filehash = filehash

	link.Model(&table.MozPlaces{}).Count(&dbInfo.PlacesCount)

	row := link.Model(&table.MozHistoryvisits{}).Select("count(id), visit_date").Order("visit_date desc").Row()
	row.Scan(&dbInfo.HistoryCount, &dbInfo.LastUsedUnix)
	dbInfo.LastUsedTime = time.Unix(int64(math.Ceil(float64(dbInfo.LastUsedUnix/1000000))), 0)

	return Db{
		Link: link,
		Info: dbInfo,
	}
}

// https://dxr.mozilla.org/mozilla-central/source/toolkit/modules/Sqlite.jsm#1175
func getDbVersion(link *gorm.DB) int {
	row := link.Raw("PRAGMA user_version").Row()
	if row == nil {
		link.Close()
		utils.ErrorExit("PRAGMA user_version not found")
	}

	var dbVersion int
	if err := row.Scan(&dbVersion); err != nil {
		link.Close()
		utils.ErrorExit(err.Error())
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
	return 57
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
