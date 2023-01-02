FROM alpine:3.15

COPY bin/app /

ENTRYPOINT ["/app"]
