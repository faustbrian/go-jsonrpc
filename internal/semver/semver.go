package semver

import (
	"errors"
	"regexp"
)

var tagPattern = regexp.MustCompile(
	`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)` +
		`(-((0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)` +
		`(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?` +
		`(\+([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?$`,
)

func ValidateTag(tag string) error {
	if !tagPattern.MatchString(tag) {
		return errors.New("tag must be a v-prefixed semantic version")
	}
	return nil
}
