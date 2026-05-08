const { WebSocketClient } = require('@fugle/marketdata');
const client = new WebSocketClient({ apiKey: 'Zjc1OWNiZTktNjZjZS00MmVmLWE4MzItMWI2MGU1YWJiNGU4IGU3NWE5YjgyLWUxZWItNGFhZi1hYmEyLWM2MjM4ZTAxMTAwZg==' });
const stock = client.stock;
stock.on('message', (m) => console.log('message', m));
stock.on('error', (e) => console.log('error', e));
stock.on('disconnect', (code, msg) => console.log('disconnect', code, msg));
stock.connect().then(() => {
  console.log('connected');
  stock.subscribe({ channel: 'aggregates', symbol: '2330' });
});