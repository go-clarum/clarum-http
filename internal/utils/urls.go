package utils

import (
	"fmt"
	clarumstrings "github.com/goclarum/clarum/core/validators/strings"
	"net/url"
)

func IsValidUrl(urlToCheck string) bool {
	parsedUrl, err := url.Parse(urlToCheck)
	return err == nil && clarumstrings.IsNotBlank(parsedUrl.Scheme) &&
		clarumstrings.IsNotBlank(parsedUrl.Host)
}

func BuildPath(base string, pathElements ...string) string {
	path, err := url.JoinPath(base, pathElements...)

	if err != nil {
		panic(fmt.Sprintf("Error while building path: %s", err))
	} else {
		return path
	}
}
