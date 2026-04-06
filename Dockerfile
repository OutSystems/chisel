FROM alpine:3.21
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates
COPY chisel /app/
ENTRYPOINT ["/app/chisel"]