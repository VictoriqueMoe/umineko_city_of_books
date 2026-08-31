import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./post";
import {
    deleteMock,
    fetchMock,
    postFormDataMock,
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

describe("the post API", () => {
    const cases: RequestCase[] = [
        {
            name: "getCornerCounts reads the corner counters",
            call: () => api.getCornerCounts(),
            transport: fetchMock,
            request: ["/posts/corner-counts"],
        },
        {
            name: "getPost reads a single post",
            call: () => api.getPost("p-1"),
            transport: fetchMock,
            request: ["/posts/p-1"],
        },
        {
            name: "updatePost puts only the body",
            call: () => api.updatePost("p-1", "the gold is real"),
            transport: putMock,
            request: ["/posts/p-1", { body: "the gold is real" }],
        },
        {
            name: "votePoll posts the snake cased option id",
            call: () => api.votePoll("p-1", 2),
            transport: postMock,
            request: ["/posts/p-1/poll/vote", { option_id: 2 }],
        },
        {
            name: "unresolveSuggestion deletes the resolution",
            call: () => api.unresolveSuggestion("p-1"),
            transport: deleteMock,
            request: ["/posts/p-1/resolve"],
        },
        {
            name: "deletePost deletes the post",
            call: () => api.deletePost("p-1"),
            transport: deleteMock,
            request: ["/posts/p-1"],
        },
        {
            name: "deletePostMedia interpolates a numeric media id",
            call: () => api.deletePostMedia("p-1", 4),
            transport: deleteMock,
            request: ["/posts/p-1/media/4"],
        },
        {
            name: "likePost posts with no body",
            call: () => api.likePost("p-1"),
            transport: postMock,
            request: ["/posts/p-1/like", undefined],
        },
        {
            name: "unlikePost deletes the like",
            call: () => api.unlikePost("p-1"),
            transport: deleteMock,
            request: ["/posts/p-1/like"],
        },
    ];

    runRequestCases(cases);
});

describe("post media uploads", () => {
    it("uploadPostMedia posts the file under the media field of the post", async () => {
        // given
        const file = new File(["x"], "post.png", { type: "image/png" });

        // when
        await api.uploadPostMedia("p-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/posts/p-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
    });
});
