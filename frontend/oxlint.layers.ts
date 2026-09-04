import type { OxlintOverride } from "oxlint";

const RENDER = ["src/components/**/*.{ts,tsx}", "src/pages/**/*.{ts,tsx}", "src/App.tsx", "src/App.test.tsx"];
const ORCHESTRATION = ["src/hooks/**/*.{ts,tsx}", "src/context/**/*.{ts,tsx}", "src/games/*/hooks/**/*.ts"];
const DATA_HOOKS = ["src/hooks/queries/**/*.ts", "src/hooks/mutations/**/*.ts"];
const PURE = [
    "src/domain/**/*.{ts,tsx}",
    "src/utils/**/*.{ts,tsx}",
    "src/types/**/*.{ts,tsx}",
    "src/games/**/*.{ts,tsx}",
];
const PURE_OVERLAPPING_ORCHESTRATION = ["src/games/*/hooks/**"];
const API = ["src/api/**/*.{ts,tsx}"];
const PLATFORM = ["src/platform/**/*.{ts,tsx}"];
const ADAPTERS = [...API, ...PLATFORM];
const COMPOSITION_ROOT = ["src/main.tsx"];
const EVERY_SOURCE_FILE = ["src/**/*.{ts,tsx}"];
const RENDER_TESTS = ["src/components/**/*.test.{ts,tsx}", "src/pages/**/*.test.{ts,tsx}", "src/App.test.tsx"];
const PURE_TESTS = ["src/domain/**/*.test.{ts,tsx}", "src/utils/**/*.test.{ts,tsx}"];
const TRANSPORT_TESTS = ["src/api/**/*.test.{ts,tsx}"];
const CACHE_KEY_ASSERTING_TESTS = ["src/api/**/*.test.{ts,tsx}", "src/hooks/**/*.test.{ts,tsx}"];
const RENDER_TESTS_STILL_MOCKING_TRANSPORT = [
    "src/components/chat/WatchParty/WatchPartyModal.test.tsx",
    "src/components/games/chat/GameChat.test.tsx",
    "src/components/live/GoLivePanel.test.tsx",
    "src/pages/live/LiveWatchPage.test.tsx",
    "src/pages/live/StreamChatPanel.test.tsx",
    "src/pages/profile/StreamOverlaySection.test.tsx",
    "src/pages/rooms/RoomsListPage.test.tsx",
];

const apiAny = ["**/api/*", "**/api/**"];
const apiEndpoints = ["**/api/endpoints", "**/api/endpoints.ts", "**/api/endpoints/**"];
const apiTransport = ["**/api/client", "**/api/client.ts", "**/api/queryClient", "**/api/queryClient.ts"];
const reactRuntime = ["react", "react-dom", "@tanstack/react-query"];
const reactQuery = ["@tanstack/react-query"];
const upward = ["**/components/**", "**/pages/**", "**/hooks/**", "**/context/**"];
const platformBeyondPredicates = [
    "**/platform/*",
    "**/platform/**",
    "!**/platform/capabilities",
    "!**/platform/capabilities.ts",
];

const renderApiMessage =
    "The render layer may not import src/api. Read the data from a hook in src/hooks and destructure it.";
const renderQueryMessage =
    "The render layer may not import react-query. Wrap the query in a data hook under src/hooks/queries or src/hooks/mutations.";
const transportMessage =
    "Only src/api, src/hooks/queries, src/hooks/mutations and src/main.tsx may import the transport modules api/client, api/queryClient and api/endpoints.";
const pureApiMessage = "The pure layer may not import src/api. A pure module takes values and returns values.";
const pureReactMessage =
    "The pure layer may not import react, react-dom or react-query. A module that needs them belongs in src/hooks.";
const adapterUpwardMessage =
    "The adapter layers may not import upward from components, pages, hooks or context. Inject what the adapter needs.";
const platformApiMessage =
    "src/platform may not import src/api. The server call is passed in as a parameter by the caller.";
const apiPlatformMessage =
    "src/api may import platform/capabilities only, which answers a device question with no effect. Anything else from src/platform is injected.";

export const layerRules: OxlintOverride[] = [
    {
        files: DATA_HOOKS,
        rules: {
            "no-restricted-imports": "off",
        },
    },
    {
        files: ORCHESTRATION,
        excludeFiles: DATA_HOOKS,
        rules: {
            "no-restricted-imports": [
                "error",
                {
                    patterns: [{ group: [...apiTransport, ...apiEndpoints], message: transportMessage }],
                },
            ],
        },
    },
    {
        files: RENDER,
        rules: {
            "no-restricted-imports": [
                "error",
                {
                    patterns: [
                        { group: apiAny, message: renderApiMessage, allowTypeImports: false },
                        { group: reactQuery, message: renderQueryMessage, allowTypeImports: false },
                    ],
                },
            ],
        },
    },
    {
        files: PURE,
        excludeFiles: PURE_OVERLAPPING_ORCHESTRATION,
        rules: {
            "no-restricted-imports": [
                "error",
                {
                    patterns: [
                        { group: apiAny, message: pureApiMessage },
                        { group: reactRuntime, message: pureReactMessage },
                    ],
                },
            ],
        },
    },
    {
        files: API,
        rules: {
            "no-restricted-imports": [
                "error",
                {
                    patterns: [
                        { group: upward, message: adapterUpwardMessage },
                        { group: platformBeyondPredicates, message: apiPlatformMessage },
                    ],
                },
            ],
        },
    },
    {
        files: PLATFORM,
        rules: {
            "no-restricted-imports": [
                "error",
                {
                    patterns: [
                        { group: upward, message: adapterUpwardMessage },
                        { group: apiAny, message: platformApiMessage },
                    ],
                },
            ],
        },
    },
    {
        files: EVERY_SOURCE_FILE,
        excludeFiles: [...RENDER, ...DATA_HOOKS, ...ORCHESTRATION, ...PURE, ...ADAPTERS, ...COMPOSITION_ROOT],
        rules: {
            "no-restricted-imports": [
                "error",
                {
                    patterns: [{ group: [...apiTransport, ...apiEndpoints], message: transportMessage }],
                },
            ],
        },
    },
    {
        files: EVERY_SOURCE_FILE,
        rules: {
            "layers/no-raw-query-key": "error",
            "layers/no-dynamic-api-import": "error",
            "layers/no-stub-global-fetch": "error",
        },
    },
    {
        files: RENDER_TESTS,
        rules: {
            "layers/no-transport-mock-in-render-test": "error",
        },
    },
    {
        files: PURE_TESTS,
        rules: {
            "layers/no-render-import-in-pure-test": "error",
        },
    },
    {
        files: TRANSPORT_TESTS,
        rules: {
            "layers/no-stub-global-fetch": "off",
        },
    },
    {
        files: CACHE_KEY_ASSERTING_TESTS,
        rules: {
            "layers/no-raw-query-key": "off",
        },
    },
    {
        files: RENDER_TESTS_STILL_MOCKING_TRANSPORT,
        rules: {
            "layers/no-transport-mock-in-render-test": "off",
        },
    },
    {
        files: ["src/api/queryKeys.ts"],
        rules: {
            "layers/no-raw-query-key": "off",
        },
    },
];
