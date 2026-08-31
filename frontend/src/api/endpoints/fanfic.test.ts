import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./fanfic";
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

describe("the fanfic API", () => {
    const oneshotPayload = {
        title: "The Golden Land",
        summary: "Beatrice pours the tea one last time",
        series: "umineko",
        rating: "teen",
        language: "English",
        is_oneshot: true,
        contains_lemons: false,
        genres: ["mystery"],
        tags: ["beatrice"],
        characters: [{ series: "umineko", character_name: "Beatrice", sort_order: 0 }],
        is_pairing: false,
    };
    const updatePayload = {
        ...oneshotPayload,
        status: "complete",
    };

    const cases: RequestCase[] = [
        {
            name: "getFanfic reads a single fanfic",
            call: () => api.getFanfic("f-1"),
            transport: fetchMock,
            request: ["/fanfics/f-1"],
        },
        {
            name: "createFanfic leaves the status and body out when they are not supplied",
            call: () => api.createFanfic(oneshotPayload),
            transport: postMock,
            request: ["/fanfics", oneshotPayload],
        },
        {
            name: "updateFanfic puts the whole payload back",
            call: () => api.updateFanfic("f-1", updatePayload),
            transport: putMock,
            request: ["/fanfics/f-1", updatePayload],
        },
        {
            name: "deleteFanfic deletes the fanfic",
            call: () => api.deleteFanfic("f-1"),
            transport: deleteMock,
            request: ["/fanfics/f-1"],
        },
        {
            name: "deleteFanficCover deletes the cover image",
            call: () => api.deleteFanficCover("f-1"),
            transport: deleteMock,
            request: ["/fanfics/f-1/cover"],
        },
        {
            name: "createFanficChapter posts the title and body to the chapter collection",
            call: () => api.createFanficChapter("f-1", "The first twilight", "Six chosen by the key"),
            transport: postMock,
            request: ["/fanfics/f-1/chapters", { title: "The first twilight", body: "Six chosen by the key" }],
        },
        {
            name: "updateFanficChapter puts the title and body under the chapter's own id",
            call: () => api.updateFanficChapter("ch-1", "The second twilight", "The two who are close"),
            transport: putMock,
            request: ["/fanfic-chapters/ch-1", { title: "The second twilight", body: "The two who are close" }],
        },
        {
            name: "deleteFanficChapter deletes from the chapter collection",
            call: () => api.deleteFanficChapter("ch-1"),
            transport: deleteMock,
            request: ["/fanfic-chapters/ch-1"],
        },
        {
            name: "favouriteFanfic posts an empty body to the favourite",
            call: () => api.favouriteFanfic("f-1"),
            transport: postMock,
            request: ["/fanfics/f-1/favourite", {}],
        },
        {
            name: "unfavouriteFanfic deletes the favourite",
            call: () => api.unfavouriteFanfic("f-1"),
            transport: deleteMock,
            request: ["/fanfics/f-1/favourite"],
        },
    ];

    runRequestCases(cases);

    it("uploadFanficCover posts the file under the image field of the fanfic", async () => {
        // given
        const file = new File(["x"], "cover.png", { type: "image/png" });

        // when
        await api.uploadFanficCover("f-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/fanfics/f-1/cover");
        expect(postFormDataMock.mock.calls[0][1].get("image")).toBe(file);
    });
});

describe("fanfic series", () => {
    it("getFanficSeries returns just the series list", async () => {
        // given
        fetchMock.mockResolvedValue({ series: ["Umineko", "Higurashi"] });

        // when
        const result = await api.getFanficSeries();

        // then
        expect(fetchMock).toHaveBeenCalledWith("/fanfic-series");
        expect(result).toEqual(["Umineko", "Higurashi"]);
    });
});
