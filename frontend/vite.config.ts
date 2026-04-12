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
        rolldownOptions: {
            output: {
                codeSplitting: {
                    groups: [
                        {
                            name: "vendor-react",
                            test: /[\\/]node_modules[\\/](react|react-dom|react-router|scheduler)[\\/]/,
                        },
                        {
                            name: "vendor-tiptap",
                            test: /@tiptap|prosemirror/,
                        },
                        {
                            name: "vendor-markdown",
                            test: /marked|dompurify/,
                        },
                        {
                            name: "vendor-turnstile",
                            test: /@marsidev|react-turnstile/,
                        },
                    ],
                },
            },
        },
    },
});
