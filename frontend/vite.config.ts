import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  base: '/app/',
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
    dedupe: ['vue'],
  },
  optimizeDeps: {
    include: ['reka-ui', '@danqing/dq-ui', '@danqing/dq-shell'],
  },
  server: {
    port: Number(process.env.DQ_FRONTEND_PORT || 5801),
    strictPort: true,
    proxy: {
      '/api': { target: `http://127.0.0.1:${process.env.DQ_BACKEND_PORT || 7801}`, changeOrigin: true },
    },
  },
  build: {
    outDir: '../out/frontend/dist',
    emptyOutDir: true,
  },
})
