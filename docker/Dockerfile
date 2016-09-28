FROM golang:1.6
MAINTAINER Barak Bar Orion  <barak.bar@gmail.com>

RUN rm /bin/sh && ln -s /bin/bash /bin/sh

RUN apt-get install -y git

RUN add-apt-repository ppa:masterminds/glide &&  apt-get update
RUN apt-get install glide

RUN mkdir -p /golang/trello/src/github.com/trello
WORKDIR /golang/trello/src/github.com/trello

RUN git clone git@github.com:barakb/trello.git


WORKDIR /golang/trello/src/github.com/trello

env PATH /golang/trello/src/github.com/trello:$PATH



EXPOSE 8080
