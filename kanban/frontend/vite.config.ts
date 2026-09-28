import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/max-kanban/',
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
  },
  server: {
    port: 3002,
    proxy: {
      '/max-kanban/api': {
        target: 'http://127.0.0.1:9300',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/max-kanban/, ''),
      },
      '/max-kanban/ws': {
        target: 'ws://127.0.0.1:9300',
        ws: true,
        rewrite: (path) => path.replace(/^\/max-kanban/, ''),
      },
      '/max-kanban/webhook': {
        target: 'http://127.0.0.1:9300',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/max-kanban/, ''),
      },
    },
  },
})
