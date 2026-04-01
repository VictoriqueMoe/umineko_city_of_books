import {defineConfig} from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
    plugins: [react()],
    server: {
        port: 5173,
        proxy: {
            "/api/v1/ws": {
                target: "http://localhost:4323",
                changeOrigin: true,
                ws: true,
            },
            "/api": {
                target: "http://localhost:4323",
                changeOrigin: true,
            },
            "/uploads": {
                target: "http://localhost:4323",
                changeOrigin: true,
            },
            "/sitemap": {
                target: "http://localhost:4323",
                changeOrigin: true,
            },
        },
    },
    build: {
        outDir: "../static",
        emptyOutDir: true,
    },
});
