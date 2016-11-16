FROM golang:1.7.1
MAINTAINER Barak Bar Orion  <barak.bar@gmail.com>

RUN rm /bin/sh && ln -s /bin/bash /bin/sh


RUN apt-get install -y git

RUN curl https://glide.sh/get | sh

RUN mkdir -p /golang/xap-trello/src/github.com/barakb
WORKDIR /golang/xap-trello/src/github.com/barakb

RUN git clone https://github.com/barakb/xap-trello.git


WORKDIR /golang/xap-trello/src/github.com/barakb/xap-trello

env PATH /golang/xap-trello/src/github.com/barakb/xap-trello:$PATH

RUN glide install

WORKDIR /golang/xap-trello

COPY serve.sh /golang/xap-trello/

EXPOSE 8080
