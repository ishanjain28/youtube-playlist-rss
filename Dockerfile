FROM ubuntu

RUN apt-get update
RUN apt-get install -y ffmpeg 

COPY youtube-playlist-rss /usr/bin

CMD /usr/bin/youtube-playlist-rss
