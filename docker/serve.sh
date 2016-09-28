#!/usr/bin/env bash
# set -x
#
export GOPATH=/golang/trello
(cd src/github.com/barakb/trello/; git fetch; git rebase)

go build -o bin/trello-burndown github.com/barakb/trello/main

./bin/rello-burndown
