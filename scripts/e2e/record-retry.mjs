// Prove the recorder survives a transient transport failure: the first two
// artifact attempts and the first preview attempt are aborted at the browser
// level, and the capture must still end as a completed row with a thumbnail.
//
// This is the failure the retry exists for - a dropped connection - as opposed
// to a rejected payload (HTTP 4xx), which must NOT be retried.

import { chromium } from 'playwright'

const APP = 'http://127.0.0.1:5173'
const SERVER = 'ws://10.211.55.2:8000'
const API = process.env.QD_API || 'http://127.0.0.1:8000'
const API_KEY = process.env.QD_API_KEY || 'qd-local-dev-key'
const TOKEN = process.env.QD_TOKEN
const DEVICE = process.env.QD_DEVICE
const RECORD_MS = Number(process.env.QD_RECORD_MS || 5000)

const browser = await chromium.launch({ channel: 'chrome', headless: true })
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await context.newPage()

let artifactAborts = 2
let thumbnailAborts = 1
await context.route('**/v1/me/recordings/*/artifact', async (route) => {
  if (artifactAborts > 0) {
    artifactAborts -= 1
    console.log(`[inject] aborting artifact upload (${2 - artifactAborts}/2)`)
    await route.abort('connectionfailed')
    return
  }
  await route.continue()
})
await context.route('**/v1/me/recordings/*/thumbnail', async (route) => {
  if (thumbnailAborts > 0) {
    thumbnailAborts -= 1
    console.log('[inject] aborting thumbnail upload')
    await route.abort('connectionfailed')
    return
  }
  await route.continue()
})

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
await remote.click('#btnRecord')

const uploadDeadline = Date.now() + 90000
let uploaded = false
while (Date.now() < uploadDeadline) {
  if (logs.some((l) => /录像已上传|Recording uploaded/.test(l))) { uploaded = true; break }
  await remote.waitForTimeout(1000)
}

// The preview is uploaded after the media, so its retry lands slightly later:
// wait for the row to actually carry a thumbnail before judging the outcome.
let row = null
const rowDeadline = Date.now() + 30000
while (uploaded && Date.now() < rowDeadline) {
  const list = await fetch(`${API}/v1/me/recordings?limit=5`, {
    headers: { Authorization: `Bearer ${TOKEN}`, 'X-API-Key': API_KEY },
  })
  const body = await list.json()
  row = (body.items || [])[0] || null
  if (row && row.thumbnail_url) break
  await remote.waitForTimeout(1000)
}
const retried = logs.filter((l) => /正在重试|retrying/.test(l))
console.log(logs.slice(-8).join('\n'))
console.log(`retry log lines: ${retried.length}`)
console.log('newest row:', JSON.stringify(row && {
  id: row.id, status: row.status, size_bytes: row.size_bytes,
  thumbnail_url: row.thumbnail_url, artifact_url: row.artifact_url,
}))

const ok = uploaded && retried.length >= 3 && !!row && row.status === 'completed' && !!row.thumbnail_url
console.log(ok ? 'RESULT=PASS' : 'RESULT=FAIL')
await browser.close()
process.exit(ok ? 0 : 1)
