import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./mystery";
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

describe("the mystery board API", () => {
    const mysteryPayload = {
        title: "The locked study",
        body: "Kinzo was found behind a bolted door",
        difficulty: "hard",
        free_for_all: false,
        keep_open_after_solve: true,
        knox_contract: {
            culprit_named_early: true,
            no_supernatural: true,
            passages_declared: true,
            no_unknown_poison: true,
            no_outsider: true,
            no_lucky_accident: true,
            detective_not_culprit: true,
            clues_shown: true,
            narrator_hides_nothing: true,
            no_unannounced_twins: true,
        },
        clues: [{ body: "the key was still inside", truth_type: "red" }],
    };

    const cases: RequestCase[] = [
        {
            name: "listMysteries defaults to a page of twenty",
            call: () => api.listMysteries({}),
            transport: fetchMock,
            request: ["/mysteries?limit=20"],
        },
        {
            name: "listMysteries passes the sort, the solved filter and the paging through",
            call: () => api.listMysteries({ sort: "top", solved: "false", limit: 5, offset: 10 }),
            transport: fetchMock,
            request: ["/mysteries?sort=top&solved=false&limit=5&offset=10"],
        },
        {
            name: "getMystery reads a single mystery",
            call: () => api.getMystery("m-1"),
            transport: fetchMock,
            request: ["/mysteries/m-1"],
        },
        {
            name: "createMystery posts the whole payload to the mystery collection",
            call: () => api.createMystery(mysteryPayload),
            transport: postMock,
            request: ["/mysteries", mysteryPayload],
        },
        {
            name: "updateMystery puts the whole payload back",
            call: () => api.updateMystery("m-1", mysteryPayload),
            transport: putMock,
            request: ["/mysteries/m-1", mysteryPayload],
        },
        {
            name: "deleteMystery deletes the mystery",
            call: () => api.deleteMystery("m-1"),
            transport: deleteMock,
            request: ["/mysteries/m-1"],
        },
    ];

    runRequestCases(cases);
});

describe("mystery attempts and votes", () => {
    const cases: RequestCase[] = [
        {
            name: "createMysteryAttempt leaves the parent unset for a top level attempt",
            call: () => api.createMysteryAttempt("m-1", "the butler did it"),
            transport: postMock,
            strict: true,
            request: ["/mysteries/m-1/attempts", { body: "the butler did it", parent_id: undefined }],
        },
        {
            name: "createMysteryAttempt carries the parent id of a reply",
            call: () => api.createMysteryAttempt("m-1", "no, the sister did", "at-0"),
            transport: postMock,
            strict: true,
            request: ["/mysteries/m-1/attempts", { body: "no, the sister did", parent_id: "at-0" }],
        },
        {
            name: "deleteMysteryAttempt deletes from the attempt collection",
            call: () => api.deleteMysteryAttempt("at-1"),
            transport: deleteMock,
            request: ["/mystery-attempts/at-1"],
        },
        {
            name: "voteMysteryAttempt posts an upvote",
            call: () => api.voteMysteryAttempt("at-1", 1),
            transport: postMock,
            request: ["/mystery-attempts/at-1/vote", { value: 1 }],
        },
        {
            name: "voteMysteryAttempt posts a downvote",
            call: () => api.voteMysteryAttempt("at-1", -1),
            transport: postMock,
            request: ["/mystery-attempts/at-1/vote", { value: -1 }],
        },
        {
            name: "markMysterySolved posts the winning attempt under the snake cased field",
            call: () => api.markMysterySolved("m-1", "at-1"),
            transport: postMock,
            request: ["/mysteries/m-1/solve", { attempt_id: "at-1" }],
        },
    ];

    runRequestCases(cases);
});

describe("mystery clues and game master controls", () => {
    const cases: RequestCase[] = [
        {
            name: "closeMystery posts an empty body to the close endpoint",
            call: () => api.closeMystery("m-1"),
            transport: postMock,
            request: ["/mysteries/m-1/close", {}],
        },
        {
            name: "setMysteryPaused posts the paused flag",
            call: () => api.setMysteryPaused("m-1", true),
            transport: postMock,
            request: ["/mysteries/m-1/pause", { paused: true }],
        },
        {
            name: "setMysteryPaused can resume the mystery again",
            call: () => api.setMysteryPaused("m-1", false),
            transport: postMock,
            request: ["/mysteries/m-1/pause", { paused: false }],
        },
        {
            name: "setMysteryGmAway posts the away flag",
            call: () => api.setMysteryGmAway("m-1", true),
            transport: postMock,
            request: ["/mysteries/m-1/away", { away: true }],
        },
        {
            name: "setMysteryGmAway can bring the game master back",
            call: () => api.setMysteryGmAway("m-1", false),
            transport: postMock,
            request: ["/mysteries/m-1/away", { away: false }],
        },
        {
            name: "deleteMysteryClue interpolates a numeric clue id",
            call: () => api.deleteMysteryClue("m-1", 4),
            transport: deleteMock,
            request: ["/mysteries/m-1/clues/4"],
        },
        {
            name: "updateMysteryClue puts only the body to the numbered clue",
            call: () => api.updateMysteryClue("m-1", 4, "the key was never inside"),
            transport: putMock,
            request: ["/mysteries/m-1/clues/4", { body: "the key was never inside" }],
        },
    ];

    runRequestCases(cases);
});

describe("mystery attachments and media", () => {
    const cases: RequestCase[] = [
        {
            name: "deleteMysteryAttachment interpolates a numeric attachment id",
            call: () => api.deleteMysteryAttachment("m-1", 4),
            transport: deleteMock,
            request: ["/mysteries/m-1/attachments/4"],
        },
        {
            name: "deleteMysteryMedia interpolates a numeric media id",
            call: () => api.deleteMysteryMedia("m-1", 7),
            transport: deleteMock,
            request: ["/mysteries/m-1/media/7"],
        },
    ];

    runRequestCases(cases);

    it("uploadMysteryAttachment posts the document under the file field of the mystery", async () => {
        // given
        const file = new File(["x"], "floor-plan.pdf", { type: "application/pdf" });

        // when
        await api.uploadMysteryAttachment("m-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/mysteries/m-1/attachments");
        expect(postFormDataMock.mock.calls[0][1].get("file")).toBe(file);
        expect(postFormDataMock.mock.calls[0][1].has("media")).toBe(false);
    });

    it("uploadMysteryMedia posts the image under the media field of the mystery", async () => {
        // given
        const file = new File(["x"], "mystery.png", { type: "image/png" });

        // when
        await api.uploadMysteryMedia("m-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/mysteries/m-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
        expect(postFormDataMock.mock.calls[0][1].has("file")).toBe(false);
    });
});

describe("mystery leaderboards", () => {
    const cases: RequestCase[] = [
        {
            name: "getMysteryLeaderboard sends no query string without a limit",
            call: () => api.getMysteryLeaderboard(),
            transport: fetchMock,
            request: ["/mysteries/leaderboard"],
        },
        {
            name: "getMysteryLeaderboard forwards the limit",
            call: () => api.getMysteryLeaderboard(10),
            transport: fetchMock,
            request: ["/mysteries/leaderboard?limit=10"],
        },
        {
            name: "getGMLeaderboard sends no query string without a limit",
            call: () => api.getGMLeaderboard(),
            transport: fetchMock,
            request: ["/mysteries/gm-leaderboard"],
        },
        {
            name: "getGMLeaderboard forwards the limit",
            call: () => api.getGMLeaderboard(5),
            transport: fetchMock,
            request: ["/mysteries/gm-leaderboard?limit=5"],
        },
    ];

    runRequestCases(cases);
});
