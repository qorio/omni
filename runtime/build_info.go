package runtime

const (
	gitCommitHash    = "@@GIT_COMMIT_HASH@@"
	gitCommitMessage = "@@GIT_COMMIT_MESSAGE@@"
	buildTimestamp   = "@@BUILD_TIMESTAMP@@"
	buildNumber      = "@@BUILD_NUMBER@@"
)

var (
	build_info = &Build{
		Commit:    gitCommitHash,
		Timestamp: buildTimestamp,
		Number:    buildNumber,
	}
)

type Build struct {
	Commit    string
	Timestamp string
	Number    string
}

func BuildInfo() *Build {
	return build_info
}

func SetBuildInfo(b *Build) {
	build_info = b
}

func (buildInfo *Build) GetCommitHash() string {
	return buildInfo.Commit
}

func (buildInfo *Build) GetBuildTimestamp() string {
	return buildInfo.Timestamp
}

func (buildInfo *Build) GetBuildNumber() string {
	return buildInfo.Number
}
