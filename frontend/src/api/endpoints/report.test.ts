import { beforeEach, describe, vi } from "vitest";
import * as api from "./report";
import { postMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

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

describe("report resolution", () => {
    const cases: RequestCase[] = [
        {
            name: "resolveReport posts the moderator's comment to the numbered report",
            call: () => api.resolveReport(3, "handled in the parlour"),
            transport: postMock,
            request: ["/admin/reports/3/resolve", { comment: "handled in the parlour" }],
        },
        {
            name: "resolveReport still sends an empty comment",
            call: () => api.resolveReport(3, ""),
            transport: postMock,
            request: ["/admin/reports/3/resolve", { comment: "" }],
        },
    ];

    runRequestCases(cases);
});
