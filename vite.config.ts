/* eslint-disable import/no-extraneous-dependencies */
/// <reference types="vitest" />
/// <reference types="vite/client" />

import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import { react } from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [tailwindcss(), react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.ts'],
  },
  server: {
    watch: {
      usePolling: true,
    }
  }
});
