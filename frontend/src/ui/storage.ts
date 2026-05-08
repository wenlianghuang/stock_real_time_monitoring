const WATCHLIST_KEY = 'twstock.watchlist.v1'
const ALERTS_KEY = 'twstock.alerts.v1'

export type AlertRule = {
  id: string
  symbol: string
  kind: 'price_gte' | 'price_lte' | 'pct_gte' | 'pct_lte'
  value: number
  enabled: boolean
}

export function loadWatchlist(): string[] {
  try {
    const raw = localStorage.getItem(WATCHLIST_KEY)
    if (!raw) return ['2330', '2317']
    const arr = JSON.parse(raw) as unknown
    if (!Array.isArray(arr)) return ['2330', '2317']
    return arr.filter((x) => typeof x === 'string' && x.length > 0)
  } catch {
    return ['2330', '2317']
  }
}

export function saveWatchlist(symbols: string[]) {
  localStorage.setItem(WATCHLIST_KEY, JSON.stringify(symbols))
}

export function loadAlertRules(): AlertRule[] {
  try {
    const raw = localStorage.getItem(ALERTS_KEY)
    if (!raw) return []
    const arr = JSON.parse(raw) as unknown
    if (!Array.isArray(arr)) return []
    return arr as AlertRule[]
  } catch {
    return []
  }
}

export function saveAlertRules(rules: AlertRule[]) {
  localStorage.setItem(ALERTS_KEY, JSON.stringify(rules))
}

