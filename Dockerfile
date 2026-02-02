FROM alpine:latest
LABEL maintainer="MagnaX Software <opensource@magnax.ca>"
LABEL org.opencontainers.image.authors="MagnaX Software" \
      org.opencontainers.image.vendor="MagnaX Software" \
      org.opencontainers.image.title="docker-retag" \
      org.opencontainers.image.description="Re-tag docker/OCI images without downloading the whole image" \
      org.opencontainers.image.source="https://github.com/MagnaXSoftware/docker-retag" \
      org.opencontainers.image.url="https://github.com/MagnaXSoftware/docker-retag" \
      org.opencontainers.image.documentation="https://github.com/MagnaXSoftware/docker-retag" \
      org.opencontainers.image.licenses="MIT"

ARG TARGETPLATFORM

ENTRYPOINT ["/usr/bin/docker-retag"]

COPY $TARGETPLATFORM/docker-retag /usr/bin/