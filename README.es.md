# proxykit

> **Navaja suiza para suscripciones de proxy.** Conversión de formatos, sondeo de actividad, detección de desbloqueo de streaming, API HTTP + UI web — un binario Go, sin dependencias en tiempo de ejecución (salvo `sing-box` con `--via` / `--sub` / `--enable-probes`).

**Idiomas**: [English](./README.md) · [简体中文](./README.zh-CN.md) · [日本語](./README.ja.md) · [Français](./README.fr.md) · [Deutsch](./README.de.md) · [Русский](./README.ru.md)

---

## Resumen

- `proxykit convert` — conversión entre Clash / sing-box / v2ray / Surge / QuantumultX / Loon (6 formatos)
- `proxykit test` — sondeo TCP + TLS de nodos, salida en table / JSON / CSV
- `proxykit unlock` — estado de desbloqueo Netflix / Disney+ / YouTube Premium / ChatGPT. Tres modos: `--direct` (local), `--via <uri>` (un proxy), `--sub <file>` (matriz de toda la suscripción)
- `proxykit serve` — servidor HTTP con SPA embebida

Documentación completa y referencia de API en el [README en inglés](./README.md).

## Instalación

```bash
go install github.com/Au1rxx/proxykit/cmd/proxykit@latest
```

## Ejemplos

```bash
proxykit convert --in nodes.yaml --to singbox > nodes.json
proxykit test --in nodes.yaml --fast --format table
proxykit unlock --direct
proxykit unlock --sub nodes.yaml --format json
proxykit serve --addr 127.0.0.1:8080
```

## Filosofía

Honestidad sobre lo soportado. Funcionan hoy: VLESS, VMess, Trojan, Shadowsocks, Hysteria2. Todavía no: VLESS Reality, TUIC, AnyTLS. Las heurísticas de desbloqueo son una instantánea del comportamiento actual; `pkg/unlock` NO promete estabilidad semver.

Proyecto hermano: [Au1rxx/free-vpn-subscriptions](https://github.com/Au1rxx/free-vpn-subscriptions) (feed público de nodos verificados cada hora). Los dos repositorios comparten código vía módulos públicos `pkg/*`.

## Licencia

MIT.
