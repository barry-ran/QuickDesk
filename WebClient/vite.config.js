import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// Only the Vue SPA goes through Vite/Rollup. remote.html + js/** are
// shipped as static assets (they still load @noble/* from an importmap
// in <head>). scripts/build_webclient_*.bat|sh copies them into dist/
// after `npm run build`.
export default defineConfig({
  plugins: [vue()],
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
})
