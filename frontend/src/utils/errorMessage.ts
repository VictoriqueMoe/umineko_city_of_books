function readMessage(thrown: unknown): string {
    if (typeof thrown === "string") {
        return thrown;
    }

    if (thrown === null || typeof thrown !== "object") {
        return "";
    }

    const message = (thrown as { message?: unknown }).message;

    return typeof message === "string" ? message : "";
}

export function errorMessage(e: unknown, fallback: string): string {
    const message = readMessage(e).trim();

    return message === "" ? fallback : message;
}
