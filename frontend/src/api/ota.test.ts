import { beforeEach, describe, expect, it, vi } from "vitest";
import { getOtaManifest, otaBundleUrl } from "./ota";

vi.mock("./client", () => ({ apiUrl: (path: string) => `https://api.test${path}` }));

const fetchMock = vi.fn();

function manifestResponse(body: unknown, ok = true): Response {
    return { ok, json: () => Promise.resolve(body) } as unknown as Response;
}

beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
});

describe("getOtaManifest", () => {
    it("reads the manifest from our own origin without letting a cache answer", async () => {
        // given
        const manifest = { version: "1.1.0", path: "/app-bundles/1.1.0.zip", checksum: "sum", session_key: "iv:key" };
        fetchMock.mockResolvedValue(manifestResponse(manifest));

        // when
        const got = await getOtaManifest();

        // then
        expect(fetchMock).toHaveBeenCalledWith("https://api.test/app-bundles/latest.json", { cache: "no-store" });
        expect(got).toEqual(manifest);
    });

    it("reports no manifest when the request comes back with an error status", async () => {
        // given
        fetchMock.mockResolvedValue(manifestResponse({}, false));

        // when
        const got = await getOtaManifest();

        // then
        expect(got).toBeNull();
    });

    it("lets a transport failure reach the caller rather than swallowing it", async () => {
        // given
        fetchMock.mockRejectedValue(new Error("offline"));

        // when
        const attempt = getOtaManifest();

        // then
        await expect(attempt).rejects.toThrow("offline");
    });
});

describe("otaBundleUrl", () => {
    it("resolves a manifest bundle path against the api origin", () => {
        // given
        const path = "/app-bundles/1.1.0.zip";

        // when
        const got = otaBundleUrl(path);

        // then
        expect(got).toBe("https://api.test/app-bundles/1.1.0.zip");
    });
});
