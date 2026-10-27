# WebClient File Manager E2E

`file-manager.e2e.mjs` loads the real `RemoteFileManager` and
`FileManagerPanel` modules in Chrome against an in-page fake host. It covers
directory navigation, create, rename, truncated text preview, download, chunked
upload, and trash.

```bash
cd WebClient
npm install
npm run test:e2e:file-manager
```

Set `QD_BROWSER_CHANNEL` to use a channel other than the locally installed
Google Chrome.

## Mobile controls

`mobile-controls.e2e.mjs` exercises the real touch handler using Chromium's native
CDP touch input in mobile-emulated portrait and landscape viewports. It checks
click/drag/cancel, locked scroll versus pinch, scrolling at zoom, and orientation
cursor mapping against an in-page recording host (not a live remote computer).

```bash
cd WebClient
npm install
npx playwright install chromium
npm run test:e2e:mobile
```

The default uses Playwright's bundled Chromium. For a system browser, set
`QD_BROWSER_EXECUTABLE=/usr/bin/chromium` (also accepts
`PLAYWRIGHT_CHROMIUM_EXECUTABLE`), or use `QD_BROWSER_CHANNEL=chrome`.
These tests do not substitute for physical Android/iOS, native IME, or end-to-end
remote host interoperability checks. A sandbox must permit Chromium's local IPC
sockets for browser launch.

The same suite also loads the actual `remote.html` and `remote-main.js` at 320px,
using installed local crypto modules instead of CDN imports. After the normal
missing-credentials path it attaches a recording data-channel boundary and calls
the app's real setup methods: no toolbar bindings are duplicated in the fixture.
It checks horizontally reachable 48px actions, help disclosure, drag pressed
state, keyboard command row and a synthetic committed-composition sequence.
