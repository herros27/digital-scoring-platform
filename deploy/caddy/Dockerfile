# Caddy + modul DNS Cloudflare: sertifikat Let's Encrypt lewat tantangan DNS, sehingga server di jaringan lokal
# (tidak terjangkau dari internet) tetap mendapat sertifikat yang langsung dipercaya tablet. Lihat docs/PWA.md.
FROM caddy:2-builder AS builder
RUN xcaddy build --with github.com/caddy-dns/cloudflare

FROM caddy:2-alpine
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
