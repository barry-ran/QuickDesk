// Two control sessions record the same device at the same time, which is the
// case the "one row per capture" design has to keep straight: separate rows,
// separate artifacts, separate previews, no cross-talk.
//
// QD_SESSIONS=2 by default; raise it to push the server harder.

import { chromium } from 'playwright'

const APP = 'http://127.0.0.1:5173'
const SERVER = 'ws://10.211.55.2:8000'
const API = process.env.QD_API || 'http://127.0.0.1:8000'
const API_KEY = process.env.QD_API_KEY || 'qd-local-dev-key'
const TOKEN = process.env.QD_TOKEN
const DEVICE = process.env.QD_DEVICE
const SESSIONS = Number(process.env.QD_SESSIONS || 2)
const RECORD_MS = Number(process.env.QD_RECORD_MS || 6000)

const browser = await chromium.launch({ channel: 'chrome', headless: true })

async function recordOnce(index) {
  const context = await browser.newContext({ viewport: { width: 1280, height: 800 } })
  const page = await context.newPage()
  await page.goto(APP, { waitUntil: 'domcontentloaded' })
  await page.evaluate(({ s, t }) => {
    localStorage.setItem('quickdesk_signaling_url', s)
    localStorage.setItem('quickdesk_user_access_token', t)
  }, { s: SERVER, t: TOKEN })
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.goto(`${APP}/#/devices`, { waitUntil: 'networkidle' })

  const popupPromise = context.waitForEvent('page', { timeout: 45000 })
  await page.locator('.device-item', { hasText: DEVICE }).first().getByRole('button').first().click()
  const remote = await popupPromise
  const logs = []
  remote.on('console', (m) => {
    if (m.text().includes('[RemoteDesktop]')) logs.push(m.text())
  })
  await remote.waitForLoadState('domcontentloaded')
  await remote.waitForSelector('#remoteVideo', { timeout: 45000 })

  const deadline = Date.now() + 60000
  let frames = { w: 0, t: 0 }
  while (Date.now() < deadline) {
    frames = await remote.evaluate(() => {
      const v = document.getElementById('remoteVideo')
      return { w: v.videoWidth, t: v.currentTime }
    })
    if (frames.w > 0 && frames.t > 0) break
    await remote.waitForTimeout(1500)
  }

  await remote.click('#btnRecord')
  await remote.waitForTimeout(1200)
  const armed = await remote.evaluate(() => {
    const b = document.getElementById('btnRecord')
    return { recording: !!b && b.classList.contains('recording'), text: b ? b.textContent.trim() : '' }
  })
  return {
    index,
    context,
    remote,
    logs,
    frames,
    armed,
    stop: async () => {
      await remote.click('#btnRecord')
      const uploadDeadline = Date.now() + 90000
      while (Date.now() < uploadDeadline) {
        if (logs.some((l) => /录像已上传|Recording uploaded/.test(l))) return true
        await remote.waitForTimeout(1000)
      }
      return false
    },
  }
}

console.log(`starting ${SESSIONS} concurrent recordings against device ${DEVICE}`)
const sessions = []
for (let i = 0; i < SESSIONS; i++) {
  const session = await recordOnce(i)
  sessions.push(session)
  console.log(`session ${i}: video=${session.frames.w}x${session.frames.t ? 'live' : 'stalled'} armed=${session.armed.recording}`)
}

// Let every session capture real frames before any of them stops.
await new Promise((r) => setTimeout(r, RECORD_MS))

const stopped = await Promise.all(sessions.map((s) => s.stop()))
console.log('upload logs:', stopped.map((ok, i) => `s${i}=${ok ? 'ok' : 'missing'}`).join(' '))
for (const s of sessions) {
  if (stopped[s.index]) continue
  console.log(`--- session ${s.index} never uploaded; video ${s.frames.w}px, armed=${s.armed.recording}`)
  console.log(s.logs.slice(-6).join('\n'))
}

const list = await fetch(`${API}/v1/me/recordings?limit=${SESSIONS}`, {
  headers: { Authorization: `Bearer ${TOKEN}`, 'X-API-Key': API_KEY },
})
const body = await list.json()
const rows = (body.items || []).slice(0, SESSIONS)
for (const row of rows) {
  console.log(`row ${row.id}: status=${row.status} size=${row.size_bytes} thumb=${row.thumbnail_url || '-'} artifact=${row.artifact_url || '-'}`)
}

const ok = stopped.every(Boolean) && rows.length === SESSIONS &&
  rows.every((r) => r.status === 'completed' && r.size_bytes > 0 && r.artifact_url && r.thumbnail_url)
console.log(ok ? 'RESULT=PASS' : 'RESULT=FAIL')
await browser.close()
process.exit(ok ? 0 : 1)
