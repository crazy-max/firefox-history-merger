# Changelog

## 1.57.2 (2018/01/27)

* Mozilla has moved the old `moz_favicons` table to the `favicons.sqlite` db
* Add command `optimize` to optimize a database into a minimal amount of disk space
* Make icon cache optional and use flag `--enable-cache` to enable it (Issue #2)
* Goroutine can cause a database lockup on Linux
* Put more explicit messages when checking arguments (Issue #1)
* Check database compatibility using schema version (disable auto migration)
* Base semantic versioning on Firefox version
* Prepare implementation for Firefox 58
* New logger
* Update libs

## 0.1.1 (2017/12/20)

* Create artifacts for Linux and macOS

## 0.1.0 (2017/12/10)

* Initial version
