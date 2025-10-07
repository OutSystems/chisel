FROM alpine
RUN apk update && apk upgrade
COPY chisel /app/
ENTRYPOINT ["/app/chisel"]