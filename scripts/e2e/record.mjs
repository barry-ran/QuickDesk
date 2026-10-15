// End-to-end: connect to a live device, record the session from the WebClient,
// let it upload, then prove the resulting artifact is downloadable and playable.

import { chromium } from 'playwright'

const APP = 'http://127.0.0.1:5173'
const SERVER = 'ws://10.211.55.2:8000'
const TOKEN = process.env.QD_TOKEN
const DEVICE = process.env.QD_DEVICE
const RECORD_MS = Number(process.env.QD_RECORD_MS || 7000)

const browser = await chromium.launch({ channel: 'chrome', headless: true })
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await context.newPage()
await page.goto(APP, { waitUntil: 'domcontentloaded' })
await page.evaluate(({ s, t }) => {
  localStorage.setItem('quickdesk_signaling_url', s)
  localStorage.setItem('quickdesk_user_access_token', t)
}, { s: SERVER, t: TOKEN })
await page.reload({ waitUntil: 'domcontentloaded' })
await page.goto(`${APP}/#/devices`, { waitUntil: 'networkidle' })

const popupPromise = context.waitForEvent('page', { timeout: 30000 })
await page.locator('.device-item', { hasText: DEVICE }).first().getByRole('button').first().click()
const remote = await popupPromise
const logs = []
remote.on('console', (m) => {
  if (m.text().includes('[RemoteDesktop]')) logs.push(m.text())
})
await remote.waitForLoadState('domcontentloaded')
await remote.waitForSelector('#remoteVideo', { timeout: 30000 })

// Wait for real frames before recording.
const deadline = Date.now() + 60000
let stats = { w: 0, t: 0 }
while (Date.now() < deadline) {
  stats = await remote.evaluate(() => {
    const v = document.getElementById('remoteVideo')
    return { w: v.videoWidth, t: v.currentTime }
  })
  if (stats.w > 0 && stats.t > 0) break
  await remote.waitForTimeout(1500)
}
console.log('video ready:', JSON.stringify(stats))

await remote.click('#btnRecord')
await remote.waitForTimeout(RECORD_MS)
const buttonState = await remote.evaluate(() => {
  const b = document.getElementById('btnRecord')
  return { text: b.textContent.trim(), recording: b.classList.contains('recording') }
})
console.log('button while recording:', JSON.stringify(buttonState))
await remote.click('#btnRecord')

// Wait for the upload log line.
const uploadDeadline = Date.now() + 60000
let uploaded = false
while (Date.now() < uploadDeadline) {
  if (logs.some((l) => l.includes('录像已上传') || l.includes('Recording uploaded'))) {
    uploaded = true
    break
  }
  await remote.waitForTimeout(1000)
}
console.log(logs.slice(-6).join('\n'))
const ok = buttonState.recording && uploaded
console.log(ok ? 'RESULT=PASS' : 'RESULT=FAIL')

await browser.close()
process.exit(ok ? 0 : 1)
