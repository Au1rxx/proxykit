# proxykit

> **Couteau suisse pour abonnements proxy.** Conversion de formats, sondage de vivacité, détection de déblocage de streaming, API HTTP + UI web — un binaire Go, zéro dépendance d'exécution (sauf `sing-box` avec `--via` / `--sub` / `--enable-probes`).

**Langues** : [English](./README.md) · [简体中文](./README.zh-CN.md) · [日本語](./README.ja.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [Русский](./README.ru.md)

---

## Vue d'ensemble

- `proxykit convert` — conversion entre Clash / sing-box / v2ray / Surge / QuantumultX / Loon (6 formats)
- `proxykit test` — sondage TCP + TLS des nœuds, sortie table / JSON / CSV
- `proxykit unlock` — état de déblocage Netflix / Disney+ / YouTube Premium / ChatGPT. Trois modes : `--direct` (local), `--via <uri>` (proxy unique), `--sub <file>` (matrice par nœud sur tout l'abonnement)
- `proxykit serve` — serveur HTTP avec SPA intégrée

Documentation complète et référence API dans le [README anglais](./README.md).

## Installation

```bash
go install github.com/Au1rxx/proxykit/cmd/proxykit@latest
```

## Exemples

```bash
proxykit convert --in nodes.yaml --to singbox > nodes.json
proxykit test --in nodes.yaml --fast --format table
proxykit unlock --direct
proxykit unlock --sub nodes.yaml --format json
proxykit serve --addr 127.0.0.1:8080
```

## Philosophie

Honnêteté sur ce qui est supporté. Fonctionnel aujourd'hui : VLESS, VMess, Trojan, Shadowsocks, Hysteria2. Pas encore : VLESS Reality, TUIC, AnyTLS. Les heuristiques de déblocage sont un instantané du comportement actuel ; `pkg/unlock` NE promet PAS la stabilité semver.

Projet frère : [Au1rxx/free-vpn-subscriptions](https://github.com/Au1rxx/free-vpn-subscriptions) (flux public de nœuds vérifiés chaque heure). Les deux dépôts partagent le code via les modules publics `pkg/*`.

## Licence

MIT.
