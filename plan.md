Go + React 台股看盤（L1）完整規劃

目標與範圍





核心目標: 本機跑一套「即時看盤」系統，後端接行情推播並聚合，前端每秒刷新顯示。



資料層級: L1（最新成交、成交量、漲跌、開高低、最佳五檔）。



第一版功能





自選股表格: 代號/名稱/成交價/漲跌/漲跌幅/總量/更新時間



最佳五檔: bid/ask 1~5 價量



UI 節流: 後端或前端固定 1 秒推送/渲染一次（資料可更高頻進來）



排序/篩選: 漲跌幅、成交量、代號、名稱搜尋



自選股持久化: 先用 localStorage（跨平台瀏覽器可用、無後端帳號系統）



告警: 到價、漲跌幅門檻（瀏覽器通知可選）



簡易走勢: 每檔保留近 N 分鐘的 1 秒序列，做 sparkline（非完整 K 線）

架構總覽（可替換行情供應商）

flowchart LR
  provider[MarketDataProvider] --> adapter[ProviderAdapter]
  adapter --> ingest[Ingestor]
  ingest --> state[InMemoryStateStore]
  state --> aggregator[SnapshotAggregator_1s]
  aggregator --> ws[WebSocketBroadcast]
  ws --> ui[ReactUI]
  ui --> storage[LocalStorage_Watchlist]
  state --> alerts[AlertEngine]
  alerts --> ws





ProviderAdapter: 專門做「外部行情訊息 → 內部統一事件格式」的轉換；未來只要換這層就能接不同券商/行情商。



InMemoryStateStore: 以股票代號為 key 維護最新狀態（成交價、五檔、量等）。



SnapshotAggregator_1s: 每 1 秒產生一次整包快照（只針對自選股或已訂閱清單）。



WebSocketBroadcast: 對前端推送快照與告警事件。

後端（Go）詳細設計

專案結構（建議）





backend/cmd/server/：啟動入口



backend/internal/provider/：





provider.go：MarketDataProvider 介面



ws_provider/：未來接真 WebSocket 行情（依供應商實作）



mock_provider/：本機可用的 mock/回放（先把 UI 與資料流跑起來）



backend/internal/domain/：統一資料模型（Quote、OrderBookL1、Snapshot、Alert）



backend/internal/state/：in-memory store（map + RWMutex 或 sync.Map）



backend/internal/agg/：1 秒快照聚合器、序列緩衝（sparkline）



backend/internal/alerts/：告警規則與觸發



backend/internal/ws/：WebSocket hub（連線管理、訂閱管理、廣播）



backend/internal/http/：健康檢查、（可選）REST API

內部統一事件/模型（供應商無關）





MarketEvent（供應商推播進來的最小事件）





type: trade | quote | book



symbol



ts



payload: 依 type



QuoteState（最新狀態）





lastPrice, prevClose, open, high, low



volume



change, changePct



book: bid[5], ask[5]



lastUpdateTs



Snapshot（推給前端）





ts



symbols: { [symbol]: QuoteState }



spark: { [symbol]: [ {t, price} ... ] }（近 N 分鐘 1 秒序列）

前後端 WebSocket 訊息協定（建議）





client → server





subscribe: {type:"subscribe", symbols:["2330","2317"]}



unsubscribe: {type:"unsubscribe", symbols:[...]}



setAlerts: {type:"setAlerts", rules:[...]}（可先簡化：只在前端做告警）



server → client





snapshot: {type:"snapshot", ts, symbols:{...}, spark:{...}}



alert: {type:"alert", ts, symbol, ruleId, message, data}



status: {type:"status", message}



建議第一版 告警規則先放前端（跟 watchlist 一起存 localStorage、體驗快），後端只負責推快照；等需求變複雜再把 alerts 下沉到後端。

性能與穩定性策略（幾十檔規模）





寫入高頻、讀出固定頻率：provider 來一筆就更新 state；aggregator 每 1 秒讀一次並生成快照。



只對已訂閱 symbols 聚合：避免未來擴展時全市場掃描。



重連策略：provider 斷線自動重連 + 指數退避；前端 WS 斷線自動重連。

前端（React）詳細設計

專案結構（建議）





frontend/src/api/wsClient.ts：WebSocket client（重連、心跳、訊息解析）



frontend/src/store/：狀態管理（Zustand/Redux 擇一；規模不大推薦 Zustand）



frontend/src/components/





WatchlistTable.tsx



OrderBook5.tsx



Sparkline.tsx



AlertsPanel.tsx



SymbolSearch.tsx



frontend/src/pages/Dashboard.tsx



frontend/src/utils/localStorage.ts

UI 與互動





表格：可排序欄位（漲跌幅/成交量/價格）、搜尋框。



五檔：點選某檔表格列 → 右側顯示該檔五檔。



走勢：每檔 sparkline（顯示近 15–60 分鐘，依你想要的密度）。



告警：每檔可設定到價/漲跌幅；觸發後在 UI 標示並可選用瀏覽器通知。



持久化：watchlist + alerts 規則存 localStorage，刷新頁面仍保留。

本機開發與啟動方式（規劃）





後端：Go server（預設 :8080），提供 /healthz + WebSocket /ws



前端：Vite dev server（預設 :5173），透過代理或直接連 ws://localhost:8080/ws

行情供應商接入（你未選定前的處理方式）





先做 mock_provider：





產生合理的 L1 更新（含五檔），用於 UI/資料流驗證



或支援「回放檔」（CSV/JSON lines）便於重現



之後接真供應商時，只需要新增 ws_provider/<vendor>/，把對方訊息 mapping 成 MarketEvent。

風險與決策點（先列出，避免踩雷）





授權/條款：正式接入前需確認是否允許自用看盤與顯示。



資料欄位差異：五檔是否提供、成交量口徑、時間戳來源。



市場時段：盤中/盤後/試撮與資料更新規則。

里程碑（從 0 到可用）





M0：後端 mock_provider + state store + 每秒快照 WS 推送



M1：React 看盤頁（表格、排序篩選、自選股 localStorage）



M2：五檔面板 + sparkline



M3：告警（先前端）+ UI polish



M4：接入真實 provider adapter（你選定供應商後）

