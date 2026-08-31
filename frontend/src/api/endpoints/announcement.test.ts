import { beforeEach, describe, vi } from "vitest";
import * as api from "./announcement";
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

describe("the announcement API", () => {
    const cases: RequestCase[] = [
        {
            name: "listAnnouncements defaults to the first page of twenty",
            call: () => api.listAnnouncements(),
            transport: fetchMock,
            request: ["/announcements?limit=20"],
        },
        {
            name: "listAnnouncements pages through the announcements",
            call: () => api.listAnnouncements(5, 10),
            transport: fetchMock,
            request: ["/announcements?limit=5&offset=10"],
        },
        {
            name: "getAnnouncement reads a single announcement",
            call: () => api.getAnnouncement("an-1"),
            transport: fetchMock,
            request: ["/announcements/an-1"],
        },
        {
            name: "getLatestAnnouncement reads the latest announcement from its own path",
            call: () => api.getLatestAnnouncement(),
            transport: fetchMock,
            request: ["/announcements-latest"],
        },
    ];

    runRequestCases(cases);
});
