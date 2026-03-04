/** @type {import('next').NextConfig} */
const isCoverage = process.env.CYPRESS_COVERAGE === 'true'

const nextConfig = {
  output: 'standalone',
  turbopack: {
    root: __dirname,
  },
  experimental: {
    instrumentationHook: true,
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
