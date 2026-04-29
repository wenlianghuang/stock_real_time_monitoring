# wsprovider（供應商適配層）

這個資料夾預留給「真實行情供應商」的 WebSocket 介接。

## 設計原則

- **介面固定**：對外只實作 `provider.MarketDataProvider` 的 `Run(ctx, store)`。
- **內部做 mapping**：把供應商的訊息（逐筆或 quote/book 更新）轉成 `domain.QuoteState` 並寫入 `state.Store`。
- **重連策略**：\n+  - 斷線自動重連（指數退避：0.5s → 1s → 2s → … → 10s 上限）\n+  - 重連後重新訂閱 symbols\n+  - 讀寫 deadline / ping-pong keepalive\n+
## 供應商差異通常出現在哪\n+
- **授權與 token**：API key / access token / refresh token\n+- **訂閱模型**：一次訂閱一檔 vs 一次訂閱多檔\n+- **欄位**：是否提供五檔、成交量口徑、時間戳\n+- **編碼/壓縮**：JSON / protobuf / gzip\n+
## 建議落地方式\n+
每個供應商一個資料夾，例如：\n+\n+- `backend/internal/provider/wsprovider/vendorA/`\n+- `backend/internal/provider/wsprovider/vendorB/`\n+\n+並在 `cmd/server/main.go` 以 env 決定要用哪個 provider（mock 或 vendor）。\n+
