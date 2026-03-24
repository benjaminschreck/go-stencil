package version

import "fmt"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func Details() string {
	if Commit == "" || Commit == "unknown" {
		return Version
	}
	if Date == "" || Date == "unknown" {
		return fmt.Sprintf("%s (%s)", Version, shortCommit(Commit))
	}
	return fmt.Sprintf("%s (%s, %s)", Version, shortCommit(Commit), Date)
}

func shortCommit(commit string) string {
	if len(commit) <= 7 {
		return commit
	}
	return commit[:7]
}
