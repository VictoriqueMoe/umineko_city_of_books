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
const MAY_STUB_GLOBAL_FETCH = [
    "src/api/beacons/watchPartyLeave.test.ts",
    "src/api/client.test.ts",
    "src/api/endpoints/quote.test.ts",
    "src/api/ota.test.ts",
];
const CACHE_KEY_ASSERTING_TESTS = [
    "src/api/cache/patchUser.test.ts",
    "src/api/queryClient.test.ts",
    "src/hooks/mutations/admin.test.ts",
    "src/hooks/mutations/announcement.test.ts",
    "src/hooks/mutations/art.test.ts",
    "src/hooks/mutations/auth.test.ts",
    "src/hooks/mutations/gameRoom.test.ts",
    "src/hooks/mutations/report.test.ts",
    "src/hooks/mutations/secret.test.ts",
    "src/hooks/mutations/ship.test.ts",
    "src/hooks/mutations/theory.test.ts",
    "src/hooks/mutations/user.test.ts",
    "src/hooks/queries/art.test.ts",
    "src/hooks/queries/chat.test.ts",
    "src/hooks/queries/fanfic.test.ts",
    "src/hooks/queries/giphy.test.ts",
    "src/hooks/queries/journal.test.ts",
    "src/hooks/queries/quoteCharacters.test.ts",
    "src/hooks/queries/site.test.ts",
    "src/hooks/queries/user.test.ts",
    "src/hooks/useProfileSettingsForm.test.ts",
];
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
const rawQueryKeyMessage =
    "Query keys are built in src/api/queryKeys.ts only. Call the builder instead of writing a key array.";
const dynamicApiImportMessage =
    "Reaching src/api through import() bypasses the layer rules. Import it in the layer that is allowed to.";
const renderTestTransportMockMessage =
    "A render test may not mock api/endpoints, api/queryKeys, api/client or api/queryClient. Mock the hook in src/hooks that the component calls, so the test states the same contract the component is allowed to depend on.";
const stubGlobalFetchMessage =
    "Only the transport layer's own test may stub global fetch. A test above the transport mocks the module it calls instead of the network.";
const pureTestRenderMessage =
    "A test under domain or utils may not import @testing-library/react. A pure module takes values and returns values, so its test needs no DOM.";

const rawQueryKeySelectors = [
    {
        selector:
            "CallExpression[callee.property.name=/^(setQueryData|getQueryData|setQueriesData|getQueriesData|invalidateQueries|removeQueries|cancelQueries|resetQueries|refetchQueries|fetchQuery|prefetchQuery|ensureQueryData)$/] > ArrayExpression:first-child",
        message: rawQueryKeyMessage,
    },
    {
        selector: "Property[key.name='queryKey'] > ArrayExpression > Literal:first-child",
        message: rawQueryKeyMessage,
    },
];
const dynamicApiImportSelectors = [
    {
        selector: "TSImportType[source.value=/(^|\\/)api\\//]",
        message: dynamicApiImportMessage,
    },
    {
        selector: "ImportExpression[source.value=/(^|\\/)api\\//]",
        message: dynamicApiImportMessage,
    },
];
const stubGlobalFetchSelector = {
    selector:
        "CallExpression[callee.object.name='vi'][callee.property.name='stubGlobal'] > Literal[value='fetch']:first-child",
    message: stubGlobalFetchMessage,
};
const renderTestTransportMockSelector = {
    selector:
        "CallExpression[callee.object.name='vi'][callee.property.name='mock'] > Literal[value=/api\\/(endpoints|queryKeys|client|queryClient)/]:first-child",
    message: renderTestTransportMockMessage,
};
const pureTestRenderSelector = {
    selector: "ImportDeclaration[source.value=/^@testing-library\\/react/]",
    message: pureTestRenderMessage,
};

const everyFileSelectors = [...rawQueryKeySelectors, ...dynamicApiImportSelectors, stubGlobalFetchSelector];

export const layerRules = [
    {
        files: DATA_HOOKS,
        rules: {
            "no-restricted-imports": "off",
        },
    },
    {
        files: ORCHESTRATION,
        ignores: DATA_HOOKS,
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
        ignores: PURE_OVERLAPPING_ORCHESTRATION,
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
        ignores: [...RENDER, ...DATA_HOOKS, ...ORCHESTRATION, ...PURE, ...ADAPTERS, ...COMPOSITION_ROOT],
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
            "no-restricted-syntax": ["error", ...everyFileSelectors],
        },
    },
    {
        files: RENDER_TESTS,
        rules: {
            "no-restricted-syntax": ["error", ...everyFileSelectors, renderTestTransportMockSelector],
        },
    },
    {
        files: PURE_TESTS,
        rules: {
            "no-restricted-syntax": ["error", ...everyFileSelectors, pureTestRenderSelector],
        },
    },
    {
        files: MAY_STUB_GLOBAL_FETCH,
        rules: {
            "no-restricted-syntax": ["error", ...rawQueryKeySelectors, ...dynamicApiImportSelectors],
        },
    },
    {
        files: CACHE_KEY_ASSERTING_TESTS,
        rules: {
            "no-restricted-syntax": ["error", ...dynamicApiImportSelectors, stubGlobalFetchSelector],
        },
    },
    {
        files: RENDER_TESTS_STILL_MOCKING_TRANSPORT,
        rules: {
            "no-restricted-syntax": ["error", ...everyFileSelectors],
        },
    },
    {
        files: ["src/api/queryKeys.ts"],
        rules: {
            "no-restricted-syntax": ["error", ...dynamicApiImportSelectors, stubGlobalFetchSelector],
        },
    },
];
