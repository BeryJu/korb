FROM alpine:3

RUN apk add --no-cache rsync bash tar && rm -rf /var/cache/apk/*

VOLUME [ "/source", "/dest" ]

COPY ./entrypoint.sh /bin/entrypoint

ENTRYPOINT [ "/bin/entrypoint" ]
