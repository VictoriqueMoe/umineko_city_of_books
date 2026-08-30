import { reportClientError } from "./telemetry";

window.addEventListener("error", event => {
    reportClientError(event.error ?? event.message, { source: "window-error" });
});

window.addEventListener("unhandledrejection", event => {
    reportClientError(event.reason, { source: "unhandled-rejection" });
});
