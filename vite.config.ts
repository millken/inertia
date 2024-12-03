import {glob} from 'glob';
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import svelteViewsPlugin from './vite-plugin-svelte-views';

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte(),
    // svelteViewsPlugin({
    //   viewsDir: './view' // 可选，默认为 'src/views'
    // }),
],
  build: {
    // sourcemap: 'inline',
    input: "./src/main.ts",
    manifest: true, // Generate manifest.json file
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[name].js',
        chunkFileNames: 'assets/[name].js',
        assetFileNames: 'assets/[name].[ext]',
        manualChunks: undefined, // Disable automatic chunk splitting
      },
    },
  },
  optimizeDeps: {
    include: ['svelte'],
  },
  resolve: {
    alias: {
      '@': new URL('.', import.meta.url).pathname,
    },
  },
})
