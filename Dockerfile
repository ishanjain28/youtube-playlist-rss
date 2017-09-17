FROM ubuntu

RUN  apt-get update
RUN  apt-get install -y ffmpeg

COPY justforfuncrss /usr/bin

RUN /usr/bin/justforfuncrss