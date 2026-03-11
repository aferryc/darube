/// <reference types="vitest" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: './', // Necessary for electron built files
  server: {
    port: 5173,
    strictPort: true,
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.js'],
    include: ['src/**/*.{test,spec}.{js,jsx}', 'scripts/**/*.test.js'],
    coverage: {
      provider: 'v8',
      // Pragmatic coverage gate: only count unit-testable modules.
      include: [
        'src/utils/**/*.js',
        'src/components/ConnectionModal.jsx',
        'src/components/ScriptHelpModal.jsx',
        'src/components/ScriptAutocomplete.jsx',
      ],
      exclude: [
        '**/*.test.*',
        '**/*.spec.*',
        'src/test/**',
      ],
      thresholds: {
        statements: 95,
        lines: 95,
        functions: 95,
        // Branch coverage for React-heavy files is noisier (many render conditionals).
        branches: 85,
      },
    },
  },
})
