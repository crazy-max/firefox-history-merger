package utl

import (
	"math/rand"
	"net/url"
	"strings"
)

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// https://dxr.mozilla.org/mozilla-central/source/toolkit/components/places/Helpers.cpp#243
func GenerateGUID() string {
	var length = 12
	var letterRunes = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_")

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
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
