import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";

declare const process: { env: Record<string, string | undefined> };

function devBaseURLPlaceholder(): Plugin {
    return {
        name: "dev-base-url-placeholder",
        apply: "serve",
        transformIndexHtml: {
            order: "pre",
            handler(html) {
                return html.replaceAll("__BASE_URL__", "http://localhost:5173");
            },
        },
    };
}

export default defineConfig({
    define: {
        __APP_VERSION__: JSON.stringify(process.env.VITE_APP_VERSION ?? "dev"),
    },
    plugins: [react(), devBaseURLPlaceholder()],
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
                            name: "vendor-highlight",
                            test: /[\\/]node_modules[\\/]highlight\.js[\\/]/,
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
