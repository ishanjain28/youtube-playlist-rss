FROM alpine

RUN apk update
RUN apk add ffmpeg ca-certificates wget  git
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

RUN wget https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz

RUN tar -C /usr/local -xzf go1.9.linux-amd64.tar.gz

ENV PATH="$PATH:/usr/local/go/bin:/gowork/bin"
ENV GOPATH="/gowork"

RUN go get github.com/ishanjain28/youtube-playlist-rss

CMD youtube-playlist-rss
