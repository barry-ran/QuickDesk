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
