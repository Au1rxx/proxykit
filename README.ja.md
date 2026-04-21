# proxykit

> **プロキシ購読ツール箱。** フォーマット変換、生存確認、ストリーミング解除チェック、HTTP + ブラウザ UI — Go バイナリ 1 本、ランタイム依存なし（`--via` / `--sub` / `--enable-probes` 時のみ `sing-box` が必要）。

**言語**: [English](./README.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Français](./README.fr.md) · [Deutsch](./README.de.md) · [Русский](./README.ru.md)

---

## 概要

- `proxykit convert` — Clash / sing-box / v2ray / Surge / QuantumultX / Loon の 6 フォーマット相互変換
- `proxykit test` — TCP + TLS ハンドシェイクで生存確認、table / JSON / CSV 出力
- `proxykit unlock` — Netflix / Disney+ / YouTube Premium / ChatGPT の解除判定。3 モード: `--direct`（ローカル）、`--via <uri>`（単一プロキシ経由）、`--sub <file>`（購読全体のマトリクス）
- `proxykit serve` — HTTP API + 埋め込み SPA

完全なドキュメントと API リファレンスは [English README](./README.md) を参照。

## インストール

```bash
go install github.com/Au1rxx/proxykit/cmd/proxykit@latest
```

## 例

```bash
proxykit convert --in nodes.yaml --to singbox > nodes.json
proxykit test --in nodes.yaml --fast --format table
proxykit unlock --direct
proxykit unlock --sub nodes.yaml --format json
proxykit serve --addr 127.0.0.1:8080
```

## 設計方針

サポート対象について正直であること。今日動くもの: VLESS、VMess、Trojan、Shadowsocks、Hysteria2。未対応: VLESS Reality、TUIC、AnyTLS。ストリーミング解除判定は "現在既知の挙動のスナップショット" であり、`pkg/unlock` は semver 安定を約束しない。

姉妹プロジェクト: [Au1rxx/free-vpn-subscriptions](https://github.com/Au1rxx/free-vpn-subscriptions) (毎時検証済み公開購読フィード)。両リポジトリは公開 `pkg/*` モジュール経由でコードを共有する。

## License

MIT.
