# SEC-20 — Demo droplet: SSH open to the world + Cloudflare origin bypass

- **Severity:** Low
- **Status:** Confirmed present
- **Area:** Infra
- **File:** [iac/demo/main.tf:40-90](../../iac/demo/main.tf)

## Problem

The demo droplet firewall allows inbound from `0.0.0.0/0` on three ports:

```hcl
inbound_rule { port_range = "22";  source_addresses = ["0.0.0.0/0", "::/0"] }  # SSH
inbound_rule { port_range = "80";  source_addresses = ["0.0.0.0/0", "::/0"] }
inbound_rule { port_range = "443"; source_addresses = ["0.0.0.0/0", "::/0"] }  # comment says "Cloudflare Full"
```

Two issues:

1. **SSH (22) open to the entire internet** — a constant brute-force/scan surface.
2. **443 open to the world while the origin sits behind Cloudflare** — the comment states Cloudflare connects on HTTPS, but allowing `0.0.0.0/0` means an attacker can hit the origin IP directly and **bypass Cloudflare's WAF, rate limiting, and DDoS protection** entirely, removing the only edge security layer.

## Risk

This is a demo droplet, so blast radius is limited, but: SSH exposed to the world is a noisy attack surface, and the Cloudflare bypass nullifies the edge protections the architecture assumes. If the demo ever holds real data, both become real exposure.

## Fix

1. Restrict SSH `source_addresses` to a known egress (VPN/home IP, or a Tailscale-only address). Enable `fail2ban` and key-only auth via cloud-init.
2. Restrict ports 80/443 `source_addresses` to [Cloudflare's published IP ranges](https://www.cloudflare.com/ips/) so the origin is only reachable through the edge. Combine with Cloudflare Authenticated Origin Pulls (mTLS) for defense in depth.

## Verification

- `nmap` of the droplet from an unlisted IP shows 22 filtered.
- A direct `curl https://<origin-ip>` from a non-Cloudflare IP is refused; requests via the Cloudflare hostname succeed.
