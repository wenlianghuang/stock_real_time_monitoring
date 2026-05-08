export type BookLevel = {
  price: number
  size: number
}

export type OrderBook5 = {
  bids: BookLevel[]
  asks: BookLevel[]
}

export type QuoteState = {
  symbol: string
  name: string
  lastPrice: number
  prevClose: number
  open: number
  high: number
  low: number
  volume: number
  change: number
  changePct: number
  book: OrderBook5
  lastUpdateTs: number
}

export type SparkPoint = { t: number; p: number }

export type SnapshotMsg = {
  type: 'snapshot'
  ts: number
  symbols: Record<string, QuoteState>
  spark?: Record<string, SparkPoint[]>
}

export type StatusMsg = { type: 'status'; message: string }
export type AlertMsg = {
  type: 'alert'
  ts: number
  symbol: string
  ruleId: string
  message: string
  data?: unknown
}

export type ServerMsg = SnapshotMsg | StatusMsg | AlertMsg

export type SubscribeMsg = { type: 'subscribe'; symbols: string[] }
export type UnsubscribeMsg = { type: 'unsubscribe'; symbols: string[] }
export type PingMsg = { type: 'ping'; ts: number }
export type ClientMsg = SubscribeMsg | UnsubscribeMsg | PingMsg

