import { describe, it, expect, vi } from "vitest";

vi.mock("@capacitor/core", () => ({
    Capacitor: {
        isNativePlatform: () => false,
        getPlatform: () => "web",
    },
}));

import { apiUrl, authHeaders } from "./client";

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
