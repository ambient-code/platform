import { defineConfig } from 'cypress'
import * as dotenv from 'dotenv'
import * as path from 'path'
import * as fs from 'fs'

// Load env files in precedence order (first match wins per variable)
// .env.local > .env > .env.test
const envFiles = ['.env.local', '.env', '.env.test'].map(f => path.resolve(__dirname, f))

for (const envFile of envFiles) {
  if (fs.existsSync(envFile)) {
    dotenv.config({ path: envFile })
  }
}

export default defineConfig({
  e2e: {
    // Use CYPRESS_BASE_URL env var, fallback to default
    baseUrl: process.env.CYPRESS_BASE_URL || 'http://vteam.local',
    video: true,  // Enable video recording
    screenshotOnRunFailure: true,
    defaultCommandTimeout: 10000,
    requestTimeout: 10000,
    responseTimeout: 10000,
    viewportWidth: 1280,
    viewportHeight: 720,
    setupNodeEvents(on, config) {
      // Pass environment variables to Cypress tests
      // CYPRESS_* env vars are automatically exposed, but we explicitly set these too
      config.env.ANTHROPIC_API_KEY = process.env.CYPRESS_ANTHROPIC_API_KEY || process.env.ANTHROPIC_API_KEY || ''
      config.env.TEST_TOKEN = process.env.CYPRESS_TEST_TOKEN || process.env.TEST_TOKEN || config.env.TEST_TOKEN || ''

      return config
    },
  },
})

