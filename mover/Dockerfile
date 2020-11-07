FROM alpine:3.12

RUN apk add --no-cache rsync && rm -rf /var/cache/apk/*

VOLUME [ "/source", "/dest" ]

CMD [ "rsync", "-aHA", "--progress", "/source/", "/dest" ]
