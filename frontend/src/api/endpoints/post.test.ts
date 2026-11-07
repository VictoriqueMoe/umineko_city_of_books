import { beforeEach, describe, expect, vi } from "vitest";
import * as api from "./post";
import {
    deleteMock,
    fetchMock,
    postMock,
    putMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

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

describe("post media", () => {
    const media = [
        { upload_id: "upload-1", is_spoiler: true },
        { upload_id: "upload-2", is_spoiler: false },
    ];

    const cases: RequestCase[] = [
        {
            name: "attachPostMedia posts the finished upload with its spoiler flag to the post",
            call: () => api.attachPostMedia("p-1", media[0]),
            transport: postMock,
            request: ["/posts/p-1/media", media[0]],
        },
        {
            name: "createPost sends the post and its finished uploads in display order in one JSON request",
            call: () => api.createPost("look", "umineko", undefined, undefined, undefined, media),
            transport: postMock,
            request: ["/posts", expect.objectContaining({ body: "look", corner: "umineko", media })],
        },
        {
            name: "createPost sends an empty media list when nothing is attached",
            call: () => api.createPost("words only"),
            transport: postMock,
            request: ["/posts", expect.objectContaining({ body: "words only", corner: "general", media: [] })],
        },
    ];

    runRequestCases(cases);
});
