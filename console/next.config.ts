import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:1997/api/:path*",
      },
      {
        source: "/access/:path*",
        destination: "http://localhost:1997/access/:path*",
      },
      {
        source: "/healthz",
        destination: "http://localhost:1997/healthz",
      },
    ];
  },
};

export default nextConfig;
