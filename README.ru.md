# proxykit

> **Швейцарский нож для прокси-подписок.** Конвертация форматов, проверка живости, детект разблокировки стриминга, HTTP API + веб-UI — один Go-бинарник, без зависимостей в рантайме (кроме `sing-box` при `--via` / `--sub` / `--enable-probes`).

**Языки**: [English](./README.md) · [简体中文](./README.zh-CN.md) · [日本語](./README.ja.md) · [Español](./README.es.md) · [Français](./README.fr.md) · [Deutsch](./README.de.md)

---

## Обзор

- `proxykit convert` — конвертация между Clash / sing-box / v2ray / Surge / QuantumultX / Loon (6 форматов)
- `proxykit test` — TCP + TLS хендшейк-проверка узлов, вывод в table / JSON / CSV
- `proxykit unlock` — статус разблокировки Netflix / Disney+ / YouTube Premium / ChatGPT. Три режима: `--direct` (локально), `--via <uri>` (через один прокси), `--sub <file>` (матрица по всей подписке)
- `proxykit serve` — HTTP-сервер со встроенным SPA

Полная документация и справка API — в [английском README](./README.md).

## Установка

```bash
go install github.com/Au1rxx/proxykit/cmd/proxykit@latest
```

## Примеры

```bash
proxykit convert --in nodes.yaml --to singbox > nodes.json
proxykit test --in nodes.yaml --fast --format table
proxykit unlock --direct
proxykit unlock --sub nodes.yaml --format json
proxykit serve --addr 127.0.0.1:8080
```

## Философия

Честность относительно поддержки. Работают сейчас: VLESS, VMess, Trojan, Shadowsocks, Hysteria2. Пока нет: VLESS Reality, TUIC, AnyTLS. Эвристики разблокировки — снимок текущего поведения; `pkg/unlock` НЕ обещает semver-стабильности.

Сестринский проект: [Au1rxx/free-vpn-subscriptions](https://github.com/Au1rxx/free-vpn-subscriptions) (публичный фид ежечасно верифицируемых узлов). Оба репозитория делят код через публичные `pkg/*`-модули.

## Лицензия

MIT.
