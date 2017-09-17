FROM alpine

RUN apk update
RUN apk add ffmpeg ca-certificates

COPY youtube-playlist-rss /usr/bin

CMD /usr/bin/youtube-playlist-rss
