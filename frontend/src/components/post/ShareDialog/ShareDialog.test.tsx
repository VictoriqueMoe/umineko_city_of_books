import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { ShareDialog } from "./ShareDialog";

const mocks = vi.hoisted(() => ({
    createPost: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../../api/mutations/post", () => ({
    useCreatePost: () => ({ mutateAsync: mocks.createPost }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

const contentId = "11111111-1111-1111-1111-111111111111";

interface DialogOverrides {
    isOpen?: boolean;
    contentTitle?: string;
    onClose?: () => void;
    onShared?: () => void;
}

function renderDialog(overrides: DialogOverrides = {}) {
    const onClose = overrides.onClose ?? vi.fn();
    const onShared = overrides.onShared;

    renderWithProviders(
        <ShareDialog
            isOpen={overrides.isOpen ?? true}
            onClose={onClose}
            contentId={contentId}
            contentType="theory"
            contentTitle={overrides.contentTitle}
            onShared={onShared}
        />,
    );

    return { onClose };
}

beforeEach(() => {
    mocks.createPost.mockResolvedValue({ id: "22222222-2222-2222-2222-222222222222" });
});

describe("ShareDialog", () => {
    it("renders nothing while it is closed", () => {
        // given
        const isOpen = false;

        // when
        renderDialog({ isOpen });

        // then
        expect(screen.queryByText("Share to Game Board")).not.toBeInTheDocument();
    });

    it("previews the title of what is being shared", () => {
        // given
        const contentTitle = "Kanon is Yasu";

        // when
        renderDialog({ contentTitle });

        // then
        expect(screen.getByText("Sharing: Kanon is Yasu")).toBeInTheDocument();
    });

    it("falls back to the content type when there is no title", () => {
        // given
        const contentTitle = undefined;

        // when
        renderDialog({ contentTitle });

        // then
        expect(screen.getByText("Sharing: theory")).toBeInTheDocument();
    });

    it("shares to the general corner with an empty message by default", async () => {
        // given
        const user = userEvent.setup();
        renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "Share" }));

        // then
        expect(mocks.createPost).toHaveBeenCalledWith({
            body: "",
            corner: "general",
            sharedContentId: contentId,
            sharedContentType: "theory",
        });
    });

    it("sends the typed message and the chosen corner", async () => {
        // given
        const user = userEvent.setup();
        renderDialog();

        // when
        await user.type(screen.getByPlaceholderText("Add a comment (optional)"), "look at this");
        await user.selectOptions(screen.getByRole("combobox"), "umineko");
        await user.click(screen.getByRole("button", { name: "Share" }));

        // then
        expect(mocks.createPost).toHaveBeenCalledWith({
            body: "look at this",
            corner: "umineko",
            sharedContentId: contentId,
            sharedContentType: "theory",
        });
    });

    it("notifies the opener, closes and lands on the new post", async () => {
        // given
        const user = userEvent.setup();
        const onShared = vi.fn();
        const { onClose } = renderDialog({ onShared });

        // when
        await user.click(screen.getByRole("button", { name: "Share" }));

        // then
        expect(onShared).toHaveBeenCalledOnce();
        expect(onClose).toHaveBeenCalledOnce();
        expect(mocks.navigate).toHaveBeenCalledWith("/game-board/22222222-2222-2222-2222-222222222222");
    });

    it("keeps the dialog open and shows why the share failed", async () => {
        // given
        const user = userEvent.setup();
        mocks.createPost.mockRejectedValue(new Error("the witch forbids it"));
        const { onClose } = renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "Share" }));

        // then
        expect(await screen.findByText("the witch forbids it")).toBeInTheDocument();
        expect(onClose).not.toHaveBeenCalled();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("shows a generic failure when the rejection is not an error", async () => {
        // given
        const user = userEvent.setup();
        mocks.createPost.mockRejectedValue("no reason given");
        renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "Share" }));

        // then
        expect(await screen.findByText("Failed to share")).toBeInTheDocument();
    });

    it("locks both actions while the share is in flight", async () => {
        // given
        const user = userEvent.setup();
        let release: (result: { id: string }) => void = () => {};
        mocks.createPost.mockImplementation(
            () =>
                new Promise<{ id: string }>(resolve => {
                    release = resolve;
                }),
        );
        renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "Share" }));

        // then
        expect(screen.getByRole("button", { name: "Sharing..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
        await act(async () => {
            release({ id: "33333333-3333-3333-3333-333333333333" });
        });
        expect(mocks.navigate).toHaveBeenCalledWith("/game-board/33333333-3333-3333-3333-333333333333");
    });

    it("only submits one share even when the button is pressed twice", async () => {
        // given
        const user = userEvent.setup();
        mocks.createPost.mockImplementation(() => new Promise<{ id: string }>(() => {}));
        renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "Share" }));
        await user.click(screen.getByRole("button", { name: "Sharing..." }));

        // then
        expect(mocks.createPost).toHaveBeenCalledOnce();
    });

    it("closes without sharing when cancel is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { onClose } = renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
        expect(mocks.createPost).not.toHaveBeenCalled();
    });

    it("closes without sharing when the dismiss control is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { onClose } = renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "✕" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
        expect(mocks.createPost).not.toHaveBeenCalled();
    });
});
