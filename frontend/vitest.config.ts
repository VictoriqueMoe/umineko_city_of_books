import { defineConfig } from "vitest/config";

export default defineConfig({
    test: {
        environment: "jsdom",
        globals: false,
        setupFiles: ["./src/test-utils/setup.ts"],
        include: ["src/**/*.{test,spec}.{ts,tsx}"],
        clearMocks: true,
        unstubEnvs: true,
        unstubGlobals: true,
        env: {
            VITE_API_BASE: "",
        },
        coverage: {
            provider: "v8",
            reporter: ["text-summary", "html"],
            include: ["src/**/*.{ts,tsx}"],
            exclude: [
                "src/**/*.test.{ts,tsx}",
                "src/test-utils/**",
                "src/main.tsx",
                "src/vite-env.d.ts",
                "src/types/**",
                "src/api/endpoints.ts",
            ],
        },
    },
});
