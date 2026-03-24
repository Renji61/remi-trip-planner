# Remote Access Options

## Option A: Domain + HTTPS (Caddy)

1. Deploy this project on a small VPS or always-on home machine.
2. Point DNS `A` record to server IP.
3. Replace `trip.example.com` in `deploy/Caddyfile`.
4. Start `docker compose up -d --build`.

Caddy handles automatic TLS certificates.

## Option B: Private Access (Tailscale)

For personal use without public exposure:
- install Tailscale on host and phone/laptop
- access app over tailnet IP and keep service private

## Security Baseline

- use strong app password once auth module is enabled
- keep host firewall enabled (`80/443` only if public)
- enable automatic container restarts and OS updates
