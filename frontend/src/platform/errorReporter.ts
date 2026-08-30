export type PlatformErrorSource = "native-push" | "ota-update";

export type PlatformErrorReporter = (error: unknown, context: { source: PlatformErrorSource }) => void;

let reporter: PlatformErrorReporter | null = null;

export function setPlatformErrorReporter(report: PlatformErrorReporter): void {
    reporter = report;
}

export function reportPlatformError(error: unknown, source: PlatformErrorSource): void {
    reporter?.(error, { source });
}
