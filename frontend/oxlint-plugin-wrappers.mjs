const emptyBodyMessage =
    "This function has an empty body, so calling it does nothing. Delete it and drop the call at each call site.";
const forwardingMessage =
    "This function only renames {{callee}}. Delete it and call {{callee}} directly at each call site.";

const functionSelectors = [
    "Program > FunctionDeclaration",
    "Program > ExportNamedDeclaration > FunctionDeclaration",
    "Program > ExportDefaultDeclaration > FunctionDeclaration",
    "Program > ExportDefaultDeclaration > FunctionExpression",
    "Program > ExportDefaultDeclaration > ArrowFunctionExpression",
];
const declaratorSelectors = [
    "Program > VariableDeclaration > VariableDeclarator",
    "Program > ExportNamedDeclaration > VariableDeclaration > VariableDeclarator",
];

const functionExpressionTypes = new Set(["FunctionExpression", "ArrowFunctionExpression"]);
const typeWrapperTypes = new Set([
    "TSAsExpression",
    "TSSatisfiesExpression",
    "TSNonNullExpression",
    "TSInstantiationExpression",
    "TSTypeAssertion",
]);
const callTypes = new Set(["CallExpression", "NewExpression"]);

const stripTypeWrappers = expression => {
    let current = expression;

    while (current && typeWrapperTypes.has(current.type)) {
        current = current.expression;
    }

    return current;
};

const asCall = expression => {
    let current = stripTypeWrappers(expression);

    if (current && current.type === "AwaitExpression") {
        current = stripTypeWrappers(current.argument);
    }

    if (current && callTypes.has(current.type)) {
        return current;
    }

    return null;
};

const passesParamsThrough = (params, args) => {
    if (params.length !== args.length) {
        return false;
    }

    for (const [index, param] of params.entries()) {
        const argument = args[index];

        if (param.type === "Identifier" && argument.type === "Identifier" && argument.name === param.name) {
            continue;
        }

        const restsMatch =
            param.type === "RestElement" &&
            param.argument.type === "Identifier" &&
            argument.type === "SpreadElement" &&
            argument.argument.type === "Identifier" &&
            argument.argument.name === param.argument.name;

        if (restsMatch) {
            continue;
        }

        return false;
    }

    return true;
};

const calleeText = (context, call) => {
    const text = context.sourceCode.getText(call.callee);

    if (call.type === "NewExpression") {
        return `new ${text}`;
    }

    return text;
};

const reportForwarding = (context, fn, reportNode, expression, span) => {
    if (!expression) {
        return;
    }

    const loc = context.sourceCode.getLoc(span);

    if (loc.start.line !== loc.end.line) {
        return;
    }

    const call = asCall(expression);

    if (!call) {
        return;
    }

    if (!passesParamsThrough(fn.params, call.arguments)) {
        return;
    }

    context.report({ node: reportNode, messageId: "forwarding", data: { callee: calleeText(context, call) } });
};

const checkFunction = (context, fn, reportNode) => {
    const body = fn.body;

    if (!body) {
        return;
    }

    if (body.type !== "BlockStatement") {
        reportForwarding(context, fn, reportNode, body, body);
        return;
    }

    const statements = body.body;

    if (statements.length === 0) {
        context.report({ node: reportNode, messageId: "emptyBody" });
        return;
    }

    if (statements.length !== 1) {
        return;
    }

    const statement = statements[0];

    if (statement.type === "ReturnStatement") {
        reportForwarding(context, fn, reportNode, statement.argument, statement);
        return;
    }

    if (statement.type === "ExpressionStatement") {
        reportForwarding(context, fn, reportNode, statement.expression, statement);
    }
};

const noForwarder = {
    meta: {
        type: "problem",
        messages: {
            emptyBody: emptyBodyMessage,
            forwarding: forwardingMessage,
        },
    },
    create(context) {
        const visitor = {};

        for (const selector of functionSelectors) {
            visitor[selector] = node => checkFunction(context, node, node);
        }

        for (const selector of declaratorSelectors) {
            visitor[selector] = node => {
                const init = stripTypeWrappers(node.init);

                if (!init || !functionExpressionTypes.has(init.type)) {
                    return;
                }

                checkFunction(context, init, node);
            };
        }

        return visitor;
    },
};

export default {
    meta: { name: "wrappers" },
    rules: {
        "no-forwarder": noForwarder,
    },
};
