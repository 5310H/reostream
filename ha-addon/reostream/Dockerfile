# Supervisor passes BUILD_ARCH; buildx also passes TARGETARCH.
FROM golang:1.26.0-alpine3.23 AS build
RUN apk add --no-cache git
ARG BUILD_ARCH
ARG TARGETARCH
# Deliberate immutable source pin, not an upstream image or a moving branch.
ENV REOSTREAM_SOURCE=ab698dba24f218be26192d25ff1cc1fd38f59778
WORKDIR /src
RUN git init . \
    && git remote add origin https://github.com/5310h/reostream.git \
    && git fetch --depth=1 origin "$REOSTREAM_SOURCE" \
    && git checkout --detach FETCH_HEAD \
    && test "$(git rev-parse HEAD)" = "$REOSTREAM_SOURCE"
RUN go mod download
RUN arch="${TARGETARCH:-$BUILD_ARCH}"; \
    case "$arch" in aarch64|arm64) arch=arm64 ;; amd64) ;; *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; esac; \
    CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags="-s -w -X github.com/VoltTech21/reostream/internal/control.version=ha-0.1.0-${REOSTREAM_SOURCE}" \
    -o /out/reostream ./cmd/reostream

FROM alpine:3.22.3
ARG BUILD_ARCH
RUN apk add --no-cache ca-certificates su-exec \
    && addgroup -g 65532 reostream \
    && adduser -D -H -u 65532 -G reostream reostream
COPY --from=build /out/reostream /usr/local/bin/reostream
COPY --from=build /src/LICENSE /src/COPYRIGHT /usr/share/doc/reostream/
COPY run.sh /run.sh
RUN chmod 0755 /run.sh
LABEL io.hass.name="reostream" \
      io.hass.description="Reolink Baichuan streaming and camera management" \
      io.hass.type="addon" \
      io.hass.arch="${BUILD_ARCH}" \
      io.hass.version="0.1.0" \
      org.opencontainers.image.source="https://github.com/5310h/reostream" \
      org.opencontainers.image.revision="ab698dba24f218be26192d25ff1cc1fd38f59778"
EXPOSE 8560/tcp 8561/tcp 8562/tcp 8000/udp 8001/udp 2121-2131/tcp
WORKDIR /data/reostream
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8560/api/status || exit 1
ENTRYPOINT ["/run.sh"]
