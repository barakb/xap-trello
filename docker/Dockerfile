FROM golang:1.7.1
MAINTAINER Barak Bar Orion  <barak.bar@gmail.com>

RUN rm /bin/sh && ln -s /bin/bash /bin/sh

#RUN apt-get update
#RUN apt-get install -y software-properties-common

RUN apt-get install -y git

#RUN add-apt-repository ppa:masterminds/glide &&  apt-get update
#RUN apt-get install -y glide

RUN curl https://glide.sh/get | sh

RUN mkdir -p /golang/trello/src/github.com/barakb
WORKDIR /golang/trello/src/github.com/barakb

RUN git clone https://github.com/barakb/trello.git


WORKDIR /golang/trello/src/github.com/barakb/trello

RUN pwd
RUN ls -lai

env PATH /golang/trello/src/github.com/barakb/trello:$PATH

RUN glide install



EXPOSE 8080
