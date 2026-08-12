FROM --platform=$BUILDPLATFORM caddy:2.11.4-builder AS builder

ARG TARGETOS
ARG TARGETARCH

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOTOOLCHAIN=auto xcaddy build \
    --with github.com/greenpau/caddy-security@v1.1.64 \
    --with github.com/caddy-dns/cloudflare@v0.2.4 \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2@v2.13.1

FROM caddy:2.11.4

COPY --from=builder /usr/bin/caddy /usr/bin/caddy

ENTRYPOINT ["caddy"]
CMD ["docker-proxy", "--caddyfile-path", "/etc/caddy/Caddyfile"]
