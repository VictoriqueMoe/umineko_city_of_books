import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./oc";
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

describe("the original character API", () => {
    const ocPayload = {
        name: "Victorique",
        description: "the golden fairy of the library",
        series: "custom",
        custom_series_name: "Gosick",
    };

    const cases: RequestCase[] = [
        {
            name: "getOC reads a single original character",
            call: () => api.getOC("oc-1"),
            transport: fetchMock,
            request: ["/ocs/oc-1"],
        },
        {
            name: "createOC posts the whole payload to the character collection",
            call: () => api.createOC(ocPayload),
            transport: postMock,
            request: ["/ocs", ocPayload],
        },
        {
            name: "updateOC puts the whole payload back under its own id",
            call: () => api.updateOC("oc-1", ocPayload),
            transport: putMock,
            request: ["/ocs/oc-1", ocPayload],
        },
        {
            name: "deleteOC deletes the character",
            call: () => api.deleteOC("oc-1"),
            transport: deleteMock,
            request: ["/ocs/oc-1"],
        },
        {
            name: "deleteOCGalleryImage interpolates a numeric image id",
            call: () => api.deleteOCGalleryImage("oc-1", 4),
            transport: deleteMock,
            request: ["/ocs/oc-1/gallery/4"],
        },
    ];

    runRequestCases(cases);

    it("uploadOCImage posts the file under the image field", async () => {
        // given
        const file = new File(["x"], "oc.png", { type: "image/png" });

        // when
        await api.uploadOCImage("oc-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/ocs/oc-1/image");
        expect(formData.get("image")).toBe(file);
    });
});

describe("original character votes and favourites", () => {
    const cases: RequestCase[] = [
        {
            name: "voteOC posts an upvote",
            call: () => api.voteOC("oc-1", 1),
            transport: postMock,
            request: ["/ocs/oc-1/vote", { value: 1 }],
        },
        {
            name: "voteOC posts a downvote",
            call: () => api.voteOC("oc-1", -1),
            transport: postMock,
            request: ["/ocs/oc-1/vote", { value: -1 }],
        },
        {
            name: "favouriteOC posts an empty body to the favourite",
            call: () => api.favouriteOC("oc-1"),
            transport: postMock,
            request: ["/ocs/oc-1/favourite", {}],
        },
    ];

    runRequestCases(cases);
});
