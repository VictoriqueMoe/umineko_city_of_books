import { beforeEach, describe, vi } from "vitest";
import * as api from "./character";
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

describe("the canon character API", () => {
    const cases: RequestCase[] = [
        {
            name: "listCharacters reads the cast of a series",
            call: () => api.listCharacters("umineko"),
            transport: fetchMock,
            request: ["/characters/umineko"],
        },
    ];

    runRequestCases(cases);
});
