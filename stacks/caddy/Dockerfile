FROM --platform=$BUILDPLATFORM caddy:2.10.2-builder AS builder

ARG TARGETOS
ARG TARGETARCH

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOTOOLCHAIN=auto xcaddy build \
    --with github.com/greenpau/caddy-security@v1.1.31 \
    --with github.com/caddy-dns/cloudflare@v0.2.2 \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2@v2.10.0

FROM caddy:2.10.2

COPY --from=builder /usr/bin/caddy /usr/bin/caddy

ENTRYPOINT ["caddy"]
CMD ["docker-proxy", "--caddyfile-path", "/etc/caddy/Caddyfile"]
