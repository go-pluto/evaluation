package utils

import (
	"strings"
	"math/rand"
)

// Functions

// GenerateString returns a random string from the alphabet [a-z,0-9] of the
// length "strlen"
func GenerateString(strlen int) string {
	// Define alphabet
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

	result := ""
	for i := 0; i < strlen; i++ {
		index := rand.Intn(len(chars))
		result += chars[index : index+1]
	}
	return result
}

// GenerateString returns a random string from the alphabet [a-z,0-9] of the
// length "strlen"
func GenerateFlags() (string, []string) {
	// Define alphabet
	flags := [...]string{"\\Seen", "\\Answered", "\\Flagged", "\\Deleted", "\\Draft"}

	numflags := 1 + rand.Intn(len(flags))

	// Generate an array of random but different indicies of length "numflags"
	var genindex []int
	for len(genindex) < numflags{
		index := rand.Intn(len(flags))
		for i := 0; i < len(genindex); i++ {
			if index == genindex[i] {
				index = rand.Intn(len(flags))
				i = -1
			}
		}
		genindex = append(genindex, index)
	}

	// Add the corresponding flag of the previously generated index to the
	// string array "genflags".
	var genflags []string
	for i := 0; i < len(genindex); i++ {
		genflags = append(genflags, flags[genindex[i]])
	}

	flagstring := "("+ strings.Join(genflags, " ") + ")"
	return flagstring, genflags
}
