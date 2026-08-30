import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { nonFatal } from "./nonFatal";

beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.spyOn(console, "log").mockImplementation(() => {});
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("nonFatal", () => {
    it("settles a rejected promise so the swallow is a decision the reader can see", async () => {
        // given
        const rejected = Promise.reject(new Error("autoplay was blocked"));

        // when
        const settled = rejected.catch(nonFatal);

        // then
        await expect(settled).resolves.toBeUndefined();
    });

    it("returns nothing when called directly", () => {
        // when
        const result = nonFatal();

        // then
        expect(result).toBeUndefined();
    });

    it("reports nowhere, because a marker that logs is not a marker", async () => {
        // given
        const rejected = Promise.reject(new Error("autoplay was blocked"));

        // when
        await rejected.catch(nonFatal);

        // then
        expect(vi.mocked(console.error)).not.toHaveBeenCalled();
        expect(vi.mocked(console.warn)).not.toHaveBeenCalled();
        expect(vi.mocked(console.log)).not.toHaveBeenCalled();
    });
});
