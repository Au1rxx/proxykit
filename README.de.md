# proxykit

> **Schweizer Taschenmesser für Proxy-Abonnements.** Format-Konvertierung, Liveness-Sondierung, Streaming-Unlock-Erkennung, HTTP-API + Web-UI — eine Go-Binary, keine Laufzeitabhängigkeiten (außer `sing-box` bei `--via` / `--sub` / `--enable-probes`).

**Sprachen**: [English](./README.md) · [简体中文](./README.zh-CN.md) · [日本語](./README.ja.md) · [Español](./README.es.md) · [Français](./README.fr.md) · [Русский](./README.ru.md)

---

## Übersicht

- `proxykit convert` — Konvertierung zwischen Clash / sing-box / v2ray / Surge / QuantumultX / Loon (6 Formate)
- `proxykit test` — TCP + TLS Handshake-Sondierung, Ausgabe als Tabelle / JSON / CSV
- `proxykit unlock` — Unlock-Status für Netflix / Disney+ / YouTube Premium / ChatGPT. Drei Modi: `--direct` (lokal), `--via <uri>` (einzelner Proxy), `--sub <file>` (Matrix über gesamtes Abonnement)
- `proxykit serve` — HTTP-Server mit eingebetteter SPA

Vollständige Dokumentation und API-Referenz im [englischen README](./README.md).

## Installation

```bash
go install github.com/Au1rxx/proxykit/cmd/proxykit@latest
```

## Beispiele

```bash
proxykit convert --in nodes.yaml --to singbox > nodes.json
proxykit test --in nodes.yaml --fast --format table
proxykit unlock --direct
proxykit unlock --sub nodes.yaml --format json
proxykit serve --addr 127.0.0.1:8080
```

## Designphilosophie

Ehrlichkeit bezüglich unterstützter Protokolle. Heute funktionieren: VLESS, VMess, Trojan, Shadowsocks, Hysteria2. Noch nicht: VLESS Reality, TUIC, AnyTLS. Die Unlock-Heuristiken sind eine Momentaufnahme des aktuellen Verhaltens; `pkg/unlock` verspricht KEINE semver-Stabilität.

Schwesterprojekt: [Au1rxx/free-vpn-subscriptions](https://github.com/Au1rxx/free-vpn-subscriptions) (öffentlicher Feed stündlich verifizierter Knoten). Beide Repositories teilen Code über die öffentlichen `pkg/*`-Module.

## Lizenz

MIT.
