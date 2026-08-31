import { beforeEach, describe, vi } from "vitest";
import * as api from "./art";
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

describe("the art API", () => {
    const artPayload = {
        title: "Golden Witch",
        description: "the endless witch in her rose garden",
        tags: ["beatrice", "umineko"],
        is_spoiler: true,
    };

    const cases: RequestCase[] = [
        {
            name: "listArt sends no query string when nothing is filtered",
            call: () => api.listArt({}),
            transport: fetchMock,
            request: ["/art"],
        },
        {
            name: "listArt passes every filter through",
            call: () =>
                api.listArt({
                    corner: "umineko",
                    type: "digital",
                    search: "golden witch",
                    tag: "beatrice",
                    sort: "top",
                    limit: 10,
                    offset: 20,
                }),
            transport: fetchMock,
            request: ["/art?corner=umineko&type=digital&search=golden+witch&tag=beatrice&sort=top&limit=10&offset=20"],
        },
        {
            name: "getArt reads a single piece",
            call: () => api.getArt("a-1"),
            transport: fetchMock,
            request: ["/art/a-1"],
        },
        {
            name: "updateArt puts the whole payload back",
            call: () => api.updateArt("a-1", artPayload),
            transport: putMock,
            request: ["/art/a-1", artPayload],
        },
        {
            name: "deleteArt deletes the piece",
            call: () => api.deleteArt("a-1"),
            transport: deleteMock,
            request: ["/art/a-1"],
        },
        {
            name: "likeArt posts with no body",
            call: () => api.likeArt("a-1"),
            transport: postMock,
            request: ["/art/a-1/like", undefined],
        },
    ];

    runRequestCases(cases);
});

describe("art likes and corner counts", () => {
    const cases: RequestCase[] = [
        {
            name: "unlikeArt deletes the like",
            call: () => api.unlikeArt("a-1"),
            transport: deleteMock,
            request: ["/art/a-1/like"],
        },
        {
            name: "getArtCornerCounts reads the art corner counters",
            call: () => api.getArtCornerCounts(),
            transport: fetchMock,
            request: ["/art/corner-counts"],
        },
    ];

    runRequestCases(cases);
});

describe("the gallery API", () => {
    const cases: RequestCase[] = [
        {
            name: "updateGallery defaults the description to an empty string",
            call: () => api.updateGallery("g-1", "Witches"),
            transport: putMock,
            request: ["/galleries/g-1", { name: "Witches", description: "" }],
        },
        {
            name: "updateGallery forwards the description when one was written",
            call: () => api.updateGallery("g-1", "Witches", "the endless witch and her furniture"),
            transport: putMock,
            request: ["/galleries/g-1", { name: "Witches", description: "the endless witch and her furniture" }],
        },
        {
            name: "setGalleryCover puts the chosen cover under the snake cased field",
            call: () => api.setGalleryCover("g-1", "a-1"),
            transport: putMock,
            request: ["/galleries/g-1/cover", { cover_art_id: "a-1" }],
        },
        {
            name: "setGalleryCover clears the cover with an explicit null",
            call: () => api.setGalleryCover("g-1", null),
            transport: putMock,
            request: ["/galleries/g-1/cover", { cover_art_id: null }],
        },
        {
            name: "deleteGallery deletes the gallery",
            call: () => api.deleteGallery("g-1"),
            transport: deleteMock,
            request: ["/galleries/g-1"],
        },
        {
            name: "getGallery defaults to the first page of twenty four",
            call: () => api.getGallery("g-1"),
            transport: fetchMock,
            request: ["/galleries/g-1?limit=24"],
        },
        {
            name: "getGallery pages through the gallery",
            call: () => api.getGallery("g-1", 10, 20),
            transport: fetchMock,
            request: ["/galleries/g-1?limit=10&offset=20"],
        },
        {
            name: "listAllGalleries sends no query string without a corner",
            call: () => api.listAllGalleries(),
            transport: fetchMock,
            request: ["/galleries"],
        },
        {
            name: "listAllGalleries encodes the corner",
            call: () => api.listAllGalleries("umineko & co"),
            transport: fetchMock,
            request: ["/galleries?corner=umineko%20%26%20co"],
        },
        {
            name: "setArtGallery puts the chosen gallery under the snake cased field",
            call: () => api.setArtGallery("a-1", "g-1"),
            transport: putMock,
            request: ["/art/a-1/gallery", { gallery_id: "g-1" }],
        },
        {
            name: "setArtGallery removes the piece from its gallery with an explicit null",
            call: () => api.setArtGallery("a-1", null),
            transport: putMock,
            request: ["/art/a-1/gallery", { gallery_id: null }],
        },
    ];

    runRequestCases(cases);
});
