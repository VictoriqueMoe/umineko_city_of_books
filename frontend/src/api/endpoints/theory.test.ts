import { beforeEach, describe, vi } from "vitest";
import * as api from "./theory";
import {
    deleteMock,
    fetchMock,
    postMock,
    putMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

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

describe("the theory API", () => {
    const theoryPayload = {
        title: "The gold is real",
        body: "Beatrice hid it beneath the rose garden",
        episode: 4,
        series: "umineko",
        evidence: [{ note: "the letter", quote_index: 2 }],
    };
    const responsePayload = {
        side: "with_love" as const,
        body: "Then the seventh twilight is a trap",
        evidence: [],
    };
    const replyPayload = { ...responsePayload, parent_id: "resp-0" };

    const cases: RequestCase[] = [
        {
            name: "createTheory posts the whole payload to the theory collection",
            call: () => api.createTheory(theoryPayload),
            transport: postMock,
            request: ["/theories", theoryPayload],
        },
        {
            name: "getTheory reads a single theory",
            call: () => api.getTheory("t-1"),
            transport: fetchMock,
            request: ["/theories/t-1"],
        },
        {
            name: "updateTheory puts the whole payload back",
            call: () => api.updateTheory("t-1", theoryPayload),
            transport: putMock,
            request: ["/theories/t-1", theoryPayload],
        },
        {
            name: "deleteTheory deletes the theory",
            call: () => api.deleteTheory("t-1"),
            transport: deleteMock,
            request: ["/theories/t-1"],
        },
        {
            name: "createResponse nests a top level response under its theory",
            call: () => api.createResponse("t-1", responsePayload),
            transport: postMock,
            request: ["/theories/t-1/responses", responsePayload],
        },
        {
            name: "createResponse carries the parent id of a reply",
            call: () => api.createResponse("t-1", replyPayload),
            transport: postMock,
            request: ["/theories/t-1/responses", replyPayload],
        },
        {
            name: "deleteResponse deletes from the response collection",
            call: () => api.deleteResponse("resp-1"),
            transport: deleteMock,
            request: ["/responses/resp-1"],
        },
        {
            name: "voteTheory posts an upvote",
            call: () => api.voteTheory("t-1", 1),
            transport: postMock,
            request: ["/theories/t-1/vote", { value: 1 }],
        },
        {
            name: "voteResponse posts a downvote",
            call: () => api.voteResponse("resp-1", -1),
            transport: postMock,
            request: ["/responses/resp-1/vote", { value: -1 }],
        },
    ];

    runRequestCases(cases);
});
