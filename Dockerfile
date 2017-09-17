FROM alpine

RUN apk update
RUN apk add ffmpeg ca-certificates 
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

COPY youtube-playlist-rss /usr/bin/

CMD /usr/bin/youtube-playlist-rss
