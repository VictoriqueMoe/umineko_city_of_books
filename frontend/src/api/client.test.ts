import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearAuthToken, getAuthToken, setAuthToken } from "../utils/authToken";
import {
    ApiError,
    absolutizeMedia,
    apiDelete,
    apiDeleteWithBody,
    apiFetch,
    apiPatch,
    apiPost,
    apiPostFormData,
    apiPut,
    apiUrl,
    authHeaders,
    buildQueryString,
} from "./client";

const capacitor = vi.hoisted(() => ({ native: false, platform: "web" }));

vi.mock("@capacitor/core", () => ({
    Capacitor: {
        isNativePlatform: () => capacitor.native,
        getPlatform: () => capacitor.platform,
    },
}));

vi.mock("@capacitor/preferences", () => ({
    Preferences: {
        get: () => Promise.resolve({ value: null }),
        set: () => Promise.resolve(),
        remove: () => Promise.resolve(),
    },
}));

function stubFetch(response: Response) {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(response);
    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
}

function jsonResponse(body: unknown, status = 200, headers: Record<string, string> = {}): Response {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json", ...headers },
    });
}

function goNative(platform = "android"): void {
    capacitor.native = true;
    capacitor.platform = platform;
}

beforeEach(() => {
    capacitor.native = false;
    capacitor.platform = "web";
    clearAuthToken();
});

describe("apiUrl", () => {
    it("prefixes the api origin (empty on web, so same-origin relative)", () => {
        expect(apiUrl("/api/v1/site-info")).toBe("/api/v1/site-info");
    });
});

describe("authHeaders", () => {
    it("sends no auth headers on web (cookie auth is used instead)", () => {
        expect(authHeaders()).toEqual({});
    });

    it("sends the platform and a bearer token in the native app", () => {
        // given
        goNative("android");
        setAuthToken("token-123");

        // when
        const headers = authHeaders();

        // then
        expect(headers).toEqual({ "X-Client-Platform": "android", Authorization: "Bearer token-123" });
    });

    it("omits the authorization header in the native app when no token is stored", () => {
        // given
        goNative("ios");

        // when
        const headers = authHeaders();

        // then
        expect(headers).toEqual({ "X-Client-Platform": "ios" });
    });
});

describe("ApiError", () => {
    it("is an error that carries the status, message and raw body", () => {
        // given
        const body = { error: "Not allowed", code: 7 };

        // when
        const error = new ApiError(403, "Not allowed", body);

        // then
        expect(error).toBeInstanceOf(Error);
        expect(error.status).toBe(403);
        expect(error.message).toBe("Not allowed");
        expect(error.body).toBe(body);
    });
});

describe("apiFetch", () => {
    it("calls the versioned endpoint with cookies and returns the parsed body", async () => {
        // given
        const fetchMock = stubFetch(jsonResponse({ name: "Beatrice" }));

        // when
        const result = await apiFetch<{ name: string }>("/theories/1");

        // then
        expect(fetchMock).toHaveBeenCalledWith("/api/v1/theories/1", { credentials: "include", headers: {} });
        expect(result).toEqual({ name: "Beatrice" });
    });

    it("sends the native auth headers when running in the app", async () => {
        // given
        goNative("android");
        setAuthToken("token-123");
        const fetchMock = stubFetch(jsonResponse({ ok: true }));

        // when
        await apiFetch("/me");

        // then
        expect(fetchMock).toHaveBeenCalledWith("/api/v1/me", {
            credentials: "include",
            headers: { "X-Client-Platform": "android", Authorization: "Bearer token-123" },
        });
    });

    it("resolves to undefined for a 204 no content response", async () => {
        // given
        stubFetch(new Response(null, { status: 204 }));

        // when
        const result = await apiFetch("/theories/1/like");

        // then
        expect(result).toBeUndefined();
    });

    it("resolves to undefined when the response declares a zero content length", async () => {
        // given
        stubFetch(new Response("", { status: 200, headers: { "content-length": "0" } }));

        // when
        const result = await apiFetch("/theories/1/like");

        // then
        expect(result).toBeUndefined();
    });

    it("throws an ApiError carrying the status and the server error message", async () => {
        // given
        stubFetch(jsonResponse({ error: "you are not the golden witch" }, 403));

        // when
        const failure = apiFetch("/admin/users");

        // then
        await expect(failure).rejects.toBeInstanceOf(ApiError);
        await expect(failure).rejects.toMatchObject({
            status: 403,
            message: "you are not the golden witch",
            body: { error: "you are not the golden witch" },
        });
    });

    it("falls back to a generic message when the error body is not json", async () => {
        // given
        stubFetch(new Response("<html>gateway exploded</html>", { status: 502 }));

        // when
        const failure = apiFetch("/theories");

        // then
        await expect(failure).rejects.toMatchObject({ status: 502, message: "API error: 502", body: null });
    });

    it("falls back to a generic message when the json error body has no error field", async () => {
        // given
        stubFetch(jsonResponse({ detail: "unprocessable" }, 422));

        // when
        const failure = apiFetch("/theories");

        // then
        await expect(failure).rejects.toMatchObject({
            status: 422,
            message: "API error: 422",
            body: { detail: "unprocessable" },
        });
    });

    it("captures a session token handed back in the response headers", async () => {
        // given
        stubFetch(jsonResponse({ ok: true }, 200, { "X-Session-Token": "fresh-token" }));

        // when
        await apiFetch("/auth/login");

        // then
        expect(getAuthToken()).toBe("fresh-token");
    });

    it("captures the session token even when the request fails", async () => {
        // given
        stubFetch(jsonResponse({ error: "nope" }, 401, { "X-Session-Token": "fresh-token" }));

        // when
        await expect(apiFetch("/me")).rejects.toBeInstanceOf(ApiError);

        // then
        expect(getAuthToken()).toBe("fresh-token");
    });

    it("leaves the stored token alone when the response carries no session token", async () => {
        // given
        setAuthToken("existing-token");
        stubFetch(jsonResponse({ ok: true }));

        // when
        await apiFetch("/me");

        // then
        expect(getAuthToken()).toBe("existing-token");
    });
});

describe("apiPost", () => {
    it("posts a json body with credentials and returns the parsed response", async () => {
        // given
        const fetchMock = stubFetch(jsonResponse({ id: "t1" }));

        // when
        const result = await apiPost<{ id: string }, { title: string }>("/theories", { title: "The Golden Land" });

        // then
        expect(fetchMock).toHaveBeenCalledWith("/api/v1/theories", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ title: "The Golden Land" }),
            credentials: "include",
        });
        expect(result).toEqual({ id: "t1" });
    });

    it("merges the native auth headers with the json content type", async () => {
        // given
        goNative("android");
        setAuthToken("token-123");
        const fetchMock = stubFetch(jsonResponse({ id: "t1" }));

        // when
        await apiPost("/theories", { title: "x" });

        // then
        expect(fetchMock.mock.calls[0][1]?.headers).toEqual({
            "Content-Type": "application/json",
            "X-Client-Platform": "android",
            Authorization: "Bearer token-123",
        });
    });

    it("throws an ApiError when the server rejects the body", async () => {
        // given
        stubFetch(jsonResponse({ error: "title is required" }, 400));

        // when
        const failure = apiPost("/theories", {});

        // then
        await expect(failure).rejects.toMatchObject({ status: 400, message: "title is required" });
    });
});

describe("apiPut", () => {
    it("sends a json body with the PUT method and credentials", async () => {
        // given
        const fetchMock = stubFetch(jsonResponse({ id: "t1" }));

        // when
        await apiPut<{ id: string }, { title: string }>("/theories/t1", { title: "Revised" });

        // then
        expect(fetchMock).toHaveBeenCalledWith("/api/v1/theories/t1", {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ title: "Revised" }),
            credentials: "include",
        });
    });
});

describe("apiPatch", () => {
    it("sends a json body with the PATCH method and credentials", async () => {
        // given
        const fetchMock = stubFetch(jsonResponse({ id: "t1" }));

        // when
        await apiPatch<{ id: string }, { pinned: boolean }>("/theories/t1", { pinned: true });

        // then
        expect(fetchMock).toHaveBeenCalledWith("/api/v1/theories/t1", {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ pinned: true }),
            credentials: "include",
        });
    });
});

describe("apiDelete", () => {
    it("sends a DELETE with credentials and no body or content type", async () => {
        // given
        const fetchMock = stubFetch(new Response(null, { status: 204 }));

        // when
        const result = await apiDelete("/theories/t1");

        // then
        expect(fetchMock).toHaveBeenCalledWith("/api/v1/theories/t1", {
            method: "DELETE",
            credentials: "include",
            headers: {},
        });
        expect(result).toBeUndefined();
    });
});

describe("apiDeleteWithBody", () => {
    it("sends a DELETE that carries a json body", async () => {
        // given
        const fetchMock = stubFetch(new Response(null, { status: 204 }));

        // when
        await apiDeleteWithBody<void, { ids: string[] }>("/notifications", { ids: ["n1", "n2"] });

        // then
        expect(fetchMock).toHaveBeenCalledWith("/api/v1/notifications", {
            method: "DELETE",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ ids: ["n1", "n2"] }),
            credentials: "include",
        });
    });
});

describe("apiPostFormData", () => {
    it("posts the form data untouched and never sets a json content type", async () => {
        // given
        const formData = new FormData();
        formData.set("file", new Blob(["art"]), "art.png");
        const fetchMock = stubFetch(jsonResponse({ url: "/uploads/art.png" }));

        // when
        await apiPostFormData("/art", formData);

        // then
        expect(fetchMock).toHaveBeenCalledWith("/api/v1/art", {
            method: "POST",
            body: formData,
            credentials: "include",
            headers: {},
        });
        expect(fetchMock.mock.calls[0][1]?.body).toBe(formData);
    });

    it("still sends the native auth headers without a content type", async () => {
        // given
        goNative("ios");
        setAuthToken("token-123");
        const fetchMock = stubFetch(jsonResponse({ ok: true }));

        // when
        await apiPostFormData("/art", new FormData());

        // then
        expect(fetchMock.mock.calls[0][1]?.headers).toEqual({
            "X-Client-Platform": "ios",
            Authorization: "Bearer token-123",
        });
    });
});

describe("buildQueryString", () => {
    it("returns an empty string when there is nothing to send", () => {
        // then
        expect(buildQueryString({})).toBe("");
        expect(buildQueryString({ q: undefined, cursor: "", offset: 0 })).toBe("");
    });

    it("skips undefined and empty string values", () => {
        // given
        const params = { q: "beatrice", cursor: "", limit: 20, sort: undefined };

        // when
        const qs = buildQueryString(params);

        // then
        expect(qs).toBe("?q=beatrice&limit=20");
    });

    it("drops a zero offset and page because zero means unset for paging", () => {
        // given
        const params = { q: "beatrice", offset: 0, page: 0, limit: 20 };

        // when
        const qs = buildQueryString(params);

        // then
        expect(qs).toBe("?q=beatrice&limit=20");
    });

    it("keeps a zero value for every other key", () => {
        // given
        const params = { limit: 0, episode: 0, seed: 0, perType: 0 };

        // when
        const qs = buildQueryString(params);

        // then
        expect(qs).toBe("?limit=0&episode=0&seed=0&perType=0");
    });

    it("keeps the declared parameter order and prefixes a question mark", () => {
        // then
        expect(buildQueryString({ limit: 20, offset: 40 })).toBe("?limit=20&offset=40");
    });

    it("percent-encodes keys and values", () => {
        // then
        expect(buildQueryString({ q: "a b&c=d" })).toBe("?q=a+b%26c%3Dd");
        expect(buildQueryString({ "a key": "v" })).toBe("?a+key=v");
    });

    it("stringifies numbers", () => {
        // then
        expect(buildQueryString({ limit: 20 })).toBe("?limit=20");
        expect(buildQueryString({ limit: -1 })).toBe("?limit=-1");
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
        return import("./client");
    }

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

describe("apiUrl (with a configured API origin)", () => {
    afterEach(() => {
        vi.unstubAllEnvs();
        vi.resetModules();
    });

    it("prefixes every path with the configured origin", async () => {
        // given
        vi.stubEnv("VITE_API_BASE", "https://whentheycry.social");
        vi.resetModules();
        const { apiUrl: prefixedApiUrl } = await import("./client");

        // when
        const result = prefixedApiUrl("/api/v1/site-info");

        // then
        expect(result).toBe("https://whentheycry.social/api/v1/site-info");
    });

    it("targets the configured origin when fetching", async () => {
        // given
        vi.stubEnv("VITE_API_BASE", "https://whentheycry.social");
        vi.resetModules();
        const { apiFetch: prefixedApiFetch } = await import("./client");
        const fetchMock = stubFetch(jsonResponse({ ok: true }));

        // when
        await prefixedApiFetch("/site-info");

        // then
        expect(fetchMock).toHaveBeenCalledWith("https://whentheycry.social/api/v1/site-info", {
            credentials: "include",
            headers: {},
        });
    });
});
