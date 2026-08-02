import { act, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WSMessage } from "../../types/api";
import type { WSMessageHandler } from "../../context/notificationContextValue";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { GameForfeitWarning } from "./GameForfeitWarning";

let listeners: WSMessageHandler[] = [];

function addWSListener(handler: WSMessageHandler): () => void {
    listeners.push(handler);
    return () => {
        listeners = listeners.filter(existing => existing !== handler);
    };
}

function emit(msg: WSMessage): void {
    act(() => {
        for (const listener of [...listeners]) {
            listener(msg);
        }
    });
}

const player = makeUser({ id: "user-1", username: "battler", display_name: "Battler" });

function renderWarning(user = player) {
    return renderWithProviders(<GameForfeitWarning />, { user, notification: { addWSListener } });
}

function warning(overrides: Record<string, unknown> = {}): WSMessage {
    return {
        type: "game_forfeit_warning",
        data: {
            room_id: "room-7",
            game_type: "chess",
            disconnected_at: "2026-02-01T12:00:00Z",
            grace_seconds: 8,
            ...overrides,
        },
    };
}

describe("GameForfeitWarning", () => {
    beforeEach(() => {
        listeners = [];
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-02-01T12:00:00Z"));
    });

    it("shows nothing until a forfeit warning arrives", () => {
        // given
        const noMessages: WSMessage[] = [];

        // when
        const { container } = renderWarning();
        for (const msg of noMessages) {
            emit(msg);
        }

        // then
        expect(listeners).toHaveLength(1);
        expect(container).toBeEmptyDOMElement();
    });

    it("counts down once the forfeit is inside the warning window", () => {
        // given
        renderWarning();

        // when
        emit(warning({ grace_seconds: 8 }));

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Forfeiting your chess game in 8s.");
        expect(screen.getByRole("link", { name: "Return to game" })).toHaveAttribute("href", "/games/chess/room-7");
    });

    it("stays quiet while the forfeit is still far away and speaks up as it approaches", () => {
        // given
        const { container } = renderWarning();
        emit(warning({ grace_seconds: 60 }));
        expect(container).toBeEmptyDOMElement();

        // when
        act(() => {
            vi.advanceTimersByTime(51_000);
        });

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Forfeiting your chess game in 9s.");
    });

    it("names the game type the warning came with", () => {
        // given
        renderWarning();

        // when
        emit(warning({ game_type: "checkers" }));

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Forfeiting your checkers game in 8s.");
    });

    it("assumes chess when the warning does not say which game it is", () => {
        // given
        renderWarning();

        // when
        emit(warning({ game_type: undefined }));

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Forfeiting your chess game in 8s.");
    });

    it("ignores a warning that names no room", () => {
        // given
        const { container } = renderWarning();

        // when
        emit(warning({ room_id: undefined }));

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("ignores a warning with no grace period", () => {
        // given
        const { container } = renderWarning();

        // when
        emit(warning({ grace_seconds: undefined }));

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("ignores a warning whose disconnect time cannot be read", () => {
        // given
        const { container } = renderWarning();

        // when
        emit(warning({ disconnected_at: "the golden land" }));

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("drops the countdown when the warning is withdrawn for that room", () => {
        // given
        const { container } = renderWarning();
        emit(warning());
        expect(screen.getByRole("status")).toBeInTheDocument();

        // when
        emit({ type: "game_forfeit_cleared", data: { room_id: "room-7" } });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("keeps the countdown when a different room is cleared", () => {
        // given
        renderWarning();
        emit(warning());

        // when
        emit({ type: "game_forfeit_cleared", data: { room_id: "room-99" } });

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Forfeiting your chess game in 8s.");
    });

    it("announces the forfeit when the signed in player is the one who abandoned the game", () => {
        // given
        renderWarning();
        emit(warning({ game_type: "checkers" }));

        // when
        emit({ type: "game_room_finished", data: { room_id: "room-7", abandoned_by: player.id } });

        // then
        expect(screen.getByRole("status")).toHaveTextContent("You forfeited the checkers game by disconnecting.");
        expect(screen.getByRole("link", { name: "View game" })).toHaveAttribute("href", "/games/checkers/room-7");
    });

    it("assumes chess when the finished game was never warned about", () => {
        // given
        renderWarning();

        // when
        emit({ type: "game_room_finished", data: { room_id: "room-3", abandoned_by: player.id } });

        // then
        expect(screen.getByRole("link", { name: "View game" })).toHaveAttribute("href", "/games/chess/room-3");
    });

    it("stays quiet when somebody else abandoned the game", () => {
        // given
        const { container } = renderWarning();

        // when
        emit({ type: "game_room_finished", data: { room_id: "room-7", abandoned_by: "user-2" } });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays quiet when nobody is signed in", () => {
        // given
        const { container } = renderWithProviders(<GameForfeitWarning />, {
            user: null,
            notification: { addWSListener },
        });

        // when
        emit({ type: "game_room_finished", data: { room_id: "room-7", abandoned_by: "user-1" } });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("dismisses the forfeit notice once its time is up", () => {
        // given
        const { container } = renderWarning();
        emit({ type: "game_room_finished", data: { room_id: "room-7", abandoned_by: player.id } });
        expect(screen.getByRole("status")).toBeInTheDocument();

        // when
        act(() => {
            vi.advanceTimersByTime(10_000);
        });

        // then
        expect(container).toBeEmptyDOMElement();
    });
});
