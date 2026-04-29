import type { ClientMsg, ServerMsg } from './types'

type Handlers = {
  onMessage: (m: ServerMsg) => void
  onStatus?: (s: { connected: boolean; message?: string }) => void
}

export class WSClient {
  private url: string
  private ws: WebSocket | null = null
  private reconnectTimer: number | null = null
  private backoffMs = 500
  private handlers: Handlers
  private queue: ClientMsg[] = []

  constructor(url: string, handlers: Handlers) {
    this.url = url
    this.handlers = handlers
  }

  connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    this.ws = new WebSocket(this.url)
    this.handlers.onStatus?.({ connected: false, message: 'connecting' })

    this.ws.onopen = () => {
      this.backoffMs = 500
      this.handlers.onStatus?.({ connected: true, message: 'connected' })
      this.flushQueue()
    }

    this.ws.onclose = () => {
      this.handlers.onStatus?.({ connected: false, message: 'disconnected' })
      this.scheduleReconnect()
    }

    this.ws.onerror = () => {
      this.handlers.onStatus?.({ connected: false, message: 'error' })
    }

    this.ws.onmessage = (ev) => {
      try {
        const m = JSON.parse(ev.data as string) as ServerMsg
        this.handlers.onMessage(m)
      } catch {
        // ignore
      }
    }
  }

  close() {
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
    this.ws = null
  }

  send(m: ClientMsg) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      this.queue.push(m)
      return
    }
    this.ws.send(JSON.stringify(m))
  }

  private flushQueue() {
    const q = this.queue
    this.queue = []
    for (const m of q) this.send(m)
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    const wait = this.backoffMs
    this.backoffMs = Math.min(this.backoffMs * 2, 10_000)
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, wait)
  }
}

