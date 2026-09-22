import { PHASE_DEVELOPMENT_SERVER } from "next/constants.js";

const backendBaseUrl =
  process.env.NEXT_API_BASE_URL?.trim() ||
  process.env.NEXT_PUBLIC_API_BASE_URL?.trim() ||
  "http://127.0.0.1:8083";
const productionBasePath = "";

/** @type {(phase: string) => import('next').NextConfig} */
export default function nextConfig(phase) {
  const config = {
    output: "export",
    basePath: productionBasePath,
    assetPrefix: `${productionBasePath}/`,
    trailingSlash: false,
    devIndicators: false,
    reactStrictMode: false,
  };

  if (phase !== PHASE_DEVELOPMENT_SERVER) {
    return config;
  }

  return {
    ...config,
    async rewrites() {
      return [
        {
          source: "/support/docs/:slug+",
          destination: "/support/docs",
        },
        {
          source: "/support/community/posts/:id(\\d+)",
          destination: "/support/community/posts/detail?id=:id",
        },
        {
          source: "/support/community/categories/:slug",
          destination: "/support/community/categories?category=:slug",
        },
        {
          source: "/api/:path*",
          destination: `${backendBaseUrl}/api/:path*`,
        },
        {
          source: "/storage/:path*",
          destination: `${backendBaseUrl}/storage/:path*`,
        },
      ];
    },
  };
}
