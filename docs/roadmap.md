# ProxyKit MVP Roadmap

8-week calendar from empty repo to v0.1.0. Tracks the plan pinned in the parent project at `vpn-lab/docs/plans/2026-04-20-proxykit-mvp.md`.

| Week | Deliverable | Done when |
|---|---|---|
| **W1** | Repo skeleton; Go module pointing at `pkg/node` from the sibling repo; `proxykit --version` runs | `go build ./...` green; `cobra` root + help text |
| **W2** | `proxykit convert` MVP: clash ↔ singbox ↔ v2ray | 3 input × 5 output = 15 test-case matrix green |
| **W3** | `proxykit test --fast` wired to reused HTTP-over-proxy verify | Produces json + table output on a 150-node subscription |
| **W4** | Surge / Quantumult X / Loon emitters | Public fixture subscriptions import cleanly into each client |
| **W5** | Streaming unlock detector (Netflix / Disney+ / YouTube Premium / ChatGPT) | `proxykit test --unlock-only` works; detection logic committed + unit-tested |
| **W6** | HTTP server + static Web UI (convert only) | Deployed to `au1rxx.github.io/proxykit`; `/api/v1/convert` subconverter-compatible |
| **W7** | Cloudflare Worker build (tinygo subset of converter) | Cold start <50ms; `clash → singbox` works on Worker runtime |
| **W8** | v0.1.0 release: binaries, Docker image, site live, cross-linking with free-vpn-subscriptions done | `gh release` published; two sibling-repo READMEs mention each other; first Reddit/V2EX post scheduled |

## Explicitly deferred past v0.1.0

- Visual rule editor (module C in the original plan)
- IP/DNS/WebRTC leak checker (module D)
- GUI / mobile app
- Paid tier — ProxyKit stays free; monetisation lives in the future NexusProxy management panel project, not here

## Success metrics (by week 16)

- 100 ★ on GitHub
- 1k UV/day on the tool site
- 100 API registrations
- 3 organic community posts (not by us) mentioning ProxyKit

If not hit, iterate on module A + B quality (more protocols, more unlock targets, better docs) — do **not** pull in modules C/D.
