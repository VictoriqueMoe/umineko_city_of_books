import { beforeEach, describe, vi } from "vitest";
import * as api from "./stream";
import {
    deleteMock,
    fetchMock,
    patchMock,
    postMock,
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

describe("the live stream API", () => {
    const cases: RequestCase[] = [
        {
            name: "listLiveStreams reads the live stream list",
            call: () => api.listLiveStreams(),
            transport: fetchMock,
            request: ["/streams/live"],
        },
        {
            name: "getStream reads a single stream",
            call: () => api.getStream("s-1"),
            transport: fetchMock,
            request: ["/streams/s-1"],
        },
        {
            name: "getMyStream reads the caller's own stream",
            call: () => api.getMyStream(),
            transport: fetchMock,
            request: ["/streams/mine"],
        },
        {
            name: "getStreamCredentials reads the ingest credentials",
            call: () => api.getStreamCredentials(),
            transport: fetchMock,
            request: ["/streams/credentials"],
        },
        {
            name: "resetStreamCredentials posts an empty body to the credential reset",
            call: () => api.resetStreamCredentials(),
            transport: postMock,
            request: ["/streams/credentials/reset", {}],
        },
        {
            name: "startStream posts the title, the default mode and the bitrate",
            call: () => api.startStream("the golden feast", "webrtc", 4500),
            transport: postMock,
            request: ["/streams", { title: "the golden feast", defaultMode: "webrtc", bitrate: 4500 }],
        },
        {
            name: "startStream can favour hls as the default mode",
            call: () => api.startStream("the golden feast", "hls", 6000),
            transport: postMock,
            request: ["/streams", { title: "the golden feast", defaultMode: "hls", bitrate: 6000 }],
        },
        {
            name: "stopStream deletes the stream",
            call: () => api.stopStream("s-1"),
            transport: deleteMock,
            request: ["/streams/s-1"],
        },
        {
            name: "updateStreamTitle patches only the title",
            call: () => api.updateStreamTitle("s-1", "the seventh twilight"),
            transport: patchMock,
            request: ["/streams/s-1", { title: "the seventh twilight" }],
        },
        {
            name: "getStreamViewerToken posts an empty body for a viewer token",
            call: () => api.getStreamViewerToken("s-1"),
            transport: postMock,
            request: ["/streams/s-1/token", {}],
        },
        {
            name: "joinStreamChat posts an empty body to the join chat endpoint",
            call: () => api.joinStreamChat("s-1"),
            transport: postMock,
            request: ["/streams/s-1/join-chat", {}],
        },
    ];

    runRequestCases(cases);
});
