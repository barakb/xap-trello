#!/usr/bin/env bash
set -e
export config_dir=/home/barakbo/golang/barak/xap-trello/src/github.com/barakb/xap-trello/
docker run  -v ${config_dir}:/trello-conf -p 8080:8080  -it barakb/xap-trello:0.1
