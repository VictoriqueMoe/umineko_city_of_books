import { describe, expect, it } from "vitest";
import { turnNotice, type TurnNoticeRequest } from "./turnNotice";

function request(overrides: Partial<TurnNoticeRequest> = {}): TurnNoticeRequest {
    return {
        watchedRoomId: "room-1",
        eventRoomId: "room-1",
        gameType: "chess",
        tabHidden: true,
        ...overrides,
    };
}

describe("turnNotice", () => {
    it("announces the turn when the watched room matches and the tab is hidden", () => {
        // given
        const input = request();

        // when
        const notice = turnNotice(input);

        // then
        expect(notice).toEqual({
            title: "Your move in chess",
            body: "It's your turn.",
            tag: "game-turn-room-1",
            route: "/games/chess/room-1",
        });
    });

    it("stays quiet while the tab is in front of the player", () => {
        // given
        const input = request({ tabHidden: false });

        // when
        const notice = turnNotice(input);

        // then
        expect(notice).toBeNull();
    });

    it("stays quiet about a turn in a room the player is not watching", () => {
        // given
        const input = request({ eventRoomId: "room-other" });

        // when
        const notice = turnNotice(input);

        // then
        expect(notice).toBeNull();
    });

    it("stays quiet when no room is being watched", () => {
        // given
        const input = request({ watchedRoomId: undefined, eventRoomId: undefined });

        // when
        const notice = turnNotice(input);

        // then
        expect(notice).toBeNull();
    });

    it("titles an unnamed game type as a game", () => {
        // given
        const input = request({ gameType: undefined });

        // when
        const notice = turnNotice(input);

        // then
        expect(notice?.title).toBe("Your move in game");
    });

    it("routes an unnamed game type to the chess board", () => {
        // given
        const input = request({ gameType: undefined });

        // when
        const notice = turnNotice(input);

        // then
        expect(notice?.route).toBe("/games/chess/room-1");
    });

    it("tags the notice by room so a second turn replaces the first", () => {
        // given
        const input = request({ watchedRoomId: "room-9", eventRoomId: "room-9", gameType: "othello" });

        // when
        const notice = turnNotice(input);

        // then
        expect(notice?.tag).toBe("game-turn-room-9");
        expect(notice?.route).toBe("/games/othello/room-9");
    });
});
