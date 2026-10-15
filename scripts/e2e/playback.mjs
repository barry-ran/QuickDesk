import { chromium } from 'playwright'

const APP = 'http://127.0.0.1:5173'
const SERVER = 'ws://10.211.55.2:8000'
const TOKEN = process.env.QD_TOKEN

const browser = await chromium.launch({ channel: 'chrome', headless: true })
const context = await browser.newContext({ viewport: { width: 1280, height: 900 } })
const page = await context.newPage()
page.on('pageerror', (e) => console.log('[pageerror]', e.message))

await page.goto(APP, { waitUntil: 'domcontentloaded' })
await page.evaluate(({ s, t }) => {
  localStorage.setItem('quickdesk_signaling_url', s)
  localStorage.setItem('quickdesk_user_access_token', t)
}, { s: SERVER, t: TOKEN })
await page.reload({ waitUntil: 'domcontentloaded' })
await page.goto(`${APP}/#/recordings`, { waitUntil: 'networkidle' })
await page.waitForSelector('.recording-item', { timeout: 20000 })

// The first row that actually has media (Play enabled).
const row = page.locator('.recording-item').filter({ hasText: '已完成' }).first()
console.log('row:', (await row.innerText()).replace(/\s+/g, ' ').trim())

const popupPromise = context.waitForEvent('page', { timeout: 30000 })
await row.getByRole('button', { name: '播放' }).click()
const player = await popupPromise
await player.waitForLoadState('domcontentloaded')
console.log('player url:', player.url())

await player.waitForSelector('video', { timeout: 20000 })
await player.waitForTimeout(2500)

const first = await player.evaluate(() => {
  const v = document.querySelector('video')
  v.muted = true
  v.play().catch(() => {})
  return { w: v.videoWidth, h: v.videoHeight, t: v.currentTime, dur: v.duration }
})
await player.waitForTimeout(2500)
const second = await player.evaluate(() => {
  const v = document.querySelector('video')
  return { w: v.videoWidth, h: v.videoHeight, t: v.currentTime, dur: v.duration }
})

await player.screenshot({ path: '/Users/timbl/PDShare/qd-e2e/playback.png' })
// Duration is asserted, not just progress: a broken frame-timestamp unit
// (milliseconds written where microseconds are expected) still "plays" but
// collapses a multi-second recording into a few milliseconds.
const ok = second.w > 0 && second.t > first.t && second.dur > 1
console.log('playback:', JSON.stringify({ first, second }))
console.log(ok ? 'RESULT=PASS' : 'RESULT=FAIL')

await browser.close()
process.exit(ok ? 0 : 1)
