import { defineConfig } from "vitest/config";

export default defineConfig({
    test: {
        environment: "jsdom",
        pool: "vmThreads",
        globals: false,
        setupFiles: ["./src/test-utils/setup.ts"],
        include: ["src/**/*.{test,spec}.{ts,tsx}"],
        unstubEnvs: true,
        unstubGlobals: true,
        taskTitleValueFormatTruncate: 1000,
        env: {
            VITE_API_BASE: "",
        },
        coverage: {
            provider: "v8",
            reporter: ["text-summary", "html"],
            include: ["src/**/*.{ts,tsx}"],
            exclude: [
                "src/test-utils/**",
                "src/api/endpoints/testHarness.ts",
                "src/vite-env.d.ts",
                "src/types/**",
                "src/**/*.fixture.ts",
            ],
        },
    },
});
