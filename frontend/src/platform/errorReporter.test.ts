import { beforeEach, describe, expect, it, vi } from "vitest";

async function loadReporterModule(): Promise<typeof import("./errorReporter")> {
    vi.resetModules();
    return import("./errorReporter");
}

beforeEach(() => {
    vi.resetModules();
});

describe("reportPlatformError", () => {
    it("hands the failure to the reporter the composition root registered", async () => {
        // given
        const { reportPlatformError, setPlatformErrorReporter } = await loadReporterModule();
        const report = vi.fn();
        setPlatformErrorReporter(report);
        const failure = new Error("the token never arrived");

        // when
        reportPlatformError(failure, "native-push");

        // then
        expect(report).toHaveBeenCalledWith(failure, { source: "native-push" });
    });

    it("stays silent rather than throwing when nothing has been registered yet", async () => {
        // given
        const { reportPlatformError } = await loadReporterModule();

        // when
        const attempt = () => reportPlatformError(new Error("too early"), "ota-update");

        // then
        expect(attempt).not.toThrow();
    });
});
