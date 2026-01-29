FROM gcr.io/distroless/static-debian12:nonroot
COPY chisel /app/
ENTRYPOINT ["/app/chisel"]