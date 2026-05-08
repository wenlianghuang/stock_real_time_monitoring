import type { QuoteState } from './types'

/** 盤前或尚無成交時 lastPrice 常為 0，畫面改以昨收為參考價 */
export function displayPrice(q: QuoteState): number {
  if (q.lastPrice > 0) return q.lastPrice
  if (q.prevClose > 0) return q.prevClose
  return q.lastPrice
}

export function isShowingPrevCloseFallback(q: QuoteState): boolean {
  return q.lastPrice <= 0 && q.prevClose > 0
}
