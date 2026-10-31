import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readFile } from 'node:fs/promises';
import { parse, compileTemplate, compileStyle } from 'vue/compiler-sfc';

const filename = new URL('../src/App.vue', import.meta.url);
const source = await readFile(filename, 'utf8');
const { descriptor, errors } = parse(source);
// Use the compiler exported by the existing direct Vue dependency.
const styles = compileStyle({
  source: descriptor.styles.map(s => s.content).join('\n'),
  filename: filename.pathname,
  id: 'app',
});
const css = styles.rawResult.root;

test('shared account trigger remains a native button wired to the existing login flow', () => {
  assert.deepEqual(errors, []);
  assert.deepEqual(styles.errors, []);
  const template = compileTemplate({ source: descriptor.template.content, filename: filename.pathname, id: 'app' });
  assert.deepEqual(template.errors, []);
  assert.match(descriptor.template.content, /<button[^>]+class="nav-header"[^>]+@click="onUserAreaClick"/);
  assert.match(descriptor.template.content, /<LoginDialog v-if="showLogin"/);
  assert.match(descriptor.scriptSetup.content, /showLogin.value = true/);
});

test('narrow layout preserves account header and navigation instead of hiding the sidebar', () => {
  const media = css.nodes.find(n => n.type === 'atrule' && n.params === '(max-width: 600px)');
  assert.ok(media);
  const declarations = selector => Object.fromEntries(media.nodes.filter(n => n.selector === selector).flatMap(n => n.nodes.map(d => [d.prop, d.value])));
  for (const selector of ['.nav-sidebar', '.nav-header', '.nav-items']) {
    assert.notEqual(declarations(selector).display, 'none');
    assert.notEqual(declarations(selector).visibility, 'hidden');
  }
  assert.equal(declarations('.app-layout')['flex-direction'], 'column');
  assert.equal(declarations('.nav-sidebar').width, '100%');
  assert.equal(declarations('.nav-items')['overflow-x'], 'auto');
  assert.equal(declarations('.user-menu').left, '8px');
  assert.equal(declarations('.content-area')['min-height'], '0');
});
