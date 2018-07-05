# Changelog

## 1.61.0 (2018/07/05)

* Add implementation for Firefox 61 schema version v47 (Issue #6)

## 1.60.0 (2018/05/11)

* Add implementation for Firefox 60 schema version v43

## 1.59.0 (2018/03/15)

* Same schema (v41) for Firefox 59 (Issue #4)

## 1.58.0 (2018/01/27)

* Add implementation for Firefox 58 schema version v41

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
