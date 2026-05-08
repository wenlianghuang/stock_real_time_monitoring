import { create } from 'zustand'
import type { QuoteState, ServerMsg, SnapshotMsg, SparkPoint } from './types'
import type { AlertRule } from './storage'

export type UIState = {
  connected: boolean
  statusMessage?: string

  watchlist: string[]
  quotes: Record<string, QuoteState>
  spark: Record<string, SparkPoint[]>

  selectedSymbol?: string
  filter: string
  sortKey: 'symbol' | 'lastPrice' | 'changePct' | 'volume'
  sortDir: 'asc' | 'desc'

  alertRules: AlertRule[]
  triggered: { ts: number; ruleId: string; symbol: string; message: string }[]

  setConnected: (v: boolean, msg?: string) => void
  setWatchlist: (syms: string[]) => void
  setSelectedSymbol: (sym?: string) => void
  setFilter: (v: string) => void
  setSort: (key: UIState['sortKey']) => void
  setAlertRules: (rules: AlertRule[]) => void

  ingest: (m: ServerMsg) => void
  evalAlerts: (ts: number) => void
}

export const useUIStore = create<UIState>((set, get) => ({
  connected: false,
  statusMessage: undefined,

  watchlist: [],
  quotes: {},
  spark: {},

  selectedSymbol: undefined,
  filter: '',
  sortKey: 'symbol',
  sortDir: 'asc',

  alertRules: [],
  triggered: [],

  setConnected: (v, msg) => set({ connected: v, statusMessage: msg }),
  setWatchlist: (syms) => set({ watchlist: syms }),
  setSelectedSymbol: (sym) => set({ selectedSymbol: sym }),
  setFilter: (v) => set({ filter: v }),
  setSort: (key) =>
    set((s) => {
      const dir = s.sortKey === key ? (s.sortDir === 'asc' ? 'desc' : 'asc') : 'desc'
      return { sortKey: key, sortDir: dir }
    }),
  setAlertRules: (rules) => set({ alertRules: rules }),

  ingest: (m) => {
    if (m.type === 'snapshot') {
      const snap = m as SnapshotMsg
      set((s) => ({
        quotes: { ...s.quotes, ...snap.symbols },
        spark: snap.spark ? { ...s.spark, ...snap.spark } : s.spark,
      }))
      get().evalAlerts(snap.ts)
      return
    }
    if (m.type === 'status') {
      set({ statusMessage: m.message })
      return
    }
    if (m.type === 'alert') {
      set((s) => ({
        triggered: [
          { ts: m.ts, ruleId: m.ruleId, symbol: m.symbol, message: m.message },
          ...s.triggered,
        ].slice(0, 50),
      }))
      return
    }
  },

  evalAlerts: (ts) => {
    const { alertRules, quotes } = get()
    if (alertRules.length === 0) return

    const newly: UIState['triggered'] = []
    for (const r of alertRules) {
      if (!r.enabled) continue
      const q = quotes[r.symbol]
      if (!q) continue

      let ok = false
      if (r.kind === 'price_gte') ok = q.lastPrice >= r.value
      if (r.kind === 'price_lte') ok = q.lastPrice <= r.value
      if (r.kind === 'pct_gte') ok = q.changePct >= r.value
      if (r.kind === 'pct_lte') ok = q.changePct <= r.value

      if (ok) {
        newly.push({ ts, ruleId: r.id, symbol: r.symbol, message: `${r.kind} ${r.value}` })
      }
    }

    if (newly.length > 0) {
      set((s) => ({ triggered: [...newly, ...s.triggered].slice(0, 50) }))
    }
  },
}))

