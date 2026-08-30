import mainSource from "./main.tsx?raw";
import type { ReactElement } from "react";
import type { RootOptions } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const {
    createRoot,
    renderRoot,
    reportClientError,
    loadAuthToken,
    initAppUpdates,
    watchInstallPrompt,
    getOtaManifest,
    otaBundleUrl,
    setPlatformErrorReporter,
} = vi.hoisted(() => ({
    createRoot: vi.fn(),
    renderRoot: vi.fn(),
    reportClientError: vi.fn(),
    loadAuthToken: vi.fn(),
    initAppUpdates: vi.fn(),
    watchInstallPrompt: vi.fn(),
    getOtaManifest: vi.fn(),
    otaBundleUrl: vi.fn(),
    setPlatformErrorReporter: vi.fn(),
}));

vi.mock("react-dom/client", () => ({ createRoot }));
vi.mock("./api/telemetry", () => ({ reportClientError }));
vi.mock("./api/authToken", () => ({ loadAuthToken }));
vi.mock("./api/ota", () => ({ getOtaManifest, otaBundleUrl }));
vi.mock("./platform/appUpdate", () => ({ initAppUpdates }));
vi.mock("./platform/errorReporter", () => ({ setPlatformErrorReporter }));
vi.mock("./platform/installPrompt", () => ({ watchInstallPrompt }));
vi.mock("./App", () => ({
    default: function App() {
        return null;
    },
}));

type AnyElement = ReactElement<Record<string, unknown>>;

type ErrorCallbacks = Required<Pick<RootOptions, "onUncaughtError" | "onCaughtError" | "onRecoverableError">>;

type BoundaryReporter = (error: unknown, componentStack: string | null) => void;

const registered: Array<[string, EventListenerOrEventListenerObject]> = [];

function isElementLike(node: unknown): node is AnyElement {
    return typeof node === "object" && node !== null && "type" in node && "props" in node;
}

function elementOr(node: unknown, missing: string): AnyElement {
    if (!isElementLike(node)) {
        throw new Error(missing);
    }

    return node;
}

function providerChain(node: unknown): string[] {
    const names: string[] = [];
    let current: unknown = node;

    while (isElementLike(current)) {
        names.push(typeof current.type === "function" ? current.type.name : String(current.type));
        current = current.props.children;
    }

    return names;
}

async function bootMain(): Promise<void> {
    vi.resetModules();
    await import("./main");

    await new Promise<void>(resolve => {
        setTimeout(() => resolve(), 0);
    });
}

function renderedTree(): AnyElement {
    return elementOr(renderRoot.mock.calls[0][0], "nothing was rendered");
}

function mountedBoundary(): AnyElement {
    return elementOr(renderedTree().props.children, "the root error boundary was not mounted");
}

function errorCallbacks(): ErrorCallbacks {
    const options: RootOptions = createRoot.mock.calls[0][1];

    if (!options.onUncaughtError || !options.onCaughtError || !options.onRecoverableError) {
        throw new Error("createRoot was handed no error callbacks");
    }

    return {
        onUncaughtError: options.onUncaughtError,
        onCaughtError: options.onCaughtError,
        onRecoverableError: options.onRecoverableError,
    };
}

function dispatchRejection(reason: unknown): void {
    const event = new Event("unhandledrejection");
    Object.defineProperty(event, "reason", { value: reason });

    window.dispatchEvent(event);
}

beforeEach(() => {
    document.body.innerHTML = '<div id="root"></div>';

    const original = window.addEventListener.bind(window);
    vi.spyOn(window, "addEventListener").mockImplementation((type, handler, options) => {
        if (handler) {
            registered.push([type, handler]);
        }

        original(type, handler, options);
    });

    createRoot.mockReturnValue({ render: renderRoot, unmount: vi.fn() });
    loadAuthToken.mockResolvedValue(undefined);
    initAppUpdates.mockReturnValue(undefined);
});

afterEach(() => {
    for (const [type, handler] of registered) {
        window.removeEventListener(type, handler);
    }

    registered.length = 0;
    vi.restoreAllMocks();
});

describe("main", () => {
    it("renders the app once the stored auth token has been read", async () => {
        // given
        loadAuthToken.mockResolvedValue(undefined);

        // when
        await bootMain();

        // then
        expect(watchInstallPrompt).toHaveBeenCalledTimes(1);
        expect(renderRoot).toHaveBeenCalledTimes(1);
        expect(initAppUpdates).toHaveBeenCalledTimes(1);
    });

    it("hands the update channel the two server calls the platform layer may not import", async () => {
        // given
        loadAuthToken.mockResolvedValue(undefined);

        // when
        await bootMain();

        // then
        expect(initAppUpdates).toHaveBeenCalledWith({ getManifest: getOtaManifest, bundleUrl: otaBundleUrl });
    });

    it("points the platform layer at the same reporter as everything else", async () => {
        // given
        loadAuthToken.mockResolvedValue(undefined);

        // when
        await bootMain();

        // then
        expect(setPlatformErrorReporter).toHaveBeenCalledWith(reportClientError);
    });

    it("boots anyway when the native token read fails, and does not report it", async () => {
        // given
        loadAuthToken.mockRejectedValue(new Error("preferences are unavailable"));

        // when
        await bootMain();

        // then
        expect(renderRoot).toHaveBeenCalledTimes(1);
        expect(reportClientError).not.toHaveBeenCalled();
    });

    it("reports a failure that stopped the boot chain", async () => {
        // given
        const failure = new Error("the update channel is gone");
        initAppUpdates.mockImplementation(() => {
            throw failure;
        });

        // when
        await bootMain();

        // then
        expect(reportClientError).toHaveBeenCalledWith(failure, { source: "boot" });
    });

    it("reports a render error that no boundary caught", async () => {
        // given
        await bootMain();
        const error = new Error("the golden land is closed");

        // when
        errorCallbacks().onUncaughtError(error, { componentStack: "\n    at Beatrice" });

        // then
        expect(reportClientError).toHaveBeenCalledWith(error, {
            source: "uncaught",
            componentStack: "\n    at Beatrice",
        });
    });

    it("reports a render error a boundary caught, which is how a lazy child is covered", async () => {
        // given
        await bootMain();
        const error = new Error("the golden land is closed");

        // when
        errorCallbacks().onCaughtError(error, { componentStack: "\n    at Beatrice" });

        // then
        expect(reportClientError).toHaveBeenCalledWith(error, {
            source: "caught",
            componentStack: "\n    at Beatrice",
        });
    });

    it("reports an error React recovered from on its own", async () => {
        // given
        await bootMain();
        const error = new Error("the render was retried");

        // when
        errorCallbacks().onRecoverableError(error, { componentStack: "\n    at Beatrice" });

        // then
        expect(reportClientError).toHaveBeenCalledWith(error, {
            source: "recoverable",
            componentStack: "\n    at Beatrice",
        });
    });

    it("reports an error that escaped everything and reached the window", async () => {
        // given
        await bootMain();
        const error = new Error("the witch left the board");

        // when
        window.dispatchEvent(new ErrorEvent("error", { error, message: "the witch left the board" }));

        // then
        expect(reportClientError).toHaveBeenCalledWith(error, { source: "window-error" });
    });

    it("falls back to the message when the browser hands over no error object", async () => {
        // given
        await bootMain();

        // when
        window.dispatchEvent(new ErrorEvent("error", { error: null, message: "Script error." }));

        // then
        expect(reportClientError).toHaveBeenCalledWith("Script error.", { source: "window-error" });
    });

    it("reports a promise rejection nobody caught", async () => {
        // given
        await bootMain();
        const reason = new Error("the fetch never landed");

        // when
        dispatchRejection(reason);

        // then
        expect(reportClientError).toHaveBeenCalledWith(reason, { source: "unhandled-rejection" });
    });

    it("puts the boundary above every provider so a provider throw still shows a fallback", async () => {
        // given
        await bootMain();
        const [{ RootErrorBoundary }, { StrictMode }] = await Promise.all([
            import("./components/RootErrorBoundary/RootErrorBoundary"),
            import("react"),
        ]);

        // when
        const tree = renderedTree();
        const boundary = mountedBoundary();

        // then
        expect(tree.type).toBe(StrictMode);
        expect(boundary.type).toBe(RootErrorBoundary);
        expect(providerChain(boundary.props.children)).toEqual([
            "QueryClientProvider",
            "SiteInfoProvider",
            "AuthProvider",
            "ThemeProvider",
            "NotificationProvider",
            "GifFavouritesProvider",
            "MentionResolverProvider",
            "App",
        ]);
    });

    it("installs the global error handlers before any other module is evaluated", () => {
        // given
        const lines = mainSource.split("\n");

        // when
        const firstImport = lines.find((line: string) => line.startsWith("import"));

        // then
        expect(firstImport).toBe('import "./api/globalErrorHandlers";');
    });

    it("wires the boundary to the same reporter as everything else", async () => {
        // given
        await bootMain();
        const boundary = mountedBoundary();
        const error = new Error("a provider gave up");

        // when
        expect(boundary.props.onError).toBeTypeOf("function");
        const onError = boundary.props.onError as BoundaryReporter;
        onError(error, "\n    at ThemeProvider");

        // then
        expect(reportClientError).toHaveBeenCalledWith(error, {
            source: "error-boundary",
            componentStack: "\n    at ThemeProvider",
        });
    });
});
