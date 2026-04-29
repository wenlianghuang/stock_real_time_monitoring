import { useMemo, useState } from 'react'
import type { AlertRule } from '../storage'
import type { QuoteState } from '../types'

export function AlertsPanel({
  rules,
  quotes,
  onChange,
  symbol,
}: {
  rules: AlertRule[]
  quotes: Record<string, QuoteState>
  onChange: (rules: AlertRule[]) => void
  symbol?: string
}) {
  const [kind, setKind] = useState<AlertRule['kind']>('price_gte')
  const [value, setValue] = useState<string>('0')

  const currentSymbol = symbol ?? ''
  const currentPrice = currentSymbol ? quotes[currentSymbol]?.lastPrice : undefined

  const rows = useMemo(() => rules.slice().sort((a, b) => a.symbol.localeCompare(b.symbol)), [rules])

  const addRule = () => {
    if (!currentSymbol) return
    const v = Number(value)
    if (!Number.isFinite(v)) return
    const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`
    onChange([{ id, symbol: currentSymbol, kind, value: v, enabled: true }, ...rules])
  }

  const toggle = (id: string) => onChange(rules.map((r) => (r.id === id ? { ...r, enabled: !r.enabled } : r)))
  const remove = (id: string) => onChange(rules.filter((r) => r.id !== id))

  return (
    <div className="card">
      <div className="cardTitle">告警</div>
      <div className="row">
        <div style={{ flex: 1, opacity: currentSymbol ? 1 : 0.6 }}>
          <div style={{ fontSize: 12, opacity: 0.8 }}>目前選擇</div>
          <div style={{ fontWeight: 600 }}>
            {currentSymbol || '（請先點選一檔股票）'}
            {typeof currentPrice === 'number' ? <span style={{ marginLeft: 8, opacity: 0.85 }}>{currentPrice.toFixed(1)}</span> : null}
          </div>
        </div>
        <select value={kind} onChange={(e) => setKind(e.target.value as AlertRule['kind'])} disabled={!currentSymbol}>
          <option value="price_gte">價格 ≥</option>
          <option value="price_lte">價格 ≤</option>
          <option value="pct_gte">漲跌幅 ≥（小數）</option>
          <option value="pct_lte">漲跌幅 ≤（小數）</option>
        </select>
        <input value={value} onChange={(e) => setValue(e.target.value)} placeholder="value" style={{ width: 120 }} disabled={!currentSymbol} />
        <button onClick={addRule} disabled={!currentSymbol}>
          新增
        </button>
      </div>

      <div className="tableWrap" style={{ marginTop: 8 }}>
        <table className="table">
          <thead>
            <tr>
              <th>Symbol</th>
              <th>Rule</th>
              <th>Value</th>
              <th>On</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id}>
                <td>{r.symbol}</td>
                <td>{r.kind}</td>
                <td>{r.value}</td>
                <td>
                  <input type="checkbox" checked={r.enabled} onChange={() => toggle(r.id)} />
                </td>
                <td>
                  <button onClick={() => remove(r.id)}>刪除</button>
                </td>
              </tr>
            ))}
            {rows.length === 0 ? (
              <tr>
                <td colSpan={5} style={{ opacity: 0.7, padding: 12 }}>
                  尚未設定告警
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      <div style={{ fontSize: 12, opacity: 0.7, marginTop: 8 }}>
        漲跌幅用「小數」：例如 3% 請填 <code>0.03</code>。
      </div>
    </div>
  )
}

