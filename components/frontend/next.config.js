/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  turbopack: {
    root: __dirname, // Silence "inferred workspace root" warning in monorepo
  }
}

module.exports = nextConfig
