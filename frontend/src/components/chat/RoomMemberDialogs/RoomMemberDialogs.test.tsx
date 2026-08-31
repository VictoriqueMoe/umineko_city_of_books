import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ChatRoomMember } from "../../../types/api";
import { makePublicUser, makeRoomMember } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import {
    RoomMemberDialogs,
    type NicknameDialogProps,
    type RoomMemberDialogsVariant,
    type TimeoutDialogProps,
} from "./RoomMemberDialogs";

function makeMember(displayName: string): ChatRoomMember {
    return makeRoomMember({ user: makePublicUser({ id: "member-1", display_name: displayName }) });
}

function makeNickname(overrides: Partial<NicknameDialogProps> = {}): NicknameDialogProps {
    return {
        target: null,
        value: "",
        error: "",
        saving: false,
        onValueChange: vi.fn(),
        onCancel: vi.fn(),
        onSave: vi.fn(),
        ...overrides,
    };
}

function makeTimeout(overrides: Partial<TimeoutDialogProps> = {}): TimeoutDialogProps {
    return {
        target: null,
        amount: "10",
        unit: "seconds",
        error: "",
        saving: false,
        onAmountChange: vi.fn(),
        onUnitChange: vi.fn(),
        onCancel: vi.fn(),
        onSave: vi.fn(),
        ...overrides,
    };
}

function renderDialogs(options: {
    variant?: RoomMemberDialogsVariant;
    nickname?: NicknameDialogProps;
    timeout?: TimeoutDialogProps;
}) {
    return renderWithProviders(
        <RoomMemberDialogs
            variant={options.variant ?? "desktop"}
            nickname={options.nickname ?? makeNickname()}
            timeout={options.timeout ?? makeTimeout()}
        />,
    );
}

describe("RoomMemberDialogs nickname dialog", () => {
    it("stays shut until a target member is chosen", () => {
        // given
        const nickname = makeNickname({ target: null });

        // when
        renderDialogs({ nickname });

        // then
        expect(screen.queryByRole("heading", { name: /Change nickname for/ })).not.toBeInTheDocument();
    });

    it("names the member it is renaming", () => {
        // given
        const nickname = makeNickname({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ nickname });

        // then
        expect(screen.getByRole("heading", { name: "Change nickname for Beatrice" })).toBeInTheDocument();
    });

    it("seeds the field with the current nickname and reports every edit", async () => {
        // given
        const user = userEvent.setup();
        const nickname = makeNickname({ target: makeMember("Beatrice"), value: "Bea" });

        // when
        renderDialogs({ nickname });
        const field = screen.getByPlaceholderText("Nickname (leave blank to clear)");
        await user.type(field, "!");

        // then
        expect(field).toHaveValue("Bea");
        expect(nickname.onValueChange).toHaveBeenCalledWith("Bea!");
    });

    it("caps the nickname at 32 characters", () => {
        // given
        const nickname = makeNickname({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ nickname });

        // then
        expect(screen.getByPlaceholderText("Nickname (leave blank to clear)")).toHaveAttribute("maxlength", "32");
    });

    it("saves when Save is pressed", async () => {
        // given
        const user = userEvent.setup();
        const nickname = makeNickname({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ nickname });
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(nickname.onSave).toHaveBeenCalledTimes(1);
    });

    it("cancels when Cancel is pressed", async () => {
        // given
        const user = userEvent.setup();
        const nickname = makeNickname({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ nickname });
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(nickname.onCancel).toHaveBeenCalledTimes(1);
    });

    it("cancels when the backdrop is clicked", async () => {
        // given
        const user = userEvent.setup();
        const nickname = makeNickname({ target: makeMember("Beatrice") });

        // when
        const { container } = renderDialogs({ nickname });
        await user.click(container.firstElementChild as HTMLElement);

        // then
        expect(nickname.onCancel).toHaveBeenCalledTimes(1);
    });

    it("keeps the dialog open when the dialog body itself is clicked", async () => {
        // given
        const user = userEvent.setup();
        const nickname = makeNickname({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ nickname });
        await user.click(screen.getByRole("heading", { name: "Change nickname for Beatrice" }));

        // then
        expect(nickname.onCancel).not.toHaveBeenCalled();
    });

    it("locks both buttons while the save is in flight", () => {
        // given
        const nickname = makeNickname({ target: makeMember("Beatrice"), saving: true });

        // when
        renderDialogs({ nickname });

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    });

    it("shows the error the save came back with", () => {
        // given
        const nickname = makeNickname({ target: makeMember("Beatrice"), error: "That nickname is taken" });

        // when
        renderDialogs({ nickname });

        // then
        expect(screen.getByText("That nickname is taken")).toBeInTheDocument();
    });
});

describe("RoomMemberDialogs timeout dialog", () => {
    it("stays shut until a target member is chosen", () => {
        // given
        const timeout = makeTimeout({ target: null });

        // when
        renderDialogs({ timeout });

        // then
        expect(screen.queryByRole("heading", { name: /Set timeout for/ })).not.toBeInTheDocument();
    });

    it("names the member it is timing out", () => {
        // given
        const timeout = makeTimeout({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ timeout });

        // then
        expect(screen.getByRole("heading", { name: "Set timeout for Beatrice" })).toBeInTheDocument();
    });

    it("offers every timeout unit from seconds to centuries", () => {
        // given
        const timeout = makeTimeout({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ timeout });

        // then
        expect(screen.getAllByRole("option").map(option => option.textContent)).toEqual([
            "seconds",
            "hours",
            "weeks",
            "years",
            "decades",
            "centuries",
        ]);
    });

    it("reports a change of unit", async () => {
        // given
        const user = userEvent.setup();
        const timeout = makeTimeout({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ timeout });
        await user.selectOptions(screen.getByRole("combobox"), "weeks");

        // then
        expect(timeout.onUnitChange).toHaveBeenCalledWith("weeks");
    });

    it("reports a change of amount", async () => {
        // given
        const user = userEvent.setup();
        const timeout = makeTimeout({ target: makeMember("Beatrice"), amount: "1" });

        // when
        renderDialogs({ timeout });
        await user.type(screen.getByRole("spinbutton"), "2");

        // then
        expect(timeout.onAmountChange).toHaveBeenCalledWith("12");
    });

    it("refuses an amount below one", () => {
        // given
        const timeout = makeTimeout({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ timeout });

        // then
        expect(screen.getByRole("spinbutton")).toHaveAttribute("min", "1");
    });

    it("applies the timeout when Set timeout is pressed", async () => {
        // given
        const user = userEvent.setup();
        const timeout = makeTimeout({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ timeout });
        await user.click(screen.getByRole("button", { name: "Set timeout" }));

        // then
        expect(timeout.onSave).toHaveBeenCalledTimes(1);
    });

    it("cancels when the backdrop is clicked", async () => {
        // given
        const user = userEvent.setup();
        const timeout = makeTimeout({ target: makeMember("Beatrice") });

        // when
        const { container } = renderDialogs({ timeout });
        await user.click(container.firstElementChild as HTMLElement);

        // then
        expect(timeout.onCancel).toHaveBeenCalledTimes(1);
    });

    it("locks both buttons while the timeout is being applied", () => {
        // given
        const timeout = makeTimeout({ target: makeMember("Beatrice"), saving: true });

        // when
        renderDialogs({ timeout });

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    });

    it("shows the error the timeout came back with", () => {
        // given
        const timeout = makeTimeout({ target: makeMember("Beatrice"), error: "Enter a whole number" });

        // when
        renderDialogs({ timeout });

        // then
        expect(screen.getByText("Enter a whole number")).toBeInTheDocument();
    });
});

describe("RoomMemberDialogs shared shell", () => {
    it("shows both dialogs when both have a target", () => {
        // given
        const nickname = makeNickname({ target: makeMember("Beatrice") });
        const timeout = makeTimeout({ target: makeMember("Beatrice") });

        // when
        renderDialogs({ nickname, timeout });

        // then
        expect(screen.getByRole("heading", { name: "Change nickname for Beatrice" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Set timeout for Beatrice" })).toBeInTheDocument();
    });

    it("dresses the desktop and mobile variants differently", () => {
        // given
        const nickname = makeNickname({ target: makeMember("Beatrice") });

        // when
        const desktop = renderDialogs({ variant: "desktop", nickname });
        const desktopOverlay = desktop.container.firstElementChild?.className ?? "";
        desktop.unmount();
        const mobile = renderDialogs({ variant: "mobile", nickname });
        const mobileOverlay = mobile.container.firstElementChild?.className ?? "";

        // then
        expect(desktopOverlay).not.toEqual("");
        expect(mobileOverlay).not.toEqual(desktopOverlay);
    });
});
