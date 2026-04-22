import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

function copyAssetsPlugin() {
  return {
    name: 'copy-assets',
    closeBundle() {
      const source = path.resolve(__dirname, 'src/assets/logo.png')
      const target = path.resolve(__dirname, 'dist/logo.png')
      if (fs.existsSync(source)) {
        fs.copyFileSync(source, target)
      }
    }
  }
}

function copyChatPlugin() {
  return {
    name: 'copy-chat-html',
    closeBundle() {
      const source = path.resolve(__dirname, 'chat.html')
      const target = path.resolve(__dirname, 'dist/chat.html')
      if (fs.existsSync(source)) {
        fs.copyFileSync(source, target)
      }
    }
  }
}

function copyWidgetDemoPlugin() {
  return {
    name: 'copy-widget-demo-html',
    closeBundle() {
      const source = path.resolve(__dirname, 'widget.html')
      const target = path.resolve(__dirname, 'dist/widget.html')
      if (fs.existsSync(source)) {
        fs.copyFileSync(source, target)
      }
    }
  }
}

function copyI18nPlugin() {
  return {
    name: 'copy-i18n-js',
    closeBundle() {
      const source = path.resolve(__dirname, 'public/i18n.js')
      const target = path.resolve(__dirname, 'dist/i18n.js')
      if (fs.existsSync(source)) {
        fs.copyFileSync(source, target)
      }
    }
  }
}

function copyLocaleJsonPlugin() {
  return {
    name: 'copy-locale-json',
    closeBundle() {
      const sourceDir = path.resolve(__dirname, 'public/locales')
      const targetDir = path.resolve(__dirname, 'dist/locales')
      if (fs.existsSync(sourceDir)) {
        fs.mkdirSync(targetDir, { recursive: true })
        fs.cpSync(sourceDir, targetDir, { recursive: true })
      }
    }
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    copyAssetsPlugin(),
    copyChatPlugin(),
    copyWidgetDemoPlugin(),
    copyI18nPlugin(),
    copyLocaleJsonPlugin()
  ],
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
    'process.env': '{}',
  },
  publicDir: false,
  build: {
    outDir: 'dist',
    cssCodeSplit: false,
    rollupOptions: {
      input: {
        widget: 'src/script/widget.js',
        chat: 'src/script/chat.js'
      },
      output: {
        entryFileNames: chunk => {
            if (chunk.name === 'widget') {
              return 'widget.min.js'
            } else if (chunk.name === 'chat') {
              return 'chat.min.js'
            }
            return '[name].[hash].js'
          },
        chunkFileNames: 'vendor.js',
        assetFileNames: assetInfo => {
          if (assetInfo.name === 'style.css') {
            return 'style.css'
          }
          return '[name].[ext]'
        },
        globals: {}
      }
    }
  },
})
