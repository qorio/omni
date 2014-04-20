package runtime

const (
	gitCommitHash    = "@@GIT_COMMIT_HASH@@"
	gitCommitMessage = "@@GIT_COMMIT_MESSAGE@@"
	buildTimestamp   = "@@BUILD_TIMESTAMP@@"
	buildNumber      = "@@BUILD_NUMBER@@"
)

type Build struct {
	Commit    string
	Timestamp string
	Number    string
}

func BuildInfo() Build {
	return Build{
		Commit:    gitCommitHash,
		Timestamp: buildTimestamp,
		Number:    buildNumber,
	}
}
