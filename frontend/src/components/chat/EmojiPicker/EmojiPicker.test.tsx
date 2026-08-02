import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { EmojiPicker } from "./EmojiPicker";

vi.mock("emoji-picker-react", () => ({
    default: ({
        onEmojiClick,
        searchPlaceholder,
    }: {
        onEmojiClick: (data: { emoji: string }) => void;
        searchPlaceholder: string;
    }) => (
        <div>
            <span>{searchPlaceholder}</span>
            <button type="button" onClick={() => onEmojiClick({ emoji: "\u{1F339}" })}>
                pick a rose
            </button>
        </div>
    ),
}));

function noop() {}

describe("EmojiPicker", () => {
    it("shows a loading placeholder until the picker bundle arrives", () => {
        // given
        const onPick = vi.fn();

        // when
        renderWithProviders(<EmojiPicker onPick={onPick} onClose={noop} />);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("renders the picker once its bundle has loaded", async () => {
        // given
        const onPick = vi.fn();

        // when
        renderWithProviders(<EmojiPicker onPick={onPick} onClose={noop} />);

        // then
        expect(await screen.findByText("Search emoji")).toBeInTheDocument();
        expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
    });

    it("reports only the emoji character of the chosen entry", async () => {
        // given
        const onPick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<EmojiPicker onPick={onPick} onClose={noop} />);
        const pick = await screen.findByRole("button", { name: "pick a rose" });

        // when
        await user.click(pick);

        // then
        expect(onPick).toHaveBeenCalledWith("\u{1F339}");
    });

    it("closes when escape is pressed", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<EmojiPicker onPick={noop} onClose={onClose} />);

        // when
        await user.keyboard("{Escape}");

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("ignores other keys", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<EmojiPicker onPick={noop} onClose={onClose} />);

        // when
        await user.keyboard("{Enter}");

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("closes when a press lands outside the picker", () => {
        // given
        const onClose = vi.fn();
        renderWithProviders(<EmojiPicker onPick={noop} onClose={onClose} />);

        // when
        fireEvent.mouseDown(document.body);

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("stays open when a press lands inside the picker", async () => {
        // given
        const onClose = vi.fn();
        renderWithProviders(<EmojiPicker onPick={noop} onClose={onClose} />);
        const pick = await screen.findByRole("button", { name: "pick a rose" });

        // when
        fireEvent.mouseDown(pick);

        // then
        expect(onClose).not.toHaveBeenCalled();
    });

    it("stops listening to the document once it is unmounted", () => {
        // given
        const onClose = vi.fn();
        const { unmount } = renderWithProviders(<EmojiPicker onPick={noop} onClose={onClose} />);

        // when
        unmount();
        fireEvent.mouseDown(document.body);
        fireEvent.keyDown(document, { key: "Escape" });

        // then
        expect(onClose).not.toHaveBeenCalled();
    });
});
