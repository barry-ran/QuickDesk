// Failure-path check for session recording.
//
// Records a real session but aborts the artifact upload in the browser, then
// asserts that the index row ends up "failed" with no media instead of being
// left claiming a recording that does not exist.
//
//   QD_TOKEN=<access_token> QD_DEVICE=<device_id> node record-failure.mjs

import { chromium } from 'playwright'

const APP = process.env.QD_APP || 'http://127.0.0.1:5173'
const SERVER = process.env.QD_SERVER || 'ws://10.211.55.2:8000'
const TOKEN = process.env.QD_TOKEN
const DEVICE = process.env.QD_DEVICE
const RECORD_MS = Number(process.env.QD_RECORD_MS || 6000)

if (!TOKEN || !DEVICE) {
  console.error('QD_TOKEN and QD_DEVICE are required')
  process.exit(2)
}

const api = SERVER.replace(/^ws/, 'http')
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

// Fail only the artifact upload; creating the index row still succeeds.
await remote.route('**/v1/me/recordings/*/artifact', (route) => route.abort('failed'))

const deadline = Date.now() + 60000
let ready = false
while (Date.now() < deadline) {
  const stats = await remote.evaluate(() => {
    const v = document.getElementById('remoteVideo')
    return { w: v.videoWidth, t: v.currentTime }
  })
  if (stats.w > 0 && stats.t > 0) { ready = true; break }
  await remote.waitForTimeout(1500)
}
if (!ready) {
  console.error('no video stream')
  process.exit(1)
}

await remote.click('#btnRecord')
await remote.waitForTimeout(RECORD_MS)
await remote.click('#btnRecord')

// Wait for the client to give up on the upload.
const failDeadline = Date.now() + 45000
while (Date.now() < failDeadline) {
  if (logs.some((l) => l.includes('上传失败') || l.includes('upload failed'))) break
  await remote.waitForTimeout(1000)
}

// The newest row must be failed, with no media attached.
let newest = null
for (let i = 0; i < 10 && !newest; i++) {
  const res = await context.request.get(`${api}/v1/me/recordings?limit=1`, {
    headers: { Authorization: `Bearer ${TOKEN}`, Origin: APP },
  })
  const body = await res.json()
  newest = body.items?.[0] || null
  if (newest && newest.status !== 'failed') {
    await page.waitForTimeout(1000)
    newest = null
  }
}

const ok = !!newest && newest.status === 'failed' && !newest.artifact_url && newest.size_bytes === 0
console.log('newest row:', JSON.stringify(newest))
console.log(logs.slice(-3).join('\n'))
console.log(ok ? 'RESULT=PASS' : 'RESULT=FAIL')

await browser.close()
process.exit(ok ? 0 : 1)
