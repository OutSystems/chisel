FROM gcr.io/distroless/static:nonroot
COPY chisel /app/
ENTRYPOINT ["/app/chisel"]