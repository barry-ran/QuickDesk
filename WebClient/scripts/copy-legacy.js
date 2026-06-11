const { cpSync, existsSync, mkdirSync } = require('fs');
const { resolve } = require('path');

const root = resolve(__dirname, '..');
const dist = resolve(root, 'dist');

if (!existsSync(dist)) {
  console.error('dist/ does not exist — run vite build first');
  process.exit(1);
}

cpSync(resolve(root, 'remote.html'), resolve(dist, 'remote.html'));
cpSync(resolve(root, 'js'), resolve(dist, 'js'), { recursive: true });

console.log('Copied remote.html + js/ to dist/');
