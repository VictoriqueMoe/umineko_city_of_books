import { afterEach, describe, expect, it, vi } from "vitest";

const CANONICAL = "https://whentheycry.social";

async function loadWithBase(base: string) {
    vi.resetModules();
    vi.stubEnv("VITE_API_BASE", base);

    return await import("./siteOrigin");
}

afterEach(() => {
    vi.unstubAllEnvs();
    vi.resetModules();
});

describe("siteUrl", () => {
    it("builds urls on the canonical origin when no api base is configured", async () => {
        // given
        const { siteUrl } = await loadWithBase("");

        // when
        const url = siteUrl("/theories/1");

        // then
        expect(url).toBe(`${CANONICAL}/theories/1`);
    });

    it("builds urls on the configured api base, keeping only its origin", async () => {
        // given
        const { siteUrl } = await loadWithBase("https://api.example.test/v1/");

        // when
        const url = siteUrl("/theories/1");

        // then
        expect(url).toBe("https://api.example.test/theories/1");
    });

    it("keeps a non default port from the configured api base", async () => {
        // given
        const { siteUrl } = await loadWithBase("http://localhost:8080/api");

        // when
        const url = siteUrl("/theories/1");

        // then
        expect(url).toBe("http://localhost:8080/theories/1");
    });

    it("falls back to the canonical origin when the api base is not a url", async () => {
        // given
        const { siteUrl } = await loadWithBase("not a url at all");

        // when
        const url = siteUrl("/theories/1");

        // then
        expect(url).toBe(`${CANONICAL}/theories/1`);
    });

    it("joins the path on verbatim, without adding a separator", async () => {
        // given
        const { siteUrl } = await loadWithBase("");

        // when / then
        expect(siteUrl("")).toBe(CANONICAL);
        expect(siteUrl("/search?q=beatrice#top")).toBe(`${CANONICAL}/search?q=beatrice#top`);
    });
});

describe("isInternalOrigin", () => {
    it("treats the origin the page is served from as internal", async () => {
        // given
        const { isInternalOrigin } = await loadWithBase("");

        // when
        const result = isInternalOrigin(window.location.origin);

        // then
        expect(result).toBe(true);
    });

    it("treats the canonical origin as internal when no api base is configured", async () => {
        // given
        const { isInternalOrigin } = await loadWithBase("");

        // when / then
        expect(isInternalOrigin(CANONICAL)).toBe(true);
        expect(isInternalOrigin("https://evil.test")).toBe(false);
    });

    it("swaps the canonical origin for the configured api base origin", async () => {
        // given
        const { isInternalOrigin } = await loadWithBase("https://api.example.test/v1/");

        // when / then
        expect(isInternalOrigin("https://api.example.test")).toBe(true);
        expect(isInternalOrigin(CANONICAL)).toBe(false);
    });

    it("compares origins exactly, so look alike origins stay external", async () => {
        // given
        const { isInternalOrigin } = await loadWithBase("");

        // when / then
        expect(isInternalOrigin("http://whentheycry.social")).toBe(false);
        expect(isInternalOrigin("https://evil.whentheycry.social")).toBe(false);
        expect(isInternalOrigin("https://whentheycry.social.evil.test")).toBe(false);
        expect(isInternalOrigin(`${CANONICAL}/`)).toBe(false);
        expect(isInternalOrigin("")).toBe(false);
    });
});
