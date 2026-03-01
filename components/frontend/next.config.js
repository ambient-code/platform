/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  ...(process.env.NODE_ENV !== 'production' && {
    turbopack: {
      root: `${__dirname}/../..`,
    },
  }),
  experimental: {
    instrumentationHook: true,
  }
}

module.exports = nextConfig
