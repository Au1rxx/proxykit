// Package server exposes proxykit's subscription-conversion pipeline
// over HTTP. It reuses internal/convert so the API surface stays in
// lock-step with the CLI's `proxykit convert` command.
//
// Scope (MVP)
//
//   - POST /api/convert?from=auto|clash|v2ray|uri-list|base64
//     &to=clash|singbox|v2ray|surge|quanx|loon
//     Body = raw subscription. Response = converted subscription.
//   - GET  /health      → 200 "ok"
//   - GET  /version     → 200 "proxykit <ver>"
//   - GET  /            → embedded single-page HTML tool
//
// test/unlock endpoints are intentionally deferred: they need sing-box
// on PATH and a per-request process budget, which changes the server
// threat model. Design those as a second slice.
package server
