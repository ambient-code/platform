/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  turbopack: {
    root: `${__dirname}/../..`,
  },
  experimental: {
    instrumentationHook: true,
  }
}

module.exports = nextConfig
