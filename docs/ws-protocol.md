# WebSocket Protocol（前後端通用）

目標：後端維護最新 state，並以 **1 秒快照**推送給前端；前端用 `subscribe` 指定關注的股票代號清單。

## URL

- `ws://localhost:8080/ws`

## Client → Server

### subscribe

```json
{"type":"subscribe","symbols":["2330","2317"]}
```

### unsubscribe

```json
{"type":"unsubscribe","symbols":["2330"]}
```

### ping（可選）

```json
{"type":"ping","ts":1710000000000}
```

## Server → Client

### snapshot（每秒推送）

```json
{
  "type":"snapshot",
  "ts":1710000000000,
  "symbols":{
    "2330":{
      "symbol":"2330",
      "name":"台積電",
      "lastPrice":998.0,
      "prevClose":990.0,
      "open":995.0,
      "high":1001.0,
      "low":992.0,
      "volume":123456,
      "change":8.0,
      "changePct":0.00808,
      "book":{
        "bids":[{"price":997.0,"size":120},{"price":996.0,"size":80}],
        "asks":[{"price":998.0,"size":60},{"price":999.0,"size":90}]
      },
      "lastUpdateTs":1710000000000
    }
  },
  "spark":{
    "2330":[{"t":1710000000000,"p":998.0},{"t":1710000001000,"p":998.5}]
  }
}
```

- `symbols`：只包含「該連線已訂閱」的代號。
- `spark`：只包含已訂閱代號的近 N 分鐘（1 秒粒度）價格序列，用於 sparkline。

### alert（第一版可先由前端自己判斷；此訊息保留擴充）

```json
{
  "type":"alert",
  "ts":1710000000000,
  "symbol":"2330",
  "ruleId":"r1",
  "message":"price >= 1000",
  "data":{"lastPrice":1000.0}
}
```

### status

```json
{"type":"status","message":"connected"}
```

