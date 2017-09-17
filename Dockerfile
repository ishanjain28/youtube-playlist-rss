FROM ubuntu

RUN apt-get update
RUN apt-get install -y ffmpeg wget
RUN wget https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz
RUN tar -C /usr/local -xzvf go1.9.linux-amd64.tar.gz
RUN mkdir gowork


ENV PATH="$PATH:/usr/local/go/bin"
ENV PATH="$PATH:gowork/bin"
ENV GOPATH="gowork"

RUN go get github.com/ishanjain28/youtube-playlist-rss

RUN youtube-playlist-rss