const path = require('path')

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  outputFileTracingRoot: path.resolve(__dirname, '../..'),
  transpilePackages: ['ambient-sdk'],
  experimental: {
    staticGenerationMinPagesPerWorker: 100,
  },
}

module.exports = nextConfig
