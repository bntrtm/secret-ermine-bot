package main

import (
	"strings"
)

// `shim.go` exists to provide helpers exclusively used to wrap
// RevoltGo functionalities to fix known upstream bugs.

// singularizeRoute wraps a Stoat API route to replace plural
// terms with singular counterparts.
// The Stoat API uses singular routes.
func singularizeStoatRoute(route string) string {
	singularized := route
	singularizeMap := map[string]string{
		"servers":  "server",
		"channels": "channel",
		"message":  "message",
	}
	for k, v := range singularizeMap {
		singularized = strings.ReplaceAll(singularized, k, v)
	}
	return singularized
}
