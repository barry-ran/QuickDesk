import assert from 'node:assert/strict';
import { before, after, test } from 'node:test';
import { createServer } from 'vite';
import { chromium } from 'playwright';
import { fileURLToPath } from 'node:url';
// Real Vue app + Chromium layout; API fixtures, not live-server/device QA.
let server;
let browser;
let base;
before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../../', import.meta.url)),
    server: { host: '127.0.0.1', port: 0 },
  });
  await server.listen();
  base = server.resolvedUrls.local[0];
  browser = await chromium.launch({
    headless: true,
    ...(process.env.QD_BROWSER_EXECUTABLE
      ? { executablePath: process.env.QD_BROWSER_EXECUTABLE }
      : {}),
  });
});
after(async () => {
  await browser?.close();
  await server?.close();
});
for (const width of [320, 390, 600, 601, 1280]) {
  test(`account entry and navigation reachable at ${width}px`, async () => {
    const context = await browser.newContext({ viewport: { width, height: 844 } });
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.addInitScript(() => {
      localStorage.setItem('quickdesk_signaling_url', 'ws://account.invalid');
      localStorage.setItem('quickdesk_lang', 'en-US');
    });
    await page.route('**/v1/**', route => route.fulfill({ contentType: 'application/json', body: JSON.stringify(
      route.request().url().endsWith('/features') ? { sms_enabled: false, register_enabled: false } : { items: [] }
    ) }));
    try {
      await page.goto(base);
      const entry = page.locator('button.nav-header');
      await entry.waitFor({ state: 'visible' });
      const box = await entry.boundingBox();
      assert.ok(box.width > 0 && box.x >= 0 && box.x + box.width <= width);
      await entry.click();
      await page.locator('.dialog').waitFor({ state: 'visible' });
      await page.locator('.dialog-overlay').click({ position: { x: 2, y: 2 } });
      await page.locator('.dialog').waitFor({ state: 'hidden' });
      await entry.focus();
      await page.keyboard.press('Enter');
      await page.locator('.dialog').waitFor({ state: 'visible' });
      await page.locator('.dialog-overlay').click({ position: { x: 2, y: 2 } });
      await page.evaluate(async () => {
        const { authState } = await import('/src/store/auth.js');
        authState.userInfo = { username: 'Mobile User' };
        authState.isLoggedIn = true;
      });
      await entry.click();
      const menu = page.locator('.user-menu');
      await menu.waitFor({ state: 'visible' });
      const menuBox = await menu.boundingBox();
      assert.ok(menuBox.x >= 0 && menuBox.x + menuBox.width <= width);
      assert.ok(menuBox.y >= 0 && menuBox.y + menuBox.height <= 844);
      await menu.locator('a[href="#/account"]').click();
      await page.waitForURL('**/#/account');
      await page.locator('.nav-items a[href="#/remote"]').click();
      await page.waitForURL('**/#/remote');
      assert.deepEqual(errors, []);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    } finally {
      await context.close();
    }
  });
}
