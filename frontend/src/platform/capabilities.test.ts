import { beforeEach, describe, expect, it, vi } from "vitest";

const capacitor = vi.hoisted(() => ({
    isNativePlatform: vi.fn(),
    getPlatform: vi.fn(),
}));

vi.mock("@capacitor/core", () => ({ Capacitor: capacitor }));

async function loadCapabilitiesModule(): Promise<typeof import("./capabilities")> {
    vi.resetModules();
    return import("./capabilities");
}

beforeEach(() => {
    capacitor.isNativePlatform.mockReturnValue(false);
    capacitor.getPlatform.mockReturnValue("web");
});

describe("isNativeApp", () => {
    it("reports a browser session as not native", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(false);
        const { isNativeApp } = await loadCapabilitiesModule();

        // when
        const native = isNativeApp();

        // then
        expect(native).toBe(false);
    });

    it("reports a capacitor shell as native", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(true);
        const { isNativeApp } = await loadCapabilitiesModule();

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
        const { clientPlatform } = await loadCapabilitiesModule();

        // when
        const platform = clientPlatform();

        // then
        expect(platform).toBe("android");
    });
});
