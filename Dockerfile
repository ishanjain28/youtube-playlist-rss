FROM ubuntu

RUN apt-get update
RUN apt-get install -y ffmpeg wget git
RUN wget https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.9.linux-amd64.tar.gz
RUN mkdir /gowork


ENV PATH="$PATH:/usr/local/go/bin:/gowork/bin"
ENV GOPATH="/gowork"

RUN rm go1.9.linux-amd64.tar.gz
RUN go get github.com/ishanjain28/youtube-playlist-rss

RUN youtube-playlist-rss