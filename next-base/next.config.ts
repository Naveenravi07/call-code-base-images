import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  experimental: {
    allowedDevOrigins: ["call-code.local"],
  },
};

export default nextConfig;
