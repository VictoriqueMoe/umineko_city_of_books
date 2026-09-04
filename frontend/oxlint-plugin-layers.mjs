const rawQueryKeyMessage =
    "Query keys are built in src/api/queryKeys.ts only. Call the builder instead of writing a key array.";
const dynamicApiImportMessage =
    "Reaching src/api through import() bypasses the layer rules. Import it in the layer that is allowed to.";
const stubGlobalFetchMessage =
    "Only the transport layer's own test may stub global fetch. A test above the transport mocks the module it calls instead of the network.";
const renderTestTransportMockMessage =
    "A render test may not mock api/endpoints, api/queryKeys, api/client or api/queryClient. Mock the hook in src/hooks that the component calls, so the test states the same contract the component is allowed to depend on.";
const pureTestRenderMessage =
    "A test under domain or utils may not import @testing-library/react. A pure module takes values and returns values, so its test needs no DOM.";

const queryClientMethods =
    "setQueryData|getQueryData|setQueriesData|getQueriesData|invalidateQueries|removeQueries|cancelQueries|resetQueries|refetchQueries|fetchQuery|prefetchQuery|ensureQueryData";

const rawQueryKeySelectors = [
    `CallExpression[callee.property.name=/^(${queryClientMethods})$/] > ArrayExpression:first-child`,
    "Property[key.name='queryKey'] > ArrayExpression > Literal:first-child",
];
const dynamicApiImportSelectors = [
    "TSImportType[source.value=/(^|\\/)api\\//]",
    "ImportExpression[source.value=/(^|\\/)api\\//]",
];
const stubGlobalFetchSelectors = [
    "CallExpression[callee.object.name='vi'][callee.property.name='stubGlobal'] > Literal[value='fetch']:first-child",
];
const renderTestTransportMockSelectors = [
    "CallExpression[callee.object.name='vi'][callee.property.name='mock'] > Literal[value=/api\\/(endpoints|queryKeys|client|queryClient)/]:first-child",
];
const pureTestRenderSelectors = ["ImportDeclaration[source.value=/^@testing-library\\/react/]"];

const restrict = (message, selectors) => ({
    meta: {
        type: "problem",
        messages: { restricted: message },
    },
    create(context) {
        const visitor = {};

        for (const selector of selectors) {
            visitor[selector] = node => context.report({ node, messageId: "restricted" });
        }

        return visitor;
    },
});

export default {
    meta: { name: "layers" },
    rules: {
        "no-raw-query-key": restrict(rawQueryKeyMessage, rawQueryKeySelectors),
        "no-dynamic-api-import": restrict(dynamicApiImportMessage, dynamicApiImportSelectors),
        "no-stub-global-fetch": restrict(stubGlobalFetchMessage, stubGlobalFetchSelectors),
        "no-transport-mock-in-render-test": restrict(renderTestTransportMockMessage, renderTestTransportMockSelectors),
        "no-render-import-in-pure-test": restrict(pureTestRenderMessage, pureTestRenderSelectors),
    },
};
