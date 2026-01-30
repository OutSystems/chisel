FROM alpine
RUN apk add --no-cache ca-certificates openssl \
 && apk upgrade --no-cache
WORKDIR /app
COPY chisel /app/
USER 65532:65532
ENTRYPOINT ["/app/chisel"]
ENTRYPOINT ["/app/chisel"]