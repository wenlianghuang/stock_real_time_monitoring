import { useMemo } from 'react'
import { displayPrice } from '../displayPrice'
import type { QuoteState, SparkPoint } from '../types'
import { Sparkline } from './Sparkline'

export function WatchlistTable({
  symbols,
  quotes,
  spark,
  filter,
  sortKey,
  sortDir,
  selectedSymbol,
  onSelect,
  onSort,
}: {
  symbols: string[]
  quotes: Record<string, QuoteState>
  spark: Record<string, SparkPoint[]>
  filter: string
  sortKey: 'symbol' | 'lastPrice' | 'changePct' | 'volume'
  sortDir: 'asc' | 'desc'
  selectedSymbol?: string
  onSelect: (sym: string) => void
  onSort: (key: 'symbol' | 'lastPrice' | 'changePct' | 'volume') => void
}) {
  const rows = useMemo(() => {
    const f = filter.trim().toLowerCase()
    const list = symbols
      .map((sym) => quotes[sym])
      .filter(Boolean)
      .filter((q) => (f ? q.symbol.toLowerCase().includes(f) || q.name.toLowerCase().includes(f) : true))

    const dir = sortDir === 'asc' ? 1 : -1
    list.sort((a, b) => {
      let av = 0
      let bv = 0
      if (sortKey === 'symbol') {
        return dir * a.symbol.localeCompare(b.symbol)
      }
      if (sortKey === 'lastPrice') {
        av = displayPrice(a)
        bv = displayPrice(b)
      }
      if (sortKey === 'changePct') {
        av = a.changePct
        bv = b.changePct
      }
      if (sortKey === 'volume') {
        av = a.volume
        bv = b.volume
      }
      return dir * (av - bv)
    })
    return list
  }, [filter, quotes, sortDir, sortKey, symbols])

  return (
    <div className="card">
      <div className="cardTitle">自選股</div>
      <div className="tableWrap">
        <table className="table">
          <thead>
            <tr>
              <th className="clickable" onClick={() => onSort('symbol')}>
                代號 {sortKey === 'symbol' ? (sortDir === 'asc' ? '▲' : '▼') : ''}
              </th>
              <th>名稱</th>
              <th className="clickable" onClick={() => onSort('lastPrice')}>
                價格 {sortKey === 'lastPrice' ? (sortDir === 'asc' ? '▲' : '▼') : ''}
              </th>
              <th className="clickable" onClick={() => onSort('changePct')}>
                漲跌(%) {sortKey === 'changePct' ? (sortDir === 'asc' ? '▲' : '▼') : ''}
              </th>
              <th className="clickable" onClick={() => onSort('volume')}>
                量 {sortKey === 'volume' ? (sortDir === 'asc' ? '▲' : '▼') : ''}
              </th>
              <th>走勢</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((q) => {
              const isSel = selectedSymbol === q.symbol
              const up = q.change >= 0
              return (
                <tr key={q.symbol} className={isSel ? 'rowSelected' : ''} onClick={() => onSelect(q.symbol)}>
                  <td>{q.symbol}</td>
                  <td>{q.name}</td>
                  <td>{displayPrice(q).toFixed(1)}</td>
                  <td className={up ? 'pos' : 'neg'}>
                    {q.change.toFixed(1)} ({(q.changePct * 100).toFixed(2)}%)
                  </td>
                  <td>{q.volume.toLocaleString()}</td>
                  <td>
                    <Sparkline points={spark[q.symbol]} />
                  </td>
                </tr>
              )
            })}
            {rows.length === 0 ? (
              <tr>
                <td colSpan={6} style={{ opacity: 0.7, padding: 12 }}>
                  沒有資料（請確認已訂閱 symbols，或後端已啟動）
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  )
}

