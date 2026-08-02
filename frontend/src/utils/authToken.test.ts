import { beforeEach, describe, expect, it, vi } from "vitest";

const TOKEN_KEY = "ut_session_token";

const capacitor = vi.hoisted(() => ({
    isNativePlatform: vi.fn(),
    getPlatform: vi.fn(),
}));

const preferences = vi.hoisted(() => ({
    get: vi.fn(),
    set: vi.fn(),
    remove: vi.fn(),
}));

vi.mock("@capacitor/core", () => ({ Capacitor: capacitor }));
vi.mock("@capacitor/preferences", () => ({ Preferences: preferences }));

async function loadAuthTokenModule(): Promise<typeof import("./authToken")> {
    vi.resetModules();
    return import("./authToken");
}

beforeEach(() => {
    capacitor.isNativePlatform.mockReturnValue(false);
    capacitor.getPlatform.mockReturnValue("web");
    preferences.get.mockResolvedValue({ value: null });
    preferences.set.mockResolvedValue(undefined);
    preferences.remove.mockResolvedValue(undefined);
});

describe("isNativeApp", () => {
    it("reports a browser session as not native", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(false);
        const { isNativeApp } = await loadAuthTokenModule();

        // when
        const native = isNativeApp();

        // then
        expect(native).toBe(false);
    });

    it("reports a capacitor shell as native", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(true);
        const { isNativeApp } = await loadAuthTokenModule();

        // when
        const native = isNativeApp();

        // then
        expect(native).toBe(true);
    });
});

describe("clientPlatform", () => {
    it("passes through whatever platform capacitor reports", async () => {
        // given
        capacitor.getPlatform.mockReturnValue("android");
        const { clientPlatform } = await loadAuthTokenModule();

        // when
        const platform = clientPlatform();

        // then
        expect(platform).toBe("android");
    });
});

describe("getAuthToken", () => {
    it("returns nothing before a token has been stored", async () => {
        // given
        const { getAuthToken } = await loadAuthTokenModule();

        // when
        const token = getAuthToken();

        // then
        expect(token).toBeNull();
    });
});

describe("setAuthToken", () => {
    it("caches the token so it is readable straight away", async () => {
        // given
        const { getAuthToken, setAuthToken } = await loadAuthTokenModule();

        // when
        setAuthToken("beato-token");

        // then
        expect(getAuthToken()).toBe("beato-token");
    });

    it("persists the token under the session key", async () => {
        // given
        const { setAuthToken } = await loadAuthTokenModule();

        // when
        setAuthToken("beato-token");

        // then
        expect(preferences.set).toHaveBeenCalledWith({ key: TOKEN_KEY, value: "beato-token" });
    });

    it("replaces a previously cached token", async () => {
        // given
        const { getAuthToken, setAuthToken } = await loadAuthTokenModule();
        setAuthToken("first-token");

        // when
        setAuthToken("second-token");

        // then
        expect(getAuthToken()).toBe("second-token");
        expect(preferences.set).toHaveBeenLastCalledWith({ key: TOKEN_KEY, value: "second-token" });
    });

    it("still caches the token when persistence fails", async () => {
        // given
        preferences.set.mockRejectedValue(new Error("storage unavailable"));
        const { getAuthToken, setAuthToken } = await loadAuthTokenModule();

        // when
        setAuthToken("beato-token");
        await Promise.resolve();

        // then
        expect(getAuthToken()).toBe("beato-token");
    });
});

describe("clearAuthToken", () => {
    it("forgets the cached token", async () => {
        // given
        const { clearAuthToken, getAuthToken, setAuthToken } = await loadAuthTokenModule();
        setAuthToken("beato-token");

        // when
        clearAuthToken();

        // then
        expect(getAuthToken()).toBeNull();
    });

    it("removes the stored copy of the token", async () => {
        // given
        const { clearAuthToken } = await loadAuthTokenModule();

        // when
        clearAuthToken();

        // then
        expect(preferences.remove).toHaveBeenCalledWith({ key: TOKEN_KEY });
    });

    it("still forgets the token when removal fails", async () => {
        // given
        preferences.remove.mockRejectedValue(new Error("storage unavailable"));
        const { clearAuthToken, getAuthToken, setAuthToken } = await loadAuthTokenModule();
        setAuthToken("beato-token");

        // when
        clearAuthToken();
        await Promise.resolve();

        // then
        expect(getAuthToken()).toBeNull();
    });
});

describe("loadAuthToken", () => {
    it("never reads storage in a browser session", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(false);
        const { getAuthToken, loadAuthToken, setAuthToken } = await loadAuthTokenModule();
        setAuthToken("beato-token");

        // when
        await loadAuthToken();

        // then
        expect(preferences.get).not.toHaveBeenCalled();
        expect(getAuthToken()).toBe("beato-token");
    });

    it("restores a stored token into the cache on a native shell", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(true);
        preferences.get.mockResolvedValue({ value: "stored-token" });
        const { getAuthToken, loadAuthToken } = await loadAuthTokenModule();

        // when
        await loadAuthToken();

        // then
        expect(preferences.get).toHaveBeenCalledWith({ key: TOKEN_KEY });
        expect(getAuthToken()).toBe("stored-token");
    });

    it("leaves the cache empty when the native shell has nothing stored", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(true);
        preferences.get.mockResolvedValue({ value: null });
        const { getAuthToken, loadAuthToken } = await loadAuthTokenModule();

        // when
        await loadAuthToken();

        // then
        expect(getAuthToken()).toBeNull();
    });

    it("keeps a token captured in this session ahead of the stored one", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(true);
        preferences.get.mockResolvedValue({ value: "stored-token" });
        const { getAuthToken, loadAuthToken, setAuthToken } = await loadAuthTokenModule();
        setAuthToken("in-memory-token");

        // when
        await loadAuthToken();

        // then
        expect(getAuthToken()).toBe("in-memory-token");
    });

    it("does not wipe a token captured while storage was still being read", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(true);
        let release: (result: { value: string | null }) => void = () => {};
        preferences.get.mockReturnValue(
            new Promise<{ value: string | null }>(resolve => {
                release = resolve;
            }),
        );
        const { getAuthToken, loadAuthToken, setAuthToken } = await loadAuthTokenModule();
        const pending = loadAuthToken();

        // when
        setAuthToken("header-token");
        release({ value: null });
        await pending;

        // then
        expect(getAuthToken()).toBe("header-token");
    });
});
