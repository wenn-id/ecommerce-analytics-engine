/** @type {import('next').NextConfig} */
const nextConfig = {
  // Produce a self-contained server bundle (.next/standalone) so the
  // production Docker image does not need the full node_modules tree (#60).
  output: 'standalone',
};

export default nextConfig;
