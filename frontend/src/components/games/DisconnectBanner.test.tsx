import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { GameRoomPlayer } from "../../types/api";
import { DisconnectBanner } from "./DisconnectBanner";

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
        connected: false,
        disconnected_at: "2026-08-02T11:59:50.000Z",
        user: { id, username: "battler", display_name: "Battler" },
        ...overrides,
    };
}

describe("DisconnectBanner", () => {
    it("renders nothing while everyone is still connected", () => {
        // given
        const offlinePlayer = undefined;

        // when
        const { container } = renderWithProviders(
            <DisconnectBanner offlinePlayer={offlinePlayer} forfeitRemaining={30} />,
        );

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("renders nothing when there is no countdown to show", () => {
        // given
        const forfeitRemaining = null;

        // when
        const { container } = renderWithProviders(
            <DisconnectBanner offlinePlayer={makePlayer()} forfeitRemaining={forfeitRemaining} />,
        );

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("names the player who dropped out and how long they have left", () => {
        // given
        const offlinePlayer = makePlayer({ display_name: "Beatrice" });

        // when
        renderWithProviders(<DisconnectBanner offlinePlayer={offlinePlayer} forfeitRemaining={42} />);

        // then
        expect(screen.getByText("Beatrice disconnected - forfeits in 42s")).toBeInTheDocument();
    });

    it("keeps showing the banner at the moment the countdown reaches zero", () => {
        // given
        const forfeitRemaining = 0;

        // when
        renderWithProviders(
            <DisconnectBanner
                offlinePlayer={makePlayer({ display_name: "Battler" })}
                forfeitRemaining={forfeitRemaining}
            />,
        );

        // then
        expect(screen.getByText("Battler disconnected - forfeits in 0s")).toBeInTheDocument();
    });
});
