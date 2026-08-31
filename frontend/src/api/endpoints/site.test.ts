import { beforeEach, describe, vi } from "vitest";
import * as api from "./site";
import { fetchMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

vi.mock("../../platform/capabilities", () => ({
    isNativeApp: () => false,
    clientPlatform: () => "web",
}));

vi.mock("../client", async importOriginal => {
    const actual = await importOriginal<typeof import("../client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiFetchText: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

beforeEach(resetTransports);

describe("site metadata", () => {
    const cases: RequestCase[] = [
        {
            name: "getSiteInfo reads the site info document",
            call: () => api.getSiteInfo(),
            transport: fetchMock,
            request: ["/site-info"],
        },
        {
            name: "getStaff reads the staff list",
            call: () => api.getStaff(),
            transport: fetchMock,
            request: ["/staff"],
        },
    ];

    runRequestCases(cases);
});

describe("rule pages", () => {
    const cases: RequestCase[] = [
        {
            name: "getRules reads the named rules page",
            call: () => api.getRules("chat"),
            transport: fetchMock,
            request: ["/rules/chat"],
        },
    ];

    runRequestCases(cases);
});
