import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

async function loadTelemetryModule(): Promise<typeof import("./telemetry")> {
    vi.resetModules();
    return import("./telemetry");
}

function sink() {
    return vi.mocked(console.error);
}

beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
});

afterEach(() => {
    vi.restoreAllMocks();
    window.history.pushState({}, "", "/");
});

describe("reportClientError", () => {
    it("sends one report to the console sink under a greppable prefix", async () => {
        // given
        const { CLIENT_ERROR_PREFIX, reportClientError } = await loadTelemetryModule();

        // when
        reportClientError(new Error("the golden land is closed"), { source: "uncaught" });

        // then
        expect(sink()).toHaveBeenCalledTimes(1);
        expect(sink()).toHaveBeenCalledWith(CLIENT_ERROR_PREFIX, expect.anything());
    });

    it("carries the error, the component stack, the path and the app version", async () => {
        // given
        window.history.pushState({}, "", "/mysteries/beato");
        vi.stubGlobal("__APP_VERSION__", "7.3.2");
        const { CLIENT_ERROR_PREFIX, reportClientError } = await loadTelemetryModule();
        const error = new TypeError("witches do not exist");

        // when
        reportClientError(error, { source: "error-boundary", componentStack: "\n    at RootErrorBoundary" });

        // then
        expect(sink()).toHaveBeenCalledWith(CLIENT_ERROR_PREFIX, {
            name: "TypeError",
            message: "witches do not exist",
            stack: error.stack ?? null,
            componentStack: "\n    at RootErrorBoundary",
            source: "error-boundary",
            pathname: "/mysteries/beato",
            version: "7.3.2",
            sequence: 1,
        });
    });

    it("says the version is unknown when the build stamped none", async () => {
        // given
        const { CLIENT_ERROR_PREFIX, reportClientError } = await loadTelemetryModule();

        // when
        reportClientError(new Error("no stamp"), { source: "boot" });

        // then
        expect(sink()).toHaveBeenCalledWith(CLIENT_ERROR_PREFIX, expect.objectContaining({ version: "unknown" }));
    });

    it("records a null component stack when the caller has none", async () => {
        // given
        const { CLIENT_ERROR_PREFIX, reportClientError } = await loadTelemetryModule();

        // when
        reportClientError(new Error("no stack"), { source: "window-error" });

        // then
        expect(sink()).toHaveBeenCalledWith(CLIENT_ERROR_PREFIX, expect.objectContaining({ componentStack: null }));
    });

    it("sends every report up to the session cap", async () => {
        // given
        const { MAX_CLIENT_ERROR_REPORTS, reportClientError } = await loadTelemetryModule();

        // when
        for (let i = 0; i < MAX_CLIENT_ERROR_REPORTS; i += 1) {
            reportClientError(new Error(`render loop ${i}`), { source: "recoverable" });
        }

        // then
        expect(sink()).toHaveBeenCalledTimes(MAX_CLIENT_ERROR_REPORTS);
    });

    it("stops sending once the session cap is reached", async () => {
        // given
        const { MAX_CLIENT_ERROR_REPORTS, reportClientError } = await loadTelemetryModule();

        // when
        for (let i = 0; i < MAX_CLIENT_ERROR_REPORTS * 4; i += 1) {
            reportClientError(new Error(`render loop ${i}`), { source: "recoverable" });
        }

        // then
        expect(sink()).toHaveBeenCalledTimes(MAX_CLIENT_ERROR_REPORTS);
    });

    it("stays silent on every further call past the cap", async () => {
        // given
        const { MAX_CLIENT_ERROR_REPORTS, reportClientError } = await loadTelemetryModule();
        for (let i = 0; i < MAX_CLIENT_ERROR_REPORTS; i += 1) {
            reportClientError(new Error(`render loop ${i}`), { source: "recoverable" });
        }
        sink().mockClear();

        // when
        reportClientError(new Error("first past the cap"), { source: "recoverable" });
        reportClientError(new Error("second past the cap"), { source: "recoverable" });

        // then
        expect(sink()).not.toHaveBeenCalled();
    });

    it("numbers each report so a capped session shows where it stopped", async () => {
        // given
        const { CLIENT_ERROR_PREFIX, reportClientError } = await loadTelemetryModule();

        // when
        reportClientError(new Error("first"), { source: "caught" });
        reportClientError(new Error("second"), { source: "caught" });

        // then
        expect(sink()).toHaveBeenLastCalledWith(CLIENT_ERROR_PREFIX, expect.objectContaining({ sequence: 2 }));
    });

    it("still reports a thrown value that is not an Error", async () => {
        // given
        const { CLIENT_ERROR_PREFIX, reportClientError } = await loadTelemetryModule();

        // when
        reportClientError("a bare string was thrown", { source: "unhandled-rejection" });

        // then
        expect(sink()).toHaveBeenCalledWith(
            CLIENT_ERROR_PREFIX,
            expect.objectContaining({ name: "NonError", message: "a bare string was thrown", stack: null }),
        );
    });

    it("still reports a thrown value that cannot be serialised", async () => {
        // given
        const { CLIENT_ERROR_PREFIX, reportClientError } = await loadTelemetryModule();
        const circular: Record<string, unknown> = {};
        circular.self = circular;

        // when
        const report = () => reportClientError(circular, { source: "unhandled-rejection" });

        // then
        expect(report).not.toThrow();
        expect(sink()).toHaveBeenCalledWith(
            CLIENT_ERROR_PREFIX,
            expect.objectContaining({ name: "NonError", message: "unserialisable thrown value" }),
        );
    });

    it("still reports an Error whose own fields throw when read", async () => {
        // given
        const { CLIENT_ERROR_PREFIX, reportClientError } = await loadTelemetryModule();
        const hostile = new Error("readable");
        Object.defineProperty(hostile, "message", {
            get() {
                throw new Error("this field is a trap");
            },
        });

        // when
        const report = () => reportClientError(hostile, { source: "caught" });

        // then
        expect(report).not.toThrow();
        expect(sink()).toHaveBeenCalledWith(
            CLIENT_ERROR_PREFIX,
            expect.objectContaining({ name: "NonError", message: "unreadable thrown value" }),
        );
    });

    it("does not throw when the console sink itself fails", async () => {
        // given
        const { reportClientError } = await loadTelemetryModule();
        sink().mockImplementation(() => {
            throw new Error("the console is gone");
        });

        // when
        const report = () => reportClientError(new Error("boom"), { source: "uncaught" });

        // then
        expect(report).not.toThrow();
    });

    it("does not throw when the caller hands it no context at all", async () => {
        // given
        const { reportClientError } = await loadTelemetryModule();
        const untyped = reportClientError as (error: unknown, context: unknown) => void;

        // when
        const report = () => untyped(new Error("boom"), null);

        // then
        expect(report).not.toThrow();
    });
});
