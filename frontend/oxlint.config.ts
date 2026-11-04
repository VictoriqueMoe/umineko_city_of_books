import { defineConfig } from "oxlint";
import { layerRules } from "./oxlint.layers.ts";

export default defineConfig({
    ignorePatterns: ["dist/**", "dist-app/**", "android/**", "ios/**", "scripts/**", "node_modules/**"],
    plugins: ["eslint", "typescript", "react", "vitest", "unicorn"],
    jsPlugins: ["./oxlint-plugin-layers.mjs", "./oxlint-plugin-wrappers.mjs"],
    rules: {
        "eslint/curly": ["error", "all"],
        "eslint/no-empty": ["error", { allowEmptyCatch: true }],
        "typescript/no-floating-promises": "off",
        "vitest/require-mock-type-parameters": "off",
        "vitest/expect-expect": ["error", { assertFunctionNames: ["expect*", "assert*"] }],
        "vitest/valid-title": ["error", { ignoreTypeOfTestName: true }],
        "eslint/no-unused-vars": [
            "error",
            {
                args: "all",
                argsIgnorePattern: "^_",
                caughtErrors: "all",
                caughtErrorsIgnorePattern: "^_",
                destructuredArrayIgnorePattern: "^_",
                varsIgnorePattern: "^_",
                ignoreRestSiblings: true,
            },
        ],
    },
    overrides: [
        {
            files: ["**/*.test.{ts,tsx}", "src/test-utils/**"],
            rules: { "typescript/unbound-method": "off" },
        },
        ...layerRules,
    ],
});
