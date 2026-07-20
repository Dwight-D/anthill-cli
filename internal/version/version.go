// Package version exposes the Anthill CLI build version.
package version

// Version is the semantic version of the anthill CLI.
const Version = "0.0.1"

// String returns a human-readable version string for the anthill CLI.
func String() string {
	return "anthill " + Version
}
