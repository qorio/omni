omni
====

# Introduction

Collection of servers that make up the Omni platform for

- Universal, cross-platform linking via URL short codes
- Events / stats via Redis/ ElasticSearch/LogStash
- Beacon content management platform
- Ads / api backend


# Development Notes

# Git Subtrees

Git subtrees are used instead of git submodule.  `Embedfs` is included as a subtree under the `third_party` directory.
Refer to this [blog](http://blogs.atlassian.com/2013/05/alternatives-to-git-submodule-git-subtree/) for details.
In particular, use commands below to update the subtree:

```
git fetch embedfs
git subtree pull --prefix third_party/src/github.com/qorio/embedfs embedfs
```
