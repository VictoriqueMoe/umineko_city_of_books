import { afterEach, describe, expect, it, vi } from "vitest";
import { absolutizeMedia, apiUrl } from "./origin";

describe("apiUrl", () => {
    it("prefixes the api origin (empty on web, so same-origin relative)", () => {
        expect(apiUrl("/api/v1/site-info")).toBe("/api/v1/site-info");
    });
});

describe("apiUrl (with a configured API origin)", () => {
    afterEach(() => {
        vi.unstubAllEnvs();
        vi.resetModules();
    });

    it("prefixes every path with the configured origin", async () => {
        // given
        vi.stubEnv("VITE_API_BASE", "https://whentheycry.social");
        vi.resetModules();
        const { apiUrl: prefixedApiUrl } = await import("./origin");

        // when
        const result = prefixedApiUrl("/api/v1/site-info");

        // then
        expect(result).toBe("https://whentheycry.social/api/v1/site-info");
    });
});

describe("absolutizeMedia (web, no configured API origin)", () => {
    it("returns the data untouched because urls are already same-origin", () => {
        // given
        const data = { avatar_url: "/uploads/a.png" };

        // when
        const result = absolutizeMedia(data);

        // then
        expect(result).toBe(data);
    });
});

describe("absolutizeMedia (native app with a configured API origin)", () => {
    afterEach(() => {
        vi.unstubAllEnvs();
        vi.resetModules();
    });

    async function loadWithOrigin(origin: string) {
        vi.stubEnv("VITE_API_BASE", origin);
        vi.resetModules();
        return import("./origin");
    }

    it("absolutizes media `*_url` fields but leaves navigation `url` paths relative", async () => {
        // given
        vi.stubEnv("VITE_API_BASE", "https://whentheycry.social");
        vi.resetModules();
        const { absolutizeMedia } = await import("./origin");

        // when
        const result = absolutizeMedia({
            avatar_url: "/uploads/a.png",
            thumbnail_url: "/uploads/t.png",
            url: "/theories/1",
        });

        // then
        expect(result).toEqual({
            avatar_url: "https://whentheycry.social/uploads/a.png",
            thumbnail_url: "https://whentheycry.social/uploads/t.png",
            url: "/theories/1",
        });
    });

    it("absolutizes media urls nested inside arrays and child objects", async () => {
        // given
        const { absolutizeMedia } = await loadWithOrigin("https://whentheycry.social");

        // when
        const result = absolutizeMedia({
            items: [{ author: { avatar_url: "/uploads/a.png" } }, { author: { avatar_url: "/uploads/b.png" } }],
        });

        // then
        expect(result).toEqual({
            items: [
                { author: { avatar_url: "https://whentheycry.social/uploads/a.png" } },
                { author: { avatar_url: "https://whentheycry.social/uploads/b.png" } },
            ],
        });
    });

    it("leaves already absolute and protocol relative media urls alone", async () => {
        // given
        const { absolutizeMedia } = await loadWithOrigin("https://whentheycry.social");

        // when
        const result = absolutizeMedia({
            avatar_url: "https://cdn.example.com/a.png",
            banner_url: "//cdn.example.com/b.png",
            icon_url: "uploads/c.png",
        });

        // then
        expect(result).toEqual({
            avatar_url: "https://cdn.example.com/a.png",
            banner_url: "//cdn.example.com/b.png",
            icon_url: "uploads/c.png",
        });
    });

    it("leaves values that are not media url strings alone", async () => {
        // given
        const { absolutizeMedia } = await loadWithOrigin("https://whentheycry.social");

        // when
        const result = absolutizeMedia({
            avatar_url: null,
            count: 3,
            title: "/not/a/url",
            nested_url_count: 0,
        });

        // then
        expect(result).toEqual({ avatar_url: null, count: 3, title: "/not/a/url", nested_url_count: 0 });
    });

    it("absolutizes media urls in a top level array", async () => {
        // given
        const { absolutizeMedia } = await loadWithOrigin("https://whentheycry.social");

        // when
        const result = absolutizeMedia([{ image_url: "/uploads/a.png" }]);

        // then
        expect(result).toEqual([{ image_url: "https://whentheycry.social/uploads/a.png" }]);
    });
});
