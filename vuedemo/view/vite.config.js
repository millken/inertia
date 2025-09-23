import { fileURLToPath, URL } from 'node:url'
import fs from 'node:fs'
import path from 'node:path'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

function collectVueEntries(baseDirs) {
  const entries = {}
  for (const base of baseDirs) {
    const dir = path.resolve(base)
    if (!fs.existsSync(dir)) continue
    const stack = [dir]
    while (stack.length) {
      const cur = stack.pop()
      const files = fs.readdirSync(cur, { withFileTypes: true })
      for (const f of files) {
        const full = path.join(cur, f.name)
        if (f.isDirectory()) {
          stack.push(full)
          continue
        }
        if (f.isFile() && (f.name === 'main.js' || full.endsWith('.vue'))) {
          // use filename without extension as entry name (unique per project assumption)
          const name = path.basename(f.name, path.extname(f.name))
          // ensure unique key by prefixing with directory if collision
          let key = name
          if (entries[key]) {
            const rel = path.relative(dir, full).replace(/\\/g, '/')
            key = rel.replace(/\.(vue|js)$/, '')
          }
          entries[key] = full
        }
      }
    }
  }
  return entries
}

// create JS wrapper entry files that re-export the .vue component (no mount)
function createWrapperEntries(vueEntries) {
  const tempDir = path.resolve('.vite-entries')
  fs.mkdirSync(tempDir, { recursive: true })
  const wrappers = {}
  for (const [key, full] of Object.entries(vueEntries)) {
    if (full.endsWith('main.js')) {
      wrappers[key] = full
      continue
    }
    const wrapperPath = path.join(tempDir, `${key}.js`)
    // Rollup/Vite accepts absolute paths for import; ensure POSIX separators
    const importPath = full.replace(/\\/g, '/')
    const content = `import Comp from '${importPath}'\nexport const ${key} = Comp\nexport default Comp\n`
    fs.writeFileSync(wrapperPath, content, 'utf8')
    wrappers[key] = wrapperPath
  }
  return wrappers
}

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
  const vueEntries = collectVueEntries(['./src'])

  // generate temporary wrapper JS entries so output bundles are ES modules with exports only
  const wrapperEntries = Object.keys(vueEntries).length ? createWrapperEntries(vueEntries) : {}

  // compute output name map based on path relative to src to keep directory structure
  const outputNameMap = {}
  const srcDir = path.resolve('./src')
  for (const key of Object.keys(vueEntries)) {
    const full = vueEntries[key]
    let rel = path.relative(srcDir, full).replace(/\\/g, '/')
    rel = rel.replace(/\.(vue|js)$/, '')
    outputNameMap[key] = rel
  }

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    },
    build: {
      rollupOptions: {
        preserveEntrySignatures: 'strict',
        input: Object.keys(wrapperEntries).length ? wrapperEntries : undefined,
        output: {
          exports: 'named',
          entryFileNames: chunkInfo => {
            const name = chunkInfo.name
            if (outputNameMap[name]) {
              return `${outputNameMap[name]}.js`
            }
            return `${name}.js`
          },
          chunkFileNames: 'chunks/[name]-[hash].js',
          assetFileNames: 'assets/[name]-[hash][extname]'
        }
      }
    }
  }
})
