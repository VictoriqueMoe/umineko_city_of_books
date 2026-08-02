import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CharacterId } from "../../../games/minesweeper/types";
import { renderWithProviders } from "../../../test-utils/render";
import type { MinesweeperState } from "../../../types/api";
import { MinesweeperCharacterSelect } from "./MinesweeperCharacterSelect";

const onSelect = vi.fn<(character: CharacterId) => Promise<void>>();

function makeState(overrides: Partial<MinesweeperState> = {}): MinesweeperState {
    return {
        phase: "char_select",
        width: 9,
        height: 9,
        mine_count: 10,
        characters: ["", ""],
        revealed: [[], []],
        flagged: [[], []],
        revealed_count: [0, 0],
        values: [[], []],
        mines_placed: false,
        pending_clicks: [null, null],
        ...overrides,
    };
}

function renderSelect(state: MinesweeperState, mySlot: number | null, isSpectator = false, submitting = false) {
    return renderWithProviders(
        <MinesweeperCharacterSelect
            state={state}
            mySlot={mySlot}
            isSpectator={isSpectator}
            submitting={submitting}
            onSelect={onSelect}
        />,
    );
}

describe("MinesweeperCharacterSelect", () => {
    beforeEach(() => {
        onSelect.mockResolvedValue(undefined);
    });

    it("lines up every witch that can be fought as", () => {
        // given
        const state = makeState();

        // when
        renderSelect(state, 0);

        // then
        expect(screen.getByRole("button", { name: /Bernkastel/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Erika Furudo/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Dlanor A. Knox/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Lambdadelta/ })).toBeInTheDocument();
    });

    it("asks the player to choose before anyone has picked", () => {
        // given
        const state = makeState();

        // when
        renderSelect(state, 0);

        // then
        expect(screen.getByText("Choose a witch to fight as.")).toBeInTheDocument();
    });

    it("sends the witch the player picked", async () => {
        // given
        const user = userEvent.setup();
        renderSelect(makeState(), 0);

        // when
        await user.click(screen.getByRole("button", { name: /Erika Furudo/ }));

        // then
        expect(onSelect).toHaveBeenCalledWith(CharacterId.Erika);
    });

    it("marks the witch this player has committed to", () => {
        // given
        const state = makeState({ characters: [CharacterId.Bernkastel, ""] });

        // when
        renderSelect(state, 0);

        // then
        expect(screen.getByRole("button", { name: /Bernkastel/ })).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByText("· Chosen ·")).toBeInTheDocument();
        expect(screen.getByText("Your witch is chosen. Awaiting your opponent.")).toBeInTheDocument();
    });

    it("bars the witch the opponent already claimed", () => {
        // given
        const state = makeState({ characters: ["", CharacterId.Lambdadelta] });

        // when
        renderSelect(state, 0);

        // then
        expect(screen.getByRole("button", { name: /Lambdadelta/ })).toBeDisabled();
        expect(screen.getByText("Opposed")).toBeInTheDocument();
    });

    it("announces that the duel can begin once both have committed", () => {
        // given
        const state = makeState({ characters: [CharacterId.Bernkastel, CharacterId.Erika] });

        // when
        renderSelect(state, 0);

        // then
        expect(screen.getByText("Both have committed. The board awakens.")).toBeInTheDocument();
    });

    it("reads the second seat's pick when the player sits in it", () => {
        // given
        const state = makeState({ characters: [CharacterId.Bernkastel, CharacterId.Erika] });

        // when
        renderSelect(state, 1);

        // then
        expect(screen.getByRole("button", { name: /Erika Furudo/ })).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByRole("button", { name: /Bernkastel/ })).toBeDisabled();
    });

    it("lets a spectator watch the council without joining it", async () => {
        // given
        const user = userEvent.setup();
        renderSelect(makeState(), null, true);

        // when
        await user.click(screen.getByRole("button", { name: /Bernkastel/ }));

        // then
        expect(screen.getByText("Spectating the council.")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Bernkastel/ })).toBeDisabled();
        expect(onSelect).not.toHaveBeenCalled();
    });

    it("shuts the whole lineup while a pick is in flight", async () => {
        // given
        const user = userEvent.setup();
        let settle = () => {};
        onSelect.mockReturnValue(
            new Promise<void>(resolve => {
                settle = resolve;
            }),
        );
        renderSelect(makeState(), 0);

        // when
        await user.click(screen.getByRole("button", { name: /Bernkastel/ }));

        // then
        expect(screen.getByRole("button", { name: /Erika Furudo/ })).toBeDisabled();
        settle();
        await waitFor(() => {
            expect(screen.getByRole("button", { name: /Erika Furudo/ })).toBeEnabled();
        });
    });

    it("takes no new pick while the room is already busy", async () => {
        // given
        const user = userEvent.setup();
        renderSelect(makeState(), 0, false, true);

        // when
        await user.click(screen.getByRole("button", { name: /Bernkastel/ }));

        // then
        expect(onSelect).not.toHaveBeenCalled();
    });
});
