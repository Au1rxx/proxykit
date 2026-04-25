# proxykit

> **Swiss-army knife for proxy subscriptions.** Convert formats, probe liveness, detect streaming unlock, serve an HTTP + browser tool — one Go binary, zero runtime dependencies (except `sing-box` when you use `--via` / `--sub` / `--enable-probes`).

[![status](https://img.shields.io/badge/status-alpha-orange)](https://github.com/Au1rxx/proxykit)
[![license](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)
[![go.mod](https://img.shields.io/badge/go-1.25-00ADD8)](./go.mod)
[![release](https://img.shields.io/github/v/tag/Au1rxx/proxykit?label=release&color=00ADD8)](https://github.com/Au1rxx/proxykit/releases)

**Translations**: [简体中文](./README.zh-CN.md) · [日本語](./README.ja.md) · [Español](./README.es.md) · [Français](./README.fr.md) · [Deutsch](./README.de.md) · [Русский](./README.ru.md)

> 🌐 **Try it in your browser** — no install needed: **<https://au1rxx.github.io/proxykit/>**
> Conversion runs as WebAssembly entirely client-side. Your subscription never leaves the page.

---

## What it does

| Subcommand | Purpose | Needs `sing-box` binary on PATH? |
|---|---|---|
| `proxykit convert`  | Convert subscriptions between Clash / sing-box / v2ray / Surge / QuantumultX / Loon | no |
| `proxykit test`     | Parse a subscription, run TCP + TLS handshake probes, emit a table/JSON/CSV report of which nodes are alive | no |
| `proxykit unlock`   | Probe Netflix / Disney+ / YouTube Premium / ChatGPT unlock status. Three modes: `--direct` (from this host), `--via <uri>` (through a single proxy), or `--sub <file>` (per-node matrix against a whole subscription) | only for `--via` / `--sub` |
| `proxykit serve`    | HTTP API + embedded single-page UI wrapping the above | only when `--enable-probes` |

---

## Install

Three ways to use proxykit, in increasing order of capability:

| Method | Capability | Setup |
|---|---|---|
| **Web (online converter)** | `convert` only — pure WebAssembly, no install | Open <https://au1rxx.github.io/proxykit/> |
| **Pre-built binary** | full CLI: `convert` / `test` / `unlock --direct` / `serve` | Download from [Releases](https://github.com/Au1rxx/proxykit/releases/latest) and unpack |
| **From source** | full CLI + can build wasm yourself | `go install github.com/Au1rxx/proxykit/cmd/proxykit@latest` (requires Go 1.25+) |

```bash
# Linux / macOS — pick the right archive for your arch:
curl -L https://github.com/Au1rxx/proxykit/releases/latest/download/proxykit_$(uname -s | tr A-Z a-z | sed 's/darwin/macos/')_$(uname -m).tar.gz | tar xz
./proxykit --version

# or via Go:
go install github.com/Au1rxx/proxykit/cmd/proxykit@latest
```

> Note: `unlock --via` / `unlock --sub` / `serve --enable-probes` need a `sing-box` binary on PATH. Install it from <https://sing-box.sagernet.org/installation/>.

## Quickstart

```bash

# convert a Clash subscription → sing-box config
proxykit convert --in nodes.yaml --to singbox > nodes.json

# probe a subscription for TCP+TLS liveness
proxykit test --in nodes.yaml --fast --format table

# check which streaming services unlock from your machine
proxykit unlock --direct

# per-node streaming matrix across a whole subscription
proxykit unlock --sub nodes.yaml --format json > matrix.json

# run the browser tool on http://127.0.0.1:8080
proxykit serve --addr 127.0.0.1:8080

# same, with test/unlock endpoints enabled and a bearer token
proxykit serve --addr 0.0.0.0:8080 --enable-probes --auth-token $SECRET
```

---

## Why another subconverter?

[`subconverter`](https://github.com/tindy2013/subconverter) is the de facto tool but it's unmaintained enough to have missed several protocols. proxykit aims to be:

1. **Accurate about what it supports.** Today: VLESS, VMess, Trojan, Shadowsocks, Hysteria2. Not yet: VLESS Reality, TUIC, AnyTLS. The `emit` layer on Surge/QuantumultX/Loon is honest-partial: VLESS + Hysteria2 nodes are dropped silently because those clients have no stable native mapping.
2. **Honest about what "alive" means.** TCP+TLS handshake is _not_ proof a node proxies traffic — but it's cheap and filters out 80% of dead feeds. The sibling [free-vpn-subscriptions](https://github.com/Au1rxx/free-vpn-subscriptions) repo adds a full HTTP-over-proxy stage on top for its public feed.
3. **Honest about streaming-unlock heuristics.** They are best-effort snapshots of how Netflix / Disney+ / YouTube / ChatGPT are known to leak region information _right now_. Upstream services change these regularly; the `pkg/unlock` package is explicitly NOT semver-stable.
4. **A single binary, no Docker, no web form.** The HTTP server is an optional subcommand, not the default mode.

---

## Streaming-unlock mode

```bash
# from this host
proxykit unlock --direct

# through a single proxy URI (spawns sing-box on demand)
proxykit unlock --via 'trojan://pw@host:443?sni=host#my-node'

# matrix across a whole subscription file
proxykit unlock --sub nodes.yaml --target netflix,chatgpt --format json
```

**Output shape (matrix, `--sub --format json`):**

```json
[
  { "node": "jp-01", "server": "1.2.3.4:443",
    "results": [
      {"target": "netflix", "status": "partial",  "region": "JP", "detail": "originals only"},
      {"target": "chatgpt", "status": "unlocked", "region": "JP", "detail": "api compliance ok"}
    ]
  },
  ...
]
```

Each target returns one of `unlocked` / `partial` / `blocked` / `unknown`.

---

## HTTP API (`proxykit serve`)

| Method + path | Auth | Purpose |
|---|---|---|
| `POST /api/convert?from=auto&to=singbox` | none | Same semantics as the CLI; body = raw subscription |
| `POST /api/test?from=auto` | Bearer (if `--auth-token` set) | Returns `{total, alive, dropped}` |
| `POST /api/unlock?from=auto&target=netflix,chatgpt` | Bearer | Returns the matrix shape above |
| `GET /health` | none | `200 ok` |
| `GET /version` | none | `200 proxykit <ver>` |
| `GET /` | none | Embedded browser tool |

**Threat-model notes** (read before binding to `0.0.0.0`):

- `test` and `unlock` are opt-in (`--enable-probes`). They spawn `sing-box` subprocesses and make outbound requests through user-supplied proxy nodes.
- A built-in SSRF filter drops nodes whose `server` field is RFC1918 / loopback / link-local / 169.254.169.254. It does NOT resolve hostnames — an attacker who controls DNS for `evil.example.com` pointing at `10.0.0.1` will currently bypass it. If you expose this endpoint to untrusted users, front it with a network policy that blocks outbound private ranges.
- Per-request node caps (`--max-test-nodes 50`, `--max-unlock-nodes 10`) + global concurrency (`--parallel 2`) bound resource usage.
- `--auth-token <string>` adds `Authorization: Bearer <token>` enforcement on the guarded endpoints.

---

## Relationship to free-vpn-subscriptions

[`Au1rxx/free-vpn-subscriptions`](https://github.com/Au1rxx/free-vpn-subscriptions) hosts the public hourly-verified VPN node feed (~150 alive nodes). proxykit is a toolbox you run **on** any subscription — theirs, yours, or someone else's — and shares the `Node` data model, parsing, emit, probe, and unlock code via public `pkg/*` modules:

| proxykit uses | from |
|---|---|
| `pkg/node` | URI parsing + normalised Node type |
| `pkg/parse` | Clash YAML / base64 blob / URI-list → `[]*Node` |
| `pkg/emit` | `[]*Node` → Clash / sing-box / v2ray / Surge / QuantumultX / Loon |
| `pkg/probe` | TCP + TLS handshake liveness checks |
| `pkg/unlock` | Netflix / Disney+ / YouTube / ChatGPT heuristics |

New protocol support lands in the node repo; proxykit picks it up with a `go get -u`.

---

## Non-goals

- **Not a GUI client.** Use Clash Verge, Hiddify, v2rayN for day-to-day browsing.
- **Not a node operator.** proxykit doesn't run servers; it analyses subscriptions you already have.
- **Not a paid SaaS.** Free and open source.
- **Not a political tool.** Technical utility. Use in accordance with local law.

---

## Roadmap

See the public-facing commit log. Internally tracked milestones:

- **W1–W5** ✅ Shipped: convert (6 formats), test --fast, unlock (3 modes + sing-box launcher)
- **W6** ✅ Shipped: `serve` subcommand with convert/test/unlock endpoints + embedded SPA
- **W7** ✅ Shipped (alt path): browser-side WASM converter at <https://au1rxx.github.io/proxykit/> using the standard Go wasm target (no Cloudflare Worker dependency)
- **W8** ✅ Shipped: v0.1.0 release tag + GoReleaser CI for multi-arch binaries + two-repo cross-promotion

---

## License

MIT. See [LICENSE](./LICENSE).
