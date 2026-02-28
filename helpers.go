package main

import "math/rand"

// shuffleStrings shuffles a slice of strings in-place.
func shuffleStrings(strings []string) {
	for i := range strings {
		j := rand.Intn(i + 1)
		strings[i], strings[j] = strings[j], strings[i]
	}
}
