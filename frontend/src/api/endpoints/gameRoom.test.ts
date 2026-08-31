import { beforeEach, describe, vi } from "vitest";
import * as api from "./gameRoom";
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

describe("the game room API", () => {
    const cases: RequestCase[] = [
        {
            name: "inviteToGame posts the opponent and the snake cased game type",
            call: () => api.inviteToGame("u-1", "chess"),
            transport: postMock,
            request: ["/game-rooms", { opponent_id: "u-1", game_type: "chess" }],
        },
        {
            name: "getGameRoom reads a single room",
            call: () => api.getGameRoom("g-1"),
            transport: fetchMock,
            request: ["/game-rooms/g-1"],
        },
        {
            name: "acceptGameInvite posts an empty body to the accept",
            call: () => api.acceptGameInvite("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/accept", {}],
        },
        {
            name: "declineGameInvite posts an empty body to the decline",
            call: () => api.declineGameInvite("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/decline", {}],
        },
        {
            name: "cancelGameInvite posts an empty body to the cancel",
            call: () => api.cancelGameInvite("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/cancel", {}],
        },
        {
            name: "submitGameAction wraps the move in an action envelope",
            call: () => api.submitGameAction("g-1", { type: "move", from: "e2", to: "e4" }),
            transport: postMock,
            request: ["/game-rooms/g-1/action", { action: { type: "move", from: "e2", to: "e4" } }],
        },
        {
            name: "resignGame posts an empty body to the resignation",
            call: () => api.resignGame("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/resign", {}],
        },
        {
            name: "offerDraw posts an empty body to the draw offer",
            call: () => api.offerDraw("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/offer-draw", {}],
        },
        {
            name: "acceptDraw posts an empty body to the draw acceptance",
            call: () => api.acceptDraw("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/accept-draw", {}],
        },
        {
            name: "declineDraw posts an empty body to the draw refusal",
            call: () => api.declineDraw("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/decline-draw", {}],
        },
        {
            name: "getGameScoreboard nests the scoreboard under the game type",
            call: () => api.getGameScoreboard("chess"),
            transport: fetchMock,
            request: ["/games/chess/scoreboard"],
        },
        {
            name: "getGameScoreboard leaves an underscored game type alone",
            call: () => api.getGameScoreboard("snakes_and_ladders"),
            transport: fetchMock,
            request: ["/games/snakes_and_ladders/scoreboard"],
        },
        {
            name: "listLiveGameRooms sends no query string when no game type is given",
            call: () => api.listLiveGameRooms(),
            transport: fetchMock,
            request: ["/game-rooms/live"],
        },
        {
            name: "listLiveGameRooms filters the live rooms by game type",
            call: () => api.listLiveGameRooms("othello"),
            transport: fetchMock,
            request: ["/game-rooms/live?game_type=othello"],
        },
    ];

    runRequestCases(cases);
});

describe("spectator and player chat", () => {
    const cases: RequestCase[] = [
        {
            name: "getSpectatorChat reads the spectator messages of a room",
            call: () => api.getSpectatorChat("g-1"),
            transport: fetchMock,
            request: ["/game-rooms/g-1/chat"],
        },
        {
            name: "postSpectatorChat posts the body to the spectator chat",
            call: () => api.postSpectatorChat("g-1", "what a move"),
            transport: postMock,
            request: ["/game-rooms/g-1/chat", { body: "what a move" }],
        },
        {
            name: "getPlayerChat reads the private player messages of a room",
            call: () => api.getPlayerChat("g-1"),
            transport: fetchMock,
            request: ["/game-rooms/g-1/player-chat"],
        },
        {
            name: "postPlayerChat posts the body to the player chat",
            call: () => api.postPlayerChat("g-1", "good game"),
            transport: postMock,
            request: ["/game-rooms/g-1/player-chat", { body: "good game" }],
        },
    ];

    runRequestCases(cases);
});
