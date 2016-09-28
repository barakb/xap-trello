#!/usr/bin/env bash
set -e
export config_dir=/home/barakbo/golang/barak/trello/src/github.com/barakb/trello/
docker run  -v ${config_dir}:/trello-conf -p 127.0.0.1:8080:8080  -it barakb/trello:0.1
