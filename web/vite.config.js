import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { svelteTesting } from '@testing-library/svelte/vite'

// The UI is served by the Go binary under /admin/
export default defineConfig({
  base: '/admin/',
  plugins: [svelte(), svelteTesting()],
  build: {
    outDir: 'build',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      // Forward API calls to a locally running corto server during development
      '/api': 'http://127.0.0.1:3000',
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.js'],
  },
})
