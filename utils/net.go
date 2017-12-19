package utils

import (
	"errors"
	"fmt"

	"github.com/mat/besticon/besticon"
)

func GetFavicon(url string) (ico besticon.Icon, err error) {
	var result besticon.Icon

	finder := besticon.IconFinder{}
	icons, err := finder.FetchIcons(url)
	if err != nil {
		return result, err
	} else if len(icons) == 0 {
		return result, errors.New(fmt.Sprintf("no favicon found on %s", url))
	}

	for _, ico := range icons {
		if ico.Width == 32 || ico.Width == 16 {
			result = ico
			break
		}
	}
	if result.URL == "" {
		result = icons[len(icons)-1]
	}

	return result, nil
}
