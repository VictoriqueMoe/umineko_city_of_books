import { beforeEach, describe, vi } from "vitest";
import * as api from "./secret";
import { fetchMock, putMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

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

describe("the secret API", () => {
    const cases: RequestCase[] = [
        {
            name: "listSecrets reads the secret collection",
            call: () => api.listSecrets(),
            transport: fetchMock,
            request: ["/secrets"],
        },
        {
            name: "getSecret reads a single secret",
            call: () => api.getSecret("sec-1"),
            transport: fetchMock,
            request: ["/secrets/sec-1"],
        },
        {
            name: "unlockSecret puts the secret and the guessed phrase",
            call: () => api.unlockSecret("golden-land", "without love it cannot be seen"),
            transport: putMock,
            request: [
                "/preferences/secret-unlock",
                { secret: "golden-land", phrase: "without love it cannot be seen" },
            ],
        },
    ];

    runRequestCases(cases);
});
