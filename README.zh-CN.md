# proxykit

> **订阅工具箱瑞士军刀。** 格式互转、节点探活、流媒体解锁检测、HTTP + 浏览器前端——一个 Go 二进制，零运行时依赖（`--via` / `--sub` / `--enable-probes` 时需 `sing-box`）。

[![status](https://img.shields.io/badge/status-alpha-orange)](https://github.com/Au1rxx/proxykit)
[![license](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)
[![go.mod](https://img.shields.io/badge/go-1.25-00ADD8)](./go.mod)

**语言**：[English](./README.md) · [日本語](./README.ja.md) · [Español](./README.es.md) · [Français](./README.fr.md) · [Deutsch](./README.de.md) · [Русский](./README.ru.md)

---

## 能做什么

| 子命令 | 用途 | 是否需 `sing-box` |
|---|---|---|
| `proxykit convert` | Clash / sing-box / v2ray / Surge / QuantumultX / Loon 六格式互转 | 否 |
| `proxykit test`    | 解析订阅、TCP + TLS 握手探活，输出 table/JSON/CSV 报告 | 否 |
| `proxykit unlock`  | 检测 Netflix / Disney+ / YouTube Premium / ChatGPT 解锁。三模式：`--direct`（本机）/ `--via <uri>`（单代理）/ `--sub <file>`（整订阅每节点矩阵）| 仅 `--via` / `--sub` 需要 |
| `proxykit serve`   | HTTP API + 内嵌单页 UI 把前三个子命令挂到 Web 上 | 开 `--enable-probes` 时需要 |

---

## 快速开始

```bash
# 安装（需 Go 1.25+）
go install github.com/Au1rxx/proxykit/cmd/proxykit@latest

# Clash → sing-box 格式
proxykit convert --in nodes.yaml --to singbox > nodes.json

# 订阅 TCP+TLS 探活
proxykit test --in nodes.yaml --fast --format table

# 本机解锁检测
proxykit unlock --direct

# 整订阅每节点 × 每 target 的解锁矩阵
proxykit unlock --sub nodes.yaml --format json > matrix.json

# 启动本地浏览器工具
proxykit serve --addr 127.0.0.1:8080

# 打开 test/unlock 端点 + 鉴权
proxykit serve --addr 0.0.0.0:8080 --enable-probes --auth-token $SECRET
```

---

## 为什么又造一个 subconverter？

[`subconverter`](https://github.com/tindy2013/subconverter) 是事实标准，但久疏维护，很多新协议没跟上。proxykit 的目标是：

1. **对支持协议诚实。** 当下：VLESS / VMess / Trojan / Shadowsocks / Hysteria2。未支持：VLESS Reality、TUIC、AnyTLS。Surge / QuantumultX / Loon 的 emit 层 **partial-coverage**，VLESS + Hysteria2 节点会被静默丢弃，因为这三个客户端没有稳定原生映射。
2. **对"可用"的定义诚实。** TCP+TLS 握手**不能**证明节点真能代理流量——但它便宜，能筛掉 80% 死节点。姊妹仓 [free-vpn-subscriptions](https://github.com/Au1rxx/free-vpn-subscriptions) 在此之上加了 HTTP-over-proxy 完整验证。
3. **对解锁判定的启发式诚实。** 只是 Netflix / Disney+ / YouTube / ChatGPT 目前会泄露地域信息的行为快照。上游改版很频繁，`pkg/unlock` 明确 **不承诺 semver 稳定**。
4. **单二进制，无 Docker、无第三方网页。** HTTP server 是可选子命令，不是默认形态。

---

## 解锁矩阵输出（`--sub --format json`）

```json
[
  { "node": "jp-01", "server": "1.2.3.4:443",
    "results": [
      {"target": "netflix", "status": "partial",  "region": "JP", "detail": "originals only"},
      {"target": "chatgpt", "status": "unlocked", "region": "JP", "detail": "api compliance ok"}
    ]
  }
]
```

每个 target 返回 `unlocked` / `partial` / `blocked` / `unknown` 之一。

---

## HTTP API

| 方法 + 路径 | 鉴权 | 说明 |
|---|---|---|
| `POST /api/convert?from=auto&to=singbox` | 无 | 与 CLI 等价，body = 订阅原文 |
| `POST /api/test?from=auto` | Bearer（若设 `--auth-token`）| 返回 `{total, alive, dropped}` |
| `POST /api/unlock?from=auto&target=netflix,chatgpt` | Bearer | 返回上述矩阵 |
| `GET /health` | 无 | `200 ok` |
| `GET /version` | 无 | `200 proxykit <ver>` |
| `GET /` | 无 | 内嵌单页工具 |

**威胁模型提醒**（绑 `0.0.0.0` 前请读）：

- `test` / `unlock` 是 opt-in（`--enable-probes`），会 spawn `sing-box` 子进程并经用户给的代理发外部请求
- 内置 SSRF 过滤器丢弃 RFC1918 / loopback / link-local / 169.254.169.254 的节点，**但不做 DNS 解析**——攻击者控制 `evil.example.com` → `10.0.0.1` 能绕过。对外服务请在网络层封死私网出站
- `--max-test-nodes 50` / `--max-unlock-nodes 10` / `--parallel 2` 做资源上限
- `--auth-token <str>` 打开 Bearer 鉴权

---

## 与 free-vpn-subscriptions 的关系

[`Au1rxx/free-vpn-subscriptions`](https://github.com/Au1rxx/free-vpn-subscriptions) 维护公开的每小时验证 VPN 节点订阅（约 150 个 live 节点）。proxykit 是一个可以对**任何**订阅（那个仓的、你自己的、别人的）使用的工具箱，两仓通过公开的 `pkg/*` 模块共享 Node 数据模型 + 解析 + emit + probe + unlock 代码。新协议先落节点仓，proxykit `go get -u` 拿到。

---

## 非目标

- **不是 GUI 客户端**。日常浏览请用 Clash Verge、Hiddify、v2rayN
- **不是节点运营商**。proxykit 不跑服务端，只分析订阅
- **不是付费 SaaS**。永远免费开源
- **不是政治工具**。技术实用工具，使用请遵当地法律

---

## License

MIT. 见 [LICENSE](./LICENSE)。
