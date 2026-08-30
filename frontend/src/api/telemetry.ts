export type ClientErrorSource =
    | "uncaught"
    | "caught"
    | "recoverable"
    | "error-boundary"
    | "window-error"
    | "unhandled-rejection"
    | "boot"
    | "realtime-contract"
    | "native-push"
    | "ota-update";

export type ClientErrorContext = {
    source: ClientErrorSource;
    componentStack?: string | null;
};

export type ClientErrorReport = {
    name: string;
    message: string;
    stack: string | null;
    componentStack: string | null;
    source: ClientErrorSource;
    pathname: string;
    version: string;
    sequence: number;
};

type ErrorFacts = {
    name: string;
    message: string;
    stack: string | null;
};

export const CLIENT_ERROR_PREFIX = "[client-error]";

export const MAX_CLIENT_ERROR_REPORTS = 50;

const UNKNOWN = "unknown";

let sent = 0;

function describeValue(value: unknown): string {
    try {
        if (typeof value === "string") {
            return value;
        }

        return JSON.stringify(value) ?? String(value);
    } catch {
        return "unserialisable thrown value";
    }
}

function describeError(error: unknown): ErrorFacts {
    try {
        if (error instanceof Error) {
            return { name: error.name, message: error.message, stack: error.stack ?? null };
        }

        return { name: "NonError", message: describeValue(error), stack: null };
    } catch {
        return { name: "NonError", message: "unreadable thrown value", stack: null };
    }
}

function currentPathname(): string {
    return globalThis.location?.pathname ?? UNKNOWN;
}

function appVersion(): string {
    return typeof __APP_VERSION__ === "string" ? __APP_VERSION__ : UNKNOWN;
}

export function reportClientError(error: unknown, context: ClientErrorContext): void {
    if (sent >= MAX_CLIENT_ERROR_REPORTS) {
        return;
    }

    sent += 1;

    try {
        const facts = describeError(error);

        const report: ClientErrorReport = {
            name: facts.name,
            message: facts.message,
            stack: facts.stack,
            componentStack: context.componentStack ?? null,
            source: context.source,
            pathname: currentPathname(),
            version: appVersion(),
            sequence: sent,
        };

        console.error(CLIENT_ERROR_PREFIX, report);
    } catch {}
}
