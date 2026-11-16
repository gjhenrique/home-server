package slopper

import "regexp"

var appLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,30}[a-z0-9])?$`)
var resourceName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,23}$`)

func ValidApp(name string) bool {
	return appLabel.MatchString(name) && name != "api" && name != "admin"
}
func ValidResource(name string) bool { return resourceName.MatchString(name) }
