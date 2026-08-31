import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./ship";
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

describe("the ship API", () => {
    const shipPayload = {
        title: "Beato and Battler",
        description: "the golden witch and her opponent",
        characters: [
            { series: "umineko", character_name: "Beatrice", sort_order: 0 },
            { series: "umineko", character_id: "c-2", character_name: "Battler", sort_order: 1 },
        ],
    };

    const cases: RequestCase[] = [
        {
            name: "getShip reads a single ship",
            call: () => api.getShip("s-1"),
            transport: fetchMock,
            request: ["/ships/s-1"],
        },
        {
            name: "createShip posts the whole payload to the ship collection",
            call: () => api.createShip(shipPayload),
            transport: postMock,
            request: ["/ships", shipPayload],
        },
        {
            name: "updateShip puts the whole payload back under its own id",
            call: () => api.updateShip("s-1", shipPayload),
            transport: putMock,
            request: ["/ships/s-1", shipPayload],
        },
        {
            name: "deleteShip deletes the ship",
            call: () => api.deleteShip("s-1"),
            transport: deleteMock,
            request: ["/ships/s-1"],
        },
        {
            name: "voteShip posts an upvote",
            call: () => api.voteShip("s-1", 1),
            transport: postMock,
            request: ["/ships/s-1/vote", { value: 1 }],
        },
        {
            name: "voteShip posts a downvote",
            call: () => api.voteShip("s-1", -1),
            transport: postMock,
            request: ["/ships/s-1/vote", { value: -1 }],
        },
    ];

    runRequestCases(cases);
});

describe("ship image uploads", () => {
    it("uploadShipImage posts the file under the image field", async () => {
        // given
        const file = new File(["x"], "ship.png", { type: "image/png" });

        // when
        await api.uploadShipImage("s-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/ships/s-1/image");
        expect(formData.get("image")).toBe(file);
    });
});
