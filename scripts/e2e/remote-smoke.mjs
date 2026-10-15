// go-remote-next end-to-end smoke test.
//
// Drives the Vue WebClient in a real Chrome against a self-hosted signaling
// server, connects to a bound device, and proves that decoded video frames
// arrive. Optionally types into the remote session to prove input control.
//
//   QD_TOKEN=<access_token> QD_DEVICE=<device_id> node remote-smoke.mjs
//
// Environment:
//   QD_APP       WebClient origin     (default http://127.0.0.1:5173)
//   QD_SERVER    signaling ws:// url  (default ws://10.211.55.2:8000)
//   QD_TOKEN     user access token    (required)
//   QD_DEVICE    bound device id      (required)
//   QD_TYPE_TEXT optional; typed into the remote session after connect
//   QD_OUT       artifact directory   (default cwd)
//   QD_WAIT_MS   video wait budget ms (default 60000)
//
// Requires playwright + a locally installed Chrome (channel: 'chrome').

import { chromium } from 'playwright'
import fs from 'node:fs'

const APP = process.env.QD_APP || 'http://127.0.0.1:5173'
const SERVER = process.env.QD_SERVER || 'ws://10.211.55.2:8000'
const TOKEN = process.env.QD_TOKEN
const DEVICE = process.env.QD_DEVICE
const TYPE_TEXT = process.env.QD_TYPE_TEXT || ''
const OUT = process.env.QD_OUT || process.cwd()
const WAIT_MS = Number(process.env.QD_WAIT_MS || 60000)

if (!TOKEN || !DEVICE) {
  console.error('QD_TOKEN and QD_DEVICE are required')
  process.exit(2)
}

const log = (...a) => console.log('[e2e]', ...a)
const browser = await chromium.launch({ channel: 'chrome', headless: true })
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await context.newPage()
page.on('pageerror', (e) => log('pageerror:', e.message))
page.on('console', (m) => {
  if (m.type() === 'error') log('console.error:', m.text())
})

await page.goto(APP, { waitUntil: 'domcontentloaded' })
await page.evaluate(
  ({ server, token }) => {
    localStorage.setItem('quickdesk_signaling_url', server)
    localStorage.setItem('quickdesk_user_access_token', token)
  },
  { server: SERVER, token: TOKEN }
)

// Full reload is required: the SPA caches the signaling base URL at module
// init, so a hash-only navigation would keep the default public server.
await page.reload({ waitUntil: 'domcontentloaded' })
await page.goto(`${APP}/#/devices`, { waitUntil: 'networkidle' })
await page.waitForSelector('.device-item', { timeout: 20000 })
await page.screenshot({ path: `${OUT}/e2e-01-devices.png` })

const row = page.locator('.device-item', { hasText: DEVICE }).first()
const rowText = (await row.innerText()).replace(/\s+/g, ' ').trim()
log('device row:', rowText)

const popupPromise = context.waitForEvent('page', { timeout: 30000 })
await row.getByRole('button').first().click()
const remote = await popupPromise
await remote.waitForLoadState('domcontentloaded')
log('remote url:', remote.url())

await remote.waitForSelector('#remoteVideo', { timeout: 30000 })

// Wait until the decoded video reports real dimensions and advancing time.
const deadline = Date.now() + WAIT_MS
let stats = { w: 0, h: 0, t: 0, t2: 0 }
while (Date.now() < deadline) {
  stats = await remote.evaluate(() => {
    const v = document.getElementById('remoteVideo')
    return { w: v.videoWidth, h: v.videoHeight, t: v.currentTime, paused: v.paused }
  })
  if (stats.w > 0 && stats.h > 0 && stats.t > 0) break
  await remote.waitForTimeout(1500)
}

if (stats.w > 0) {
  await remote.waitForTimeout(3000)
  stats.t2 = await remote.evaluate(() => document.getElementById('remoteVideo').currentTime)
}

await remote.screenshot({ path: `${OUT}/e2e-02-remote.png` })
log('video stats:', JSON.stringify(stats))

if (TYPE_TEXT) {
  // Click a configurable point inside the streamed desktop. The centre is a
  // bad default on a fresh Windows guest: a Windows Security / firewall
  // consent dialog usually sits there and steals focus.
  const [fx, fy] = (process.env.QD_CLICK_FRAC || '0.12,0.20')
    .split(',')
    .map((v) => Number(v.trim()))
  const box = await remote.locator('#videoContainer').boundingBox()
  await remote.mouse.click(box.x + box.width * fx, box.y + box.height * fy)
  await remote.waitForTimeout(800)
  if (process.env.QD_CLEAR_FIRST === '1') {
    await remote.keyboard.press('Control+A')
    await remote.keyboard.press('Delete')
    await remote.waitForTimeout(400)
  }
  // Press keys explicitly. Playwright's keyboard.type() synthesises characters
  // without a real Shift keypress, so uppercase would arrive lowercase and
  // make the test lie about modifier handling.
  for (const ch of TYPE_TEXT) {
    if (/[A-Z]/.test(ch)) await remote.keyboard.press(`Shift+Key${ch}`)
    else if (/[a-z]/.test(ch)) await remote.keyboard.press(`Key${ch.toUpperCase()}`)
    else if (/[0-9]/.test(ch)) await remote.keyboard.press(`Digit${ch}`)
    else if (ch === '-') await remote.keyboard.press('Minus')
    else if (ch === ' ') await remote.keyboard.press('Space')
    else await remote.keyboard.insertText(ch)
    await remote.waitForTimeout(60)
  }
  await remote.waitForTimeout(1500)
  await remote.screenshot({ path: `${OUT}/e2e-03-input.png` })
  log('typed into remote session:', TYPE_TEXT)
}

const ok = stats.w > 0 && stats.h > 0 && stats.t2 > stats.t
log(ok ? 'RESULT=PASS' : 'RESULT=FAIL')

fs.writeFileSync(
  `${OUT}/e2e-result.json`,
  JSON.stringify(
    { app: APP, server: SERVER, device: DEVICE, rowText, video: stats, typed: TYPE_TEXT || null, ok },
    null,
    2
  )
)

await browser.close()
process.exit(ok ? 0 : 1)
