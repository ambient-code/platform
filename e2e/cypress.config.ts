import { defineConfig } from 'cypress'
import * as dotenv from 'dotenv'
import * as path from 'path'
import * as fs from 'fs'

// Load .env.local first (takes precedence), then .env
const envLocalPath = path.resolve(__dirname, '.env.local')
const envPath = path.resolve(__dirname, '.env')

if (fs.existsSync(envLocalPath)) {
  dotenv.config({ path: envLocalPath })
}
if (fs.existsSync(envPath)) {
  dotenv.config({ path: envPath })
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
      config.env.ANTHROPIC_API_KEY = process.env.ANTHROPIC_API_KEY || ''
      
      return config
    },
  },
})

