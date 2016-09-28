FROM golang:1.7.1
MAINTAINER Barak Bar Orion  <barak.bar@gmail.com>

RUN rm /bin/sh && ln -s /bin/bash /bin/sh


RUN apt-get install -y git

RUN curl https://glide.sh/get | sh

RUN mkdir -p /golang/trello/src/github.com/barakb
WORKDIR /golang/trello/src/github.com/barakb

RUN git clone https://github.com/barakb/trello.git


WORKDIR /golang/trello/src/github.com/barakb/trello

env PATH /golang/trello/src/github.com/barakb/trello:$PATH

RUN glide install

WORKDIR /golang/trello

COPY serve.sh /golang/trello/

EXPOSE 8080
