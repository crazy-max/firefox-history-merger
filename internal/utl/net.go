package utl

import (
	"errors"

	"github.com/mat/besticon/besticon"
)

func GetFavicon(url string) (besticon.Icon, error) {
	finder := besticon.IconFinder{}
	icons, err := finder.FetchIcons(url)
	if err != nil {
		return besticon.Icon{}, err
	} else if len(icons) == 0 {
		return besticon.Icon{}, errors.New("no favicon found")
	}

	for _, ico := range icons {
		if ico.Error == nil && ico.URL != "" && ico.Width <= 256 {
			return ico, nil
		}
	}

	return besticon.Icon{}, errors.New("no valid favicon found")
}
