# 台股看盤（L1）Go + React

本專案是本機可跑的台股（L1）看盤系統骨架：

- Go 後端：mock 行情來源（可替換成真實 WebSocket 行情供應商）、維護最新 state、每 1 秒推送快照到前端
- React 前端：自選股表格、最佳五檔、sparkline、排序/篩選、自選股與告警持久化（localStorage）

## 目錄

- `backend/`：Go server（WebSocket `/ws`、健康檢查 `/healthz`）
- `frontend/`：React + Vite UI

## 開發啟動（預計）

後端：

```bash
cd backend
go run ./cmd/server
```

前端：

```bash
cd frontend
npm install
npm run dev
```

## 使用 Fugle 真實行情

1. 申請 Fugle MarketData API key
2. 於 `backend/` 建立 `.env`（可先複製 `backend/.env.example`）並填入你的 key
3. 啟動後端

```bash
cd backend
go run ./cmd/server
```

