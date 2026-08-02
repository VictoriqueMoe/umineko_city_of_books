import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { resolveOptimalRegion } from "./hyperbeamRegion";

const mocks = vi.hoisted(() => ({ getRegionInfo: vi.fn() }));

vi.mock("@hyperbeam/web", () => ({
    default: vi.fn(),
    getRegionInfo: mocks.getRegionInfo,
}));

const STORAGE_KEY = "hyperbeam_region";
const DAY_MS = 24 * 60 * 60 * 1000;

function storeCache(region: unknown, storedAt: unknown): void {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ region, storedAt }));
}

beforeEach(() => {
    mocks.getRegionInfo.mockResolvedValue({ region: "NA" });
    vi.spyOn(console, "warn").mockImplementation(() => {});
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("resolveOptimalRegion", () => {
    it("asks hyperbeam for the nearest region when nothing has been cached", async () => {
        // given
        mocks.getRegionInfo.mockResolvedValue({ region: "EU" });

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("EU");
        expect(mocks.getRegionInfo).toHaveBeenCalledOnce();
    });

    it("remembers the resolved region so the next caller never asks again", async () => {
        // given
        mocks.getRegionInfo.mockResolvedValue({ region: "AS" });
        await resolveOptimalRegion();

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("AS");
        expect(mocks.getRegionInfo).toHaveBeenCalledOnce();
    });

    it("stores the region alongside the moment it was resolved", async () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T10:00:00Z"));
        mocks.getRegionInfo.mockResolvedValue({ region: "NA" });

        // when
        await resolveOptimalRegion();

        // then
        expect(JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "null")).toEqual({
            region: "NA",
            storedAt: Date.parse("2026-08-02T10:00:00Z"),
        });
    });

    it("serves a cached region without touching hyperbeam at all", async () => {
        // given
        storeCache("EU", Date.now());

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("EU");
        expect(mocks.getRegionInfo).not.toHaveBeenCalled();
    });

    it("keeps using a cached region right up to the end of its day of life", async () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T10:00:00Z"));
        storeCache("EU", Date.now() - DAY_MS);

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("EU");
        expect(mocks.getRegionInfo).not.toHaveBeenCalled();
    });

    it("re-resolves a region that was cached more than a day ago", async () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T10:00:00Z"));
        storeCache("EU", Date.now() - DAY_MS - 1);
        mocks.getRegionInfo.mockResolvedValue({ region: "NA" });

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("NA");
        expect(mocks.getRegionInfo).toHaveBeenCalledOnce();
    });

    it("ignores a cache entry that is not readable json", async () => {
        // given
        window.localStorage.setItem(STORAGE_KEY, "{not json");
        mocks.getRegionInfo.mockResolvedValue({ region: "NA" });

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("NA");
        expect(mocks.getRegionInfo).toHaveBeenCalledOnce();
    });

    it("ignores a cache entry that carries no region", async () => {
        // given
        storeCache("", Date.now());
        mocks.getRegionInfo.mockResolvedValue({ region: "NA" });

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("NA");
        expect(mocks.getRegionInfo).toHaveBeenCalledOnce();
    });

    it("ignores a cache entry whose timestamp is not a number", async () => {
        // given
        storeCache("EU", "yesterday");
        mocks.getRegionInfo.mockResolvedValue({ region: "NA" });

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("NA");
        expect(mocks.getRegionInfo).toHaveBeenCalledOnce();
    });

    it("falls back to the server default when hyperbeam cannot be reached", async () => {
        // given
        mocks.getRegionInfo.mockRejectedValue(new Error("network unreachable"));

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("");
        expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
    });

    it("falls back to the server default when hyperbeam answers with no region", async () => {
        // given
        mocks.getRegionInfo.mockResolvedValue({ region: "" });

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("");
        expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
    });

    it("falls back to the server default when hyperbeam answers with nothing", async () => {
        // given
        mocks.getRegionInfo.mockResolvedValue(null);

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("");
    });

    it("falls back to the server default when the region is not a string", async () => {
        // given
        mocks.getRegionInfo.mockResolvedValue({ region: 7 });

        // when
        const region = await resolveOptimalRegion();

        // then
        expect(region).toBe("");
    });
});
