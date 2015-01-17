package version

import (
	"fmt"
	"os"
)

var (
	// These are bound by the -X key value ldflags option of the go compiler
	gitRepo          string
	gitBranch        string
	gitTag           string
	gitCommitHash    string
	gitCommitMessage string
	buildTimestamp   string
	buildNumber      string
)

var (
	build_info = &Build{
		Timestamp:  buildTimestamp,
		BuildLabel: buildNumber,
		RepoUrl:    gitRepo,
		Tag:        gitTag,
		Branch:     gitBranch,
		Commit:     gitCommitHash,
		Message:    gitCommitMessage,
	}
)

type Build struct {
	RepoUrl    string
	Branch     string
	Tag        string
	Commit     string
	Message    string
	BuildLabel string
	Timestamp  string
	Number     string
}

func BuildInfo() *Build {
	return build_info
}

func SetBuildInfo(b *Build) {
	build_info = b
}

func (buildInfo *Build) Notice() string {
	line1 := fmt.Sprintf("%s: Version %s, Build %s, Built on %s. ", os.Args[0], buildInfo.Tag, buildInfo.BuildLabel, buildInfo.Timestamp)
	line2 := fmt.Sprintf("Git repo=%s branch=%s commit=%s message=%s", buildInfo.RepoUrl, buildInfo.Branch, buildInfo.Commit, buildInfo.Message)
	return line1 + line2
}

func (buildInfo *Build) GetRepoUrl() string {
	return buildInfo.RepoUrl
}

func (buildInfo *Build) GetBranch() string {
	return buildInfo.Branch
}

func (buildInfo *Build) GetTag() string {
	return buildInfo.Tag
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
