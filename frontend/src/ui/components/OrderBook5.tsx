import type { OrderBook5 } from '../types'
import { Fragment } from 'react'

export function OrderBook5({ book }: { book?: OrderBook5 }) {
  const bids = book?.bids ?? []
  const asks = book?.asks ?? []

  return (
    <div className="card">
      <div className="cardTitle">最佳五檔</div>
      <div className="bookGrid">
        <div className="bookHead">Bid</div>
        <div className="bookHead">Size</div>
        <div className="bookHead">Ask</div>
        <div className="bookHead">Size</div>
        {Array.from({ length: 5 }).map((_, i) => {
          const b = bids[i]
          const a = asks[i]
          return (
            <Fragment key={i}>
              <div className="bid">{b ? b.price.toFixed(1) : '-'}</div>
              <div className="bid">{b ? b.size : '-'}</div>
              <div className="ask">{a ? a.price.toFixed(1) : '-'}</div>
              <div className="ask">{a ? a.size : '-'}</div>
            </Fragment>
          )
        })}
      </div>
    </div>
  )
}

