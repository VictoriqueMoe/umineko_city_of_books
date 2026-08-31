import { beforeEach, describe, vi } from "vitest";
import * as api from "./giphy";
import { fetchMock, postMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

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

describe("the giphy API", () => {
    const favourite = {
        giphy_id: "gif-1",
        url: "https://giphy.com/gifs/gif-1",
        title: "a golden butterfly",
        preview_url: "https://giphy.com/preview/gif-1",
        width: 320,
        height: 240,
    };

    const cases: RequestCase[] = [
        {
            name: "trendingGiphy sends no query string when no paging is asked for",
            call: () => api.trendingGiphy(),
            transport: fetchMock,
            request: ["/giphy/trending"],
        },
        {
            name: "trendingGiphy pages through the trending gifs",
            call: () => api.trendingGiphy(10, 25),
            transport: fetchMock,
            request: ["/giphy/trending?offset=10&limit=25"],
        },
        {
            name: "listGiphyFavourites sends no query string when no paging is asked for",
            call: () => api.listGiphyFavourites(),
            transport: fetchMock,
            request: ["/giphy/favourites"],
        },
        {
            name: "listGiphyFavourites pages through the saved gifs",
            call: () => api.listGiphyFavourites(10, 25),
            transport: fetchMock,
            request: ["/giphy/favourites?offset=10&limit=25"],
        },
        {
            name: "addGiphyFavourite posts the whole favourite to the collection",
            call: () => api.addGiphyFavourite(favourite),
            transport: postMock,
            request: ["/giphy/favourites", favourite],
        },
    ];

    runRequestCases(cases);
});
