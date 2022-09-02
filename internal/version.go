package internal

import (
	"strconv"
	"time"
)

var (
	version    = "dev"
	commitHash = "dev"
	buildDate  = "1606748400" // 2020-12-01 00:00:00+09
)

// Version applied app version
func Version() string {
	return version
}

// CommitHash applied git hash
func CommitHash() string {
	return commitHash
}

// BuildDate applied build time
func BuildDate() time.Time {
	ts, _ := strconv.ParseInt(buildDate, 10, 64)
	return time.Unix(ts, 0).UTC()
}
