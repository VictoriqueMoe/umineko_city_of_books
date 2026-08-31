import { beforeEach, describe, vi } from "vitest";
import * as api from "./sidebar";
import { fetchMock, postMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

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

describe("home and sidebar activity", () => {
    const cases: RequestCase[] = [
        {
            name: "getHomeActivity reads the home activity feed",
            call: () => api.getHomeActivity(),
            transport: fetchMock,
            request: ["/home/activity"],
        },
        {
            name: "getSidebarActivity reads the sidebar activity counts",
            call: () => api.getSidebarActivity(),
            transport: fetchMock,
            request: ["/sidebar/activity"],
        },
        {
            name: "getSidebarLastVisited reads the last visited markers",
            call: () => api.getSidebarLastVisited(),
            transport: fetchMock,
            request: ["/sidebar/last-visited"],
        },
        {
            name: "markSidebarVisited posts the key of the visited section",
            call: () => api.markSidebarVisited("theories"),
            transport: postMock,
            request: ["/sidebar/last-visited", { key: "theories" }],
        },
    ];

    runRequestCases(cases);
});
