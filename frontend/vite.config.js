import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 7071,
    proxy: {
      '/api': {
        target: 'http://localhost:7070',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:7070',
        ws: true,
      }
    }
  },
  build: {
    target: 'es2020',
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['vue', 'vue-router'],
          utils: ['axios']
        }
      }
    }
  }
})