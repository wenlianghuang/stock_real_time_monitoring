import { useEffect, useMemo, useRef, useState } from 'react'
import { WSClient } from './wsClient'
import { loadAlertRules, loadWatchlist, saveAlertRules, saveWatchlist } from './storage'
import { displayPrice, isShowingPrevCloseFallback } from './displayPrice'
import { useUIStore } from './store'
import { WatchlistTable } from './components/WatchlistTable'
import { OrderBook5 } from './components/OrderBook5'
import { AlertsPanel } from './components/AlertsPanel'
import { LoginForm } from './components/LoginForm'

function wsUrl() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  // dev: connect directly to Go backend
  if (location.hostname === 'localhost' && location.port === '5173') {
    return `${proto}://localhost:8080/ws`
  }
  // prod: same origin
  return `${proto}://${location.host}/ws`
}

export function App() {
  const [sessionUser, setSessionUser] = useState<string | null>(() => sessionStorage.getItem('loginUsername'))

  const handleLogout = () => {
    sessionStorage.removeItem('loginUsername')
    setSessionUser(null)
  }

  if (!sessionUser) {
    return (
      <div className="page loginPage">
        <LoginForm onLoggedIn={setSessionUser} />
      </div>
    )
  }

  return <Dashboard sessionUser={sessionUser} onLogout={handleLogout} />
}

function Dashboard({ sessionUser, onLogout }: { sessionUser: string; onLogout: () => void }) {
  const {
    connected,
    statusMessage,
    watchlist,
    quotes,
    spark,
    selectedSymbol,
    filter,
    sortKey,
    sortDir,
    alertRules,
    triggered,
    setConnected,
    setWatchlist,
    setSelectedSymbol,
    setFilter,
    setSort,
    setAlertRules,
    ingest,
  } = useUIStore()

  const clientRef = useRef<WSClient | null>(null)

  useEffect(() => {
    const wl = loadWatchlist()
    setWatchlist(wl)
    const rules = loadAlertRules()
    setAlertRules(rules)
    setSelectedSymbol(wl[0])
  }, [setAlertRules, setSelectedSymbol, setWatchlist])

  useEffect(() => {
    saveWatchlist(watchlist)
    if (clientRef.current) {
      clientRef.current.send({ type: 'subscribe', symbols: watchlist })
    }
  }, [watchlist])

  useEffect(() => {
    saveAlertRules(alertRules)
  }, [alertRules])

  useEffect(() => {
    const c = new WSClient(wsUrl(), {
      onMessage: ingest,
      onStatus: ({ connected, message }) => setConnected(connected, message),
    })
    clientRef.current = c
    c.connect()
    return () => c.close()
  }, [ingest, setConnected])

  useEffect(() => {
    if (!clientRef.current) return
    clientRef.current.send({ type: 'subscribe', symbols: watchlist })
  }, [connected])

  const selected = selectedSymbol ? quotes[selectedSymbol] : undefined

  const addSymbol = (sym: string) => {
    const s = sym.trim()
    if (!s) return
    if (watchlist.includes(s)) return
    setWatchlist([s, ...watchlist])
    setSelectedSymbol(s)
  }

  const removeSymbol = (sym: string) => {
    const next = watchlist.filter((x) => x !== sym)
    setWatchlist(next)
    if (selectedSymbol === sym) setSelectedSymbol(next[0])
  }

  const header = useMemo(() => {
    return (
      <div className="topbar">
        <div className="brand">台股看盤（L1）</div>
        <div className={`pill ${connected ? 'pillOk' : 'pillBad'}`}>{connected ? 'connected' : 'offline'}</div>
        <div style={{ opacity: 0.8, fontSize: 12 }}>{statusMessage ?? ''}</div>
        <div style={{ flex: 1 }} />
        <div style={{ opacity: 0.85, fontSize: 12 }}>{sessionUser}</div>
        <button type="button" onClick={onLogout}>
          登出
        </button>
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="搜尋代號/名稱"
          className="input"
          style={{ width: 220 }}
        />
      </div>
    )
  }, [connected, filter, onLogout, sessionUser, setFilter, statusMessage])

  return (
    <div className="page">
      {header}
      <div className="grid">
        <div className="left">
          <div className="row" style={{ marginBottom: 8 }}>
            <SymbolAdd onAdd={addSymbol} />
            <div style={{ flex: 1 }} />
            {selectedSymbol ? (
              <button type="button" onClick={() => removeSymbol(selectedSymbol)} className="danger">
                移除選取
              </button>
            ) : null}
          </div>
          <WatchlistTable
            symbols={watchlist}
            quotes={quotes}
            spark={spark}
            filter={filter}
            sortKey={sortKey}
            sortDir={sortDir}
            selectedSymbol={selectedSymbol}
            onSelect={setSelectedSymbol}
            onSort={setSort}
          />
        </div>

        <div className="right">
          <div className="card">
            <div className="cardTitle">選取摘要</div>
            {selected ? (
              <div className="kpis">
                <div>
                  <div className="kpiLabel">代號</div>
                  <div className="kpiValue">{selected.symbol}</div>
                </div>
                <div>
                  <div className="kpiLabel">名稱</div>
                  <div className="kpiValue">{selected.name}</div>
                </div>
                <div>
                  <div className="kpiLabel">價格</div>
                  <div className="kpiValue">{displayPrice(selected).toFixed(1)}</div>
                  {isShowingPrevCloseFallback(selected) ? (
                    <div style={{ fontSize: 11, opacity: 0.75, marginTop: 4 }}>尚無成交，為昨收參考</div>
                  ) : null}
                </div>
                <div>
                  <div className="kpiLabel">漲跌幅</div>
                  <div className={`kpiValue ${selected.change >= 0 ? 'pos' : 'neg'}`}>{(selected.changePct * 100).toFixed(2)}%</div>
                </div>
              </div>
            ) : (
              <div style={{ opacity: 0.7 }}>尚未選取</div>
            )}
          </div>

          <OrderBook5 book={selected?.book} />

          <AlertsPanel rules={alertRules} quotes={quotes} onChange={setAlertRules} symbol={selectedSymbol} />

          <TriggeredPanel items={triggered} />
        </div>
      </div>
    </div>
  )
}

function SymbolAdd({ onAdd }: { onAdd: (sym: string) => void }) {
  const inputRef = useRef<HTMLInputElement | null>(null)
  return (
    <div className="row">
      <input ref={inputRef} className="input" placeholder="加入代號 (e.g. 2603)" style={{ width: 220 }} />
      <button
        type="button"
        onClick={() => {
          const v = inputRef.current?.value ?? ''
          onAdd(v)
          if (inputRef.current) inputRef.current.value = ''
        }}
      >
        加入
      </button>
    </div>
  )
}

function TriggeredPanel({
  items,
}: {
  items: { ts: number; ruleId: string; symbol: string; message: string }[]
}) {
  return (
    <div className="card">
      <div className="cardTitle">觸發紀錄</div>
      <div className="tableWrap">
        <table className="table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Symbol</th>
              <th>Rule</th>
            </tr>
          </thead>
          <tbody>
            {items.map((x, i) => (
              <tr key={`${x.ruleId}-${x.ts}-${i}`}>
                <td>{new Date(x.ts).toLocaleTimeString()}</td>
                <td>{x.symbol}</td>
                <td>{x.message}</td>
              </tr>
            ))}
            {items.length === 0 ? (
              <tr>
                <td colSpan={3} style={{ opacity: 0.7, padding: 12 }}>
                  尚無觸發
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  )
}
