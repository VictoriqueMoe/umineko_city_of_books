import { afterEach, describe, expect, it, vi } from "vitest";
import { apiUrl, authHeaders } from "./client";

vi.mock("@capacitor/core", () => ({
    Capacitor: {
        isNativePlatform: () => false,
        getPlatform: () => "web",
    },
}));

describe("apiUrl", () => {
    it("prefixes the api origin (empty on web, so same-origin relative)", () => {
        expect(apiUrl("/api/v1/site-info")).toBe("/api/v1/site-info");
    });
});

describe("authHeaders", () => {
    it("sends no auth headers on web (cookie auth is used instead)", () => {
        expect(authHeaders()).toEqual({});
    });
});

describe("absolutizeMedia (native app with a configured API origin)", () => {
    afterEach(() => {
        vi.unstubAllEnvs();
        vi.resetModules();
    });

    it("absolutizes media `*_url` fields but leaves navigation `url` paths relative", async () => {
        // given
        vi.stubEnv("VITE_API_BASE", "https://whentheycry.social");
        vi.resetModules();
        const { absolutizeMedia } = await import("./client");

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
});
