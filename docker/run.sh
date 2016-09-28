#!/usr/bin/env bash
set -e
#docker run -v ${config_dir}:/trello-conf -p 127.0.0.1:8080:8080  -it barakb/trello:0.1
docker run  -p 127.0.0.1:8080:8080  -it barakb/trello:0.1
