import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { GameRoom, GameRoomPlayer } from "../../types/api";
import { GamePlayerBar } from "./GamePlayerBar";

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "player-0";
    return {
        user_id: id,
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        role: "player",
        slot: 0,
        joined: true,
        connected: true,
        user: { id, username: "battler", display_name: "Battler" },
        ...overrides,
    };
}

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return {
        id: "room-1",
        game_type: "chess",
        status: "active",
        state: {},
        created_by: "player-0",
        created_at: "2026-08-02T11:58:00.000Z",
        updated_at: "2026-08-02T11:58:00.000Z",
        players: [
            makePlayer({ user_id: "a", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "b", slot: 1, display_name: "Beatrice" }),
        ],
        watcher_count: 0,
        ...overrides,
    };
}

describe("GamePlayerBar", () => {
    it("puts the second slot on the left and the first slot on the right", () => {
        // given
        const room = makeRoom();

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        const left = screen.getByText("Beatrice");
        const right = screen.getByText("Battler");
        expect(left.compareDocumentPosition(right) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    });

    it("labels each side with its colour", () => {
        // given
        const room = makeRoom();

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        expect(screen.getByText("(White)")).toBeInTheDocument();
        expect(screen.getByText("(Black)")).toBeInTheDocument();
    });

    it("falls back to the colour name when a seat is still empty", () => {
        // given
        const room = makeRoom({ players: [] });

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        expect(screen.getByText("White")).toBeInTheDocument();
        expect(screen.getByText("Black")).toBeInTheDocument();
    });

    it("marks the first slot as to move on the right hand side", () => {
        // given
        const room = makeRoom({ turn_user_id: "a" });

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        const markers = screen.getAllByText("to move");
        expect(markers).toHaveLength(1);
        const centre = screen.getByTitle("Spectators watching");
        expect(centre.compareDocumentPosition(markers[0]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    });

    it("marks the second slot as to move on the left hand side", () => {
        // given
        const room = makeRoom({ turn_user_id: "b" });

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        const markers = screen.getAllByText("to move");
        expect(markers).toHaveLength(1);
        const centre = screen.getByTitle("Spectators watching");
        expect(centre.compareDocumentPosition(markers[0]) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy();
    });

    it("drops the turn marker once the game is no longer active", () => {
        // given
        const room = makeRoom({ status: "finished", turn_user_id: "a" });

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        expect(screen.queryAllByText("to move")).toHaveLength(0);
    });

    it("only marks the seated player while the other seat is still empty", () => {
        // given
        const room = makeRoom({
            players: [makePlayer({ user_id: "a", slot: 0, display_name: "Battler" })],
            turn_user_id: "a",
        });

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        expect(screen.getAllByText("to move")).toHaveLength(1);
        expect(screen.getByText("Black")).toBeInTheDocument();
    });

    it("marks nobody as to move while an active room has no turn and no players", () => {
        // given
        const room = makeRoom({ players: [] });

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        expect(screen.queryAllByText("to move")).toHaveLength(0);
    });

    it("counts the spectators watching", () => {
        // given
        const room = makeRoom({ watcher_count: 7 });

        // when
        renderWithProviders(
            <GamePlayerBar room={room} slot0Label="White" slot1Label="Black" liveDurationSeconds={0} />,
        );

        // then
        expect(screen.getByTitle("Spectators watching")).toHaveTextContent("7");
    });

    it("shows how long the game has been running", () => {
        // given
        const liveDurationSeconds = 125;

        // when
        renderWithProviders(
            <GamePlayerBar
                room={makeRoom()}
                slot0Label="White"
                slot1Label="Black"
                liveDurationSeconds={liveDurationSeconds}
            />,
        );

        // then
        expect(screen.getByTitle("Game duration")).toHaveTextContent("2m 5s");
    });

    it("shows a dash before the clock has started", () => {
        // given
        const liveDurationSeconds = 0;

        // when
        renderWithProviders(
            <GamePlayerBar
                room={makeRoom()}
                slot0Label="White"
                slot1Label="Black"
                liveDurationSeconds={liveDurationSeconds}
            />,
        );

        // then
        expect(screen.getByTitle("Game duration")).toHaveTextContent("-");
    });
});
