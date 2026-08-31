import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./overlay";
import { fetchMock, fetchTextMock, postMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

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

describe("overlay helpers", () => {
    it("fetchOverlayConnectorSEF reads the connector file through the text transport", async () => {
        // given
        fetchTextMock.mockResolvedValue("<sef/>");

        // when
        const result = await api.fetchOverlayConnectorSEF();

        // then
        expect(fetchTextMock).toHaveBeenCalledWith("/overlay/connector.sef");
        expect(result).toBe("<sef/>");
    });
});

describe("the overlay token API", () => {
    const cases: RequestCase[] = [
        {
            name: "getOverlayConnection reads the overlay token",
            call: () => api.getOverlayConnection(),
            transport: fetchMock,
            request: ["/overlay/token"],
        },
        {
            name: "resetOverlayToken posts an empty body to the token reset",
            call: () => api.resetOverlayToken(),
            transport: postMock,
            request: ["/overlay/token/reset", {}],
        },
        {
            name: "testOverlay posts an empty body to the overlay test",
            call: () => api.testOverlay(),
            transport: postMock,
            request: ["/overlay/test", {}],
        },
    ];

    runRequestCases(cases);
});
