# ProxyKit

> **Swiss-army knife for proxy users.** Convert subscriptions, test nodes, detect streaming unlocks — one Go binary, zero configuration, ships everywhere.

[![status](https://img.shields.io/badge/status-alpha-orange)](https://github.com/Au1rxx/proxykit)
[![license](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)
[![go.mod](https://img.shields.io/badge/go-1.25-00ADD8)](./go.mod)

---

## What it does (MVP scope)

| Module | Command | Status |
|---|---|---|
| **Subscription converter** | `proxykit convert -i in.yaml --target singbox` | 🚧 W1–W2 |
| **Node speed tester** | `proxykit test https://… --fast` | 🚧 W3 |
| **Streaming unlock checker** | `proxykit test https://… --unlock-only` | 🚧 W5 |
| **HTTP API + Web UI** | `proxykit serve --listen :8080` | 🚧 W6 |

See [`docs/roadmap.md`](./docs/roadmap.md) for the 8-week MVP calendar.

---

## Why this exists

Every proxy user eventually needs to:

1. **Convert subscriptions** between Clash / sing-box / v2ray / Surge / Quantumult X / Loon — because each client wants its own format, and the de-facto tool (`subconverter`) lags several months behind new protocols like Hysteria2, TUIC, and AnyTLS.
2. **Test whether nodes actually work**, not just whether the port is open — including whether they unlock Netflix, Disney+, YouTube Premium, or ChatGPT from your region.
3. **Run both of the above** without installing Docker, writing config files, or copying an URL into a scary third-party website.

ProxyKit does all of that from a single Go binary and a zero-tracking static web UI. It's the sibling project to [`Au1rxx/free-vpn-subscriptions`](https://github.com/Au1rxx/free-vpn-subscriptions) — same author, same Node parsing code (shared via `pkg/node`), same "measure, don't promise" philosophy.

---

## Quickstart (when binaries ship)

```bash
# install
curl -fsSL https://github.com/Au1rxx/proxykit/releases/latest/download/install.sh | sh

# convert a Clash subscription into sing-box format
proxykit convert --source clash --target singbox \
  --input  my-subscription.yaml \
  --output my-subscription.json

# test 150 nodes from a public subscription, fast mode (~5 min)
proxykit test https://au1rxx.github.io/free-vpn-subscriptions/output/v2ray-base64.txt \
  --fast \
  --format table

# check which nodes unlock Netflix + ChatGPT
proxykit test https://... \
  --unlock-only \
  --services netflix,chatgpt \
  --format json > unlock-report.json

# run the web UI locally
proxykit serve --listen :8080
# open http://localhost:8080
```

---

## Supported protocols

VLESS (including Reality), VMess, Trojan, Shadowsocks, Hysteria2, TUIC v5, AnyTLS, WireGuard (passive emit only).

New protocol support lands here first, then flows back to `free-vpn-subscriptions` through the shared `pkg/node` module.

---

## Relationship to free-vpn-subscriptions

[`Au1rxx/free-vpn-subscriptions`](https://github.com/Au1rxx/free-vpn-subscriptions) is where the public subscription feed lives — 150 hourly-verified free VPN nodes, packaged for Clash / sing-box / v2ray. ProxyKit is what you use **on** those subscriptions (or your own private ones):

- need to convert the feed to Surge format? → `proxykit convert`
- need to know which of the 150 nodes actually unlock Netflix from Germany? → `proxykit test --unlock-only`
- need to host a small conversion API on your VPS? → `proxykit serve`

Both projects share the `Node` data model via the public [`github.com/Au1rxx/free-vpn-subscriptions/pkg/node`](https://pkg.go.dev/github.com/Au1rxx/free-vpn-subscriptions/pkg/node) module, so new-protocol support added in one project immediately benefits the other.

---

## Non-goals (what ProxyKit is *not*)

- **Not a GUI client.** Use Clash Verge, Hiddify, or v2rayN for day-to-day browsing.
- **Not a node operator.** ProxyKit doesn't run servers; it analyses subscriptions you already have.
- **Not a paid SaaS.** Free and open source, period. If you need a managed panel, self-host [3x-ui](https://github.com/MHSanaei/3x-ui) or [Marzban](https://github.com/Gozargah/Marzban).
- **Not a political tool.** This is a technical utility. Use it in accordance with your local laws.

---

## License

MIT. See [LICENSE](./LICENSE).
