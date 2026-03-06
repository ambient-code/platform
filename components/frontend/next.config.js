/** @type {import('next').NextConfig} */
const isCoverage = process.env.CYPRESS_COVERAGE === 'true'

const nextConfig = {
  output: 'standalone',
  turbopack: {
    root: __dirname,
  },
  experimental: {
    instrumentationHook: true,
    // Force all static pages into a single worker to prevent QEMU SIGILL
    // crashes during cross-architecture Docker builds (arm64 emulation).
    staticGenerationMinPagesPerWorker: 100,
  },
  webpack(config) {
    if (isCoverage) {
      config.module.rules.push({
        test: /\.(js|ts|jsx|tsx)$/,
        exclude: /node_modules/,
        enforce: 'post',
        use: {
          loader: 'babel-loader',
          options: {
            presets: ['next/babel'],
            plugins: ['istanbul'],
          },
        },
      })
    }
    return config
  },
}

module.exports = nextConfig
