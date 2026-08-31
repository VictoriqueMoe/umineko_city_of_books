import { beforeEach, describe, vi } from "vitest";
import * as api from "./watchParty";
import { deleteMock, fetchMock, postMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

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

describe("watch party sessions", () => {
    const fullOptions = {
        start_url: "https://example.com/video",
        region: "EU",
        title: "the golden feast",
        type: "hyperbeam" as const,
    };

    const cases: RequestCase[] = [
        {
            name: "listWatchParties reads the sessions of a room",
            call: () => api.listWatchParties("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/watch-parties"],
        },
        {
            name: "startWatchParty posts an empty options object when nothing was chosen",
            call: () => api.startWatchParty("r-1", {}),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties", {}],
        },
        {
            name: "startWatchParty forwards the start url, region, title and type",
            call: () => api.startWatchParty("r-1", fullOptions),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties", fullOptions],
        },
        {
            name: "startWatchParty can start a screenshare with only a type",
            call: () => api.startWatchParty("r-1", { type: "screenshare" }),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties", { type: "screenshare" }],
        },
        {
            name: "joinWatchParty posts an empty body to the session join",
            call: () => api.joinWatchParty("r-1", "s-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/join", {}],
        },
        {
            name: "endWatchParty deletes the whole session",
            call: () => api.endWatchParty("r-1", "s-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1"],
        },
        {
            name: "identifyWatchPartyParticipant posts the identifier",
            call: () => api.identifyWatchPartyParticipant("r-1", "s-1", "peer-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/identify", { identifier: "peer-1" }],
        },
    ];

    runRequestCases(cases);
});
