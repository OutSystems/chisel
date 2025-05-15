# FROM alpine

# # Install tini for proper signal handling
# RUN apk add --no-cache tini

# # Copy your binary
# COPY chisel /app/chisel

# # Use tini as the entrypoint to forward signals to your binary
# ENTRYPOINT ["/sbin/tini", "--", "/app/chisel"]


FROM alpine
COPY chisel /app/
ENTRYPOINT ["/app/chisel"]