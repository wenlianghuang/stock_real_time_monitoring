import { useState } from 'react'
import type { FormEvent } from 'react'

type Panel = 'login' | 'register'

export function LoginForm({ onLoggedIn }: { onLoggedIn: (username: string) => void }) {
  const [panel, setPanel] = useState<Panel>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  function setMode(next: Panel) {
    setPanel(next)
    setError(null)
    setSuccess(null)
  }

  async function onLoginSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSuccess(null)
    setBusy(true)
    try {
      const res = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      let data: { ok?: boolean; username?: string; error?: string } = {}
      try {
        data = (await res.json()) as typeof data
      } catch {
        // non-JSON body
      }
      if (!res.ok) {
        setError(data.error ?? `登入失敗（${res.status}）`)
        return
      }
      const u = data.username?.trim()
      if (!u) {
        setError('伺服器回應異常')
        return
      }
      sessionStorage.setItem('loginUsername', u)
      setSuccess(`歡迎，${u}`)
      window.setTimeout(() => onLoggedIn(u), 400)
    } finally {
      setBusy(false)
    }
  }

  async function onRegisterSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSuccess(null)
    setBusy(true)
    try {
      const res = await fetch('/api/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      let data: { ok?: boolean; username?: string; error?: string } = {}
      try {
        data = (await res.json()) as typeof data
      } catch {
        // non-JSON body
      }
      if (res.status === 409) {
        setError('此帳號已被使用，請換一個')
        return
      }
      if (!res.ok) {
        if (data.error === 'username taken') {
          setError('此帳號已被使用，請換一個')
          return
        }
        setError(data.error ?? `註冊失敗（${res.status}）`)
        return
      }
      const u = data.username?.trim()
      if (!u) {
        setError('伺服器回應異常')
        return
      }
      setSuccess(`註冊成功：${u}。可前往登入`)
      setPassword('')
      window.setTimeout(() => setMode('login'), 1600)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="card loginCard">
      <div className="authTabs" role="tablist" aria-label="登入或註冊">
        <button
          type="button"
          className={panel === 'login' ? 'authTab authTabActive' : 'authTab'}
          onClick={() => setMode('login')}
          role="tab"
          aria-selected={panel === 'login'}
        >
          登入
        </button>
        <button
          type="button"
          className={panel === 'register' ? 'authTab authTabActive' : 'authTab'}
          onClick={() => setMode('register')}
          role="tab"
          aria-selected={panel === 'register'}
        >
          註冊
        </button>
      </div>
      <div className="cardTitle">{panel === 'login' ? '登入' : '建立新帳號'}</div>
      <form onSubmit={panel === 'login' ? onLoginSubmit : onRegisterSubmit}>
        <div className="fieldStack">
          <label className="fieldLabel">
            帳號
            <input
              className="input"
              type="text"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="帳號"
              style={{ width: '100%', marginTop: 6 }}
            />
          </label>
          <label className="fieldLabel">
            密碼
            <input
              className="input"
              type="password"
              autoComplete={panel === 'login' ? 'current-password' : 'new-password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              style={{ width: '100%', marginTop: 6 }}
            />
          </label>
        </div>
        <div className="row" style={{ marginTop: 14 }}>
          <button type="submit" disabled={busy}>
            {busy ? '處理中…' : panel === 'login' ? '登入' : '註冊'}
          </button>
        </div>
        <div className="authHint">
          {panel === 'login' ? (
            <>
              沒有帳號？
              <button type="button" className="linkBtn" onClick={() => setMode('register')}>
                建立新帳號
              </button>
            </>
          ) : (
            <>
              已有帳號？
              <button type="button" className="linkBtn" onClick={() => setMode('login')}>
                返回登入
              </button>
            </>
          )}
        </div>
        {error ? <div className="loginError">{error}</div> : null}
        {success ? <div className="loginSuccess">{success}</div> : null}
      </form>
    </div>
  )
}
