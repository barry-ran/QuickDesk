import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { createReadStream } from 'node:fs'
import { stat } from 'node:fs/promises'
import http from 'node:http'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'

const WEB_ROOT = path.resolve(fileURLToPath(new URL('../../', import.meta.url)))
const MIME_TYPES = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.ico': 'image/x-icon',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
}

let browser

before(async () => {
  browser = await chromium.launch({
    channel: process.env.QD_BROWSER_CHANNEL || 'chrome',
    headless: true,
  })
})

after(async () => {
  await browser?.close()
})

test('remote file manager completes the browser workflow against a fake host', async () => {
  const server = await startWebServer()
  const context = await browser.newContext({
    acceptDownloads: true,
    viewport: { width: 1280, height: 800 },
  })
  await context.addInitScript(() => {
    localStorage.setItem('quickdesk_lang', 'en-US')
    Object.defineProperty(window, 'showSaveFilePicker', {
      configurable: true,
      value: undefined,
    })
  })
  const page = await context.newPage()
  const pageErrors = []
  page.on('pageerror', (error) => pageErrors.push(error.message))

  try {
    const base = `http://127.0.0.1:${server.address().port}`
    await page.goto(`${base}/file-manager-fixture.html`, { waitUntil: 'domcontentloaded' })
    await page.waitForFunction(() => window.__fileManagerReady === true)

    const root = page.locator('.file-manager-root')
    await assertVisible(root)
    assert.equal(await page.locator('.file-manager-path').inputValue(), '/home/test')
    await assertEntryNames(page, ['docs', 'blob.bin', 'note.txt'])

    await entryRow(page, 'docs').locator('.file-manager-entry-name').click()
    await page.waitForFunction(
      () => window.__fileManagerState.currentPath === '/home/test/docs',
    )
    assert.equal(await page.locator('.file-manager-path').inputValue(), '/home/test/docs')
    assert.equal(await page.locator('.file-manager-entry').count(), 0)

    await page.locator('.file-manager-crumb', { hasText: 'test' }).click()
    await page.waitForFunction(
      () => window.__fileManagerState.currentPath === '/home/test',
    )

    await page.locator('.file-manager-new-folder').click()
    await page.locator('.file-manager-dialog-input').fill('new-folder')
    await page.locator('.file-manager-dialog-card').evaluate((form) => form.requestSubmit())
    await page.waitForFunction(() => window.__fileManagerState.has('new-folder'))

    await entryRow(page, 'note.txt').locator('.file-manager-action').first().click()
    await page.locator('.file-manager-dialog-input').fill('renamed.txt')
    await page.locator('.file-manager-dialog-card').evaluate((form) => form.requestSubmit())
    await page.waitForFunction(() => window.__fileManagerState.has('renamed.txt'))
    assert.equal(await page.getByText('note.txt', { exact: true }).count(), 0)

    await entryRow(page, 'renamed.txt').locator('.file-manager-entry-name').click()
    await page.waitForFunction(() => {
      const preview = document.querySelector('.file-manager-preview')
      const status = document.querySelector('.file-manager-status')
      return preview?.hidden === false || status?.classList.contains('error')
    }, null, { timeout: 5000 })
    const previewState = await page.evaluate(() => ({
      text: document.querySelector('.file-manager-preview-text')?.textContent || '',
      status: document.querySelector('.file-manager-status')?.textContent || '',
      path: window.__fileManagerState.currentPath,
    }))
    assert.equal(previewState.text.startsWith('hell'), true, JSON.stringify(previewState))
    assert.match(previewState.text, /Only the beginning of the file is shown/)

    const downloadPromise = page.waitForEvent('download')
    await entryRow(page, 'blob.bin').locator('.file-manager-action').nth(1).click()
    const download = await downloadPromise
    assert.equal(download.suggestedFilename(), 'blob.bin')

    await page.locator('input[type="file"]').setInputFiles({
      name: 'upload.bin',
      mimeType: 'application/octet-stream',
      buffer: Buffer.from('uploaded-data'),
    })
    await page.waitForFunction(() => window.__fileManagerState.has('upload.bin'))
    assert.equal(
      await page.evaluate(() => window.__fakeHost.fileText('/home/test/upload.bin')),
      'uploaded-data',
    )

    page.once('dialog', (dialog) => dialog.accept())
    await entryRow(page, 'upload.bin').locator('.file-manager-action').last().click()
    await page.waitForFunction(() => !window.__fileManagerState.has('upload.bin'))
    assert.deepEqual(pageErrors, [])
  } finally {
    await context.close()
    await closeServer(server)
  }
})

async function assertVisible(locator) {
  assert.equal(await locator.isVisible(), true)
}

async function assertEntryNames(page, expected) {
  const names = await page.locator('.file-manager-entry-text').allTextContents()
  assert.deepEqual(names, expected)
}

function entryRow(page, name) {
  return page.locator('.file-manager-entry', {
    has: page.locator('.file-manager-entry-text', { hasText: name }),
  })
}

async function startWebServer() {
  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url, 'http://127.0.0.1')
    if (url.pathname === '/file-manager-fixture.html') {
      res.writeHead(200, {
        'Content-Type': MIME_TYPES['.html'],
        'Cache-Control': 'no-store',
      })
      res.end(FIXTURE_HTML)
      return
    }

    const filePath = path.resolve(WEB_ROOT, `.${decodeURIComponent(url.pathname)}`)
    const relative = path.relative(WEB_ROOT, filePath)
    if (relative.startsWith('..') || path.isAbsolute(relative)) {
      res.writeHead(403)
      res.end('forbidden')
      return
    }
    try {
      const info = await stat(filePath)
      if (!info.isFile()) throw new Error('not a file')
      res.writeHead(200, {
        'Content-Type': MIME_TYPES[path.extname(filePath)] || 'application/octet-stream',
        'Cache-Control': 'no-store',
      })
      createReadStream(filePath).pipe(res)
    } catch {
      res.writeHead(404)
      res.end('not found')
    }
  })
  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', resolve)
  })
  return server
}

async function closeServer(server) {
  await new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()))
  })
}

const FIXTURE_HTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <style>
    body { margin: 0; min-height: 100vh; font-family: sans-serif; }
    .file-manager-root[hidden] { display: none; }
    .file-manager-root { position: fixed; inset: 0; }
    .file-manager-backdrop { position: absolute; inset: 0; background: rgba(0,0,0,.45); }
    .file-manager-panel { position: absolute; inset: 24px; background: #fff; overflow: auto; }
    .file-manager-entry, .file-manager-header, .file-manager-toolbar, .file-manager-pathbar {
      display: flex; gap: 8px; align-items: center;
    }
    .file-manager-entry-text { flex: 1; }
    .file-manager-dialog[hidden], .file-manager-preview[hidden] { display: none; }
    .file-manager-dialog { position: absolute; inset: 0; background: rgba(0,0,0,.3); }
    .file-manager-dialog-card { width: 320px; margin: 80px auto; padding: 20px; background: #fff; }
  </style>
</head>
<body>
  <div id="app"></div>
  <script type="module">
    import { RemoteFileManager } from '/js/files/remote-file-manager.js'
    import { FileManagerPanel } from '/js/files/file-manager-panel.js'

    const tree = new Map([
      ['/home/test', [
        { name: 'docs', type: 'directory', size: 0 },
        { name: 'blob.bin', type: 'file', size: 4 },
        { name: 'note.txt', type: 'file', size: 11 },
      ]],
      ['/home/test/docs', []],
    ])
    const files = new Map([
      ['/home/test/blob.bin', new Uint8Array([1, 2, 3, 4])],
      ['/home/test/note.txt', new TextEncoder().encode('hello world')],
    ])
    const join = (directory, name) => directory === '/' ? '/' + name : directory + '/' + name
    const parent = (value) => value.slice(0, value.lastIndexOf('/')) || '/'
    const basename = (value) => value.slice(value.lastIndexOf('/') + 1)
    const toBase64 = (bytes) => {
      let binary = ''
      for (const byte of bytes) binary += String.fromCharCode(byte)
      return btoa(binary)
    }
    const fromBase64 = (value) => {
      const binary = atob(value || '')
      return Uint8Array.from(binary, (char) => char.charCodeAt(0))
    }
    const entriesFor = (directory) => tree.get(directory) || []
    const findEntry = (value) => {
      const directory = parent(value)
      const name = basename(value)
      return entriesFor(directory).find((entry) => entry.name === name)
    }

    const bridge = {
      async request(tool, args) {
        if (tool === 'get_default_directory') return { path: '/home/test' }
        if (tool === 'list_directory') {
          return { path: args.path, entries: entriesFor(args.path).map((entry) => ({ ...entry })) }
        }
        if (tool === 'create_directory') {
          if (!tree.has(parent(args.path))) throw new Error('Parent directory is missing')
          entriesFor(parent(args.path)).push({ name: basename(args.path), type: 'directory', size: 0 })
          tree.set(args.path, tree.get(args.path) || [])
          return { path: args.path }
        }
        if (tool === 'move_file') {
          if (parent(args.source) !== parent(args.destination)) throw new Error('Cross-directory move is unsupported')
          const entry = findEntry(args.source)
          if (!entry) throw new Error('Source is missing')
          entry.name = basename(args.destination)
          if (files.has(args.source)) {
            files.set(args.destination, files.get(args.source))
            files.delete(args.source)
          }
          return { path: args.destination }
        }
        if (tool === 'trash_path') {
          const list = entriesFor(parent(args.path))
          const index = list.findIndex((entry) => entry.name === basename(args.path))
          if (index >= 0) list.splice(index, 1)
          tree.delete(args.path)
          files.delete(args.path)
          return { path: args.path }
        }
        if (tool === 'read_file_chunk') {
          const bytes = files.get(args.path) || new Uint8Array()
          const offset = Number(args.offset) || 0
          const length = Number(args.length) || 65536
          const chunk = bytes.slice(offset, offset + length)
          const next = offset + chunk.length
          return {
            data_base64: toBase64(chunk),
            next_offset: next,
            eof: next >= bytes.length,
            size: bytes.length,
          }
        }
        if (tool === 'write_file_chunk') {
          const incoming = fromBase64(args.data_base64)
          const existing = args.truncate ? new Uint8Array() : (files.get(args.path) || new Uint8Array())
          const offset = Number(args.offset) || 0
          const merged = new Uint8Array(Math.max(existing.length, offset + incoming.length))
          merged.set(existing)
          merged.set(incoming, offset)
          files.set(args.path, merged)
          const entry = findEntry(args.path)
          if (entry) entry.size = merged.length
          else entriesFor(parent(args.path)).push({
            name: basename(args.path), type: 'file', size: merged.length,
          })
          return { next_offset: merged.length }
        }
        throw new Error('Unsupported tool: ' + tool)
      },
    }

    const manager = new RemoteFileManager(bridge, { chunkBytes: 4 })
    const panel = new FileManagerPanel(document.getElementById('app'), manager)
    panel.refreshI18n()
    await panel.open()
    window.__fakeHost = {
      fileText(value) {
        return new TextDecoder().decode(files.get(value) || new Uint8Array())
      },
    }
    window.__fileManagerState = {
      get currentPath() { return manager.currentPath },
      has(name) { return manager.entries.some((entry) => entry.name === name) },
    }
    window.__fileManagerReady = true
  </script>
</body>
</html>`
