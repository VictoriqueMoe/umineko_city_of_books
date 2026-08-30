import { describe, expect, it } from "vitest";
import { errorMessage } from "./errorMessage";

class ApiError extends Error {
    status: number;
    body: unknown;
    constructor(status: number, message: string, body: unknown) {
        super(message);
        this.status = status;
        this.body = body;
    }
}

describe("errorMessage", () => {
    it("returns the message of a plain Error", () => {
        // given
        const thrown = new Error("the network went away");

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("the network went away");
    });

    it("returns the server's message from an ApiError, because api/client already unwrapped the body", () => {
        // given
        const body = { error: "that username is already taken" };
        const thrown = new ApiError(409, body.error, body);

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("that username is already taken");
    });

    it("does not leak the status or the raw body of an ApiError into the message", () => {
        // given
        const thrown = new ApiError(500, "API error: 500", { error: "API error: 500", trace: "abc" });

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("API error: 500");
        expect(result).not.toContain("trace");
    });

    it("falls back when a server body carried an empty error string, which api/client passes through", () => {
        // given
        const thrown = new ApiError(400, "", { error: "" });

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("Failed to save");
    });

    it("returns a bare thrown string, because a thrown string is already the message", () => {
        // given
        const thrown = "Failed to fetch";

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("Failed to fetch");
    });

    it("falls back for a blank thrown string", () => {
        // given
        const thrown = "   ";

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("Failed to save");
    });

    it("falls back for null", () => {
        // when
        const result = errorMessage(null, "Failed to save");

        // then
        expect(result).toBe("Failed to save");
    });

    it("falls back for undefined", () => {
        // when
        const result = errorMessage(undefined, "Failed to save");

        // then
        expect(result).toBe("Failed to save");
    });

    it("falls back for an object with no message rather than rendering [object Object]", () => {
        // given
        const thrown = { status: 500, body: null };

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("Failed to save");
    });

    it("reads a message off an error-like object that fails instanceof, such as one crossing a realm", () => {
        // given
        const thrown = { name: "TypeError", message: "Load failed", stack: "TypeError: Load failed" };

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("Load failed");
    });

    it("falls back when message is present but is not a string", () => {
        // given
        const thrown = { message: 404 };

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("Failed to save");
    });

    it("falls back for a thrown primitive that is neither a string nor an object", () => {
        // when
        const result = errorMessage(404, "Failed to save");

        // then
        expect(result).toBe("Failed to save");
    });

    it("trims the message so padding never reaches the user", () => {
        // given
        const thrown = new Error("  the model is unreachable\n");

        // when
        const result = errorMessage(thrown, "Failed to save");

        // then
        expect(result).toBe("the model is unreachable");
    });

    it("returns the fallback exactly as given, without decorating it", () => {
        // when
        const result = errorMessage(new Error("   "), "unknown error");

        // then
        expect(result).toBe("unknown error");
    });
});
