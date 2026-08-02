import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { ShareButton } from "./ShareButton";

interface ShareDialogStubProps {
    isOpen: boolean;
    onClose: () => void;
    contentId: string;
    contentType: string;
    contentTitle?: string;
    onShared?: () => void;
}

vi.mock("../post/ShareDialog/ShareDialog", () => ({
    ShareDialog: ({ isOpen, onClose, contentId, contentType, contentTitle, onShared }: ShareDialogStubProps) => (
        <section aria-label="share dialog">
            <p>{`${isOpen ? "open" : "closed"} for ${contentType}/${contentId} titled ${contentTitle ?? "nothing"}`}</p>
            <button onClick={onClose}>dismiss dialog</button>
            <button onClick={onShared}>confirm share</button>
        </section>
    ),
}));

const sharer = makeUser({ id: "user-1", username: "battler", display_name: "Battler" });

describe("ShareButton", () => {
    it("stays hidden from signed out visitors", () => {
        // given
        const user = null;

        // when
        const { container } = renderWithProviders(<ShareButton contentId="post-1" contentType="post" />, { user });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("counts the existing shares on the button", () => {
        // given
        const shareCount = 4;

        // when
        renderWithProviders(<ShareButton contentId="post-1" contentType="post" shareCount={shareCount} />, {
            user: sharer,
        });

        // then
        expect(screen.getByRole("button", { name: "Share 4" })).toBeInTheDocument();
    });

    it("leaves the count off when nothing has been shared yet", () => {
        // given
        const shareCount = 0;

        // when
        renderWithProviders(<ShareButton contentId="post-1" contentType="post" shareCount={shareCount} />, {
            user: sharer,
        });

        // then
        expect(screen.getByRole("button", { name: "Share" })).toBeInTheDocument();
    });

    it("leaves the count off when there is no count at all", () => {
        // given
        const shareCount = undefined;

        // when
        renderWithProviders(<ShareButton contentId="post-1" contentType="post" shareCount={shareCount} />, {
            user: sharer,
        });

        // then
        expect(screen.getByRole("button", { name: "Share" })).toBeInTheDocument();
    });

    it("keeps the dialog out of the tree until the button is pressed", () => {
        // given
        const contentId = "post-1";

        // when
        renderWithProviders(<ShareButton contentId={contentId} contentType="post" />, { user: sharer });

        // then
        expect(screen.queryByRole("region", { name: "share dialog" })).not.toBeInTheDocument();
    });

    it("opens the dialog on the content it was asked to share", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ShareButton contentId="theory-3" contentType="theory" contentTitle="The Golden Truth" />, {
            user: sharer,
        });

        // when
        await user.click(screen.getByRole("button", { name: "Share" }));

        // then
        expect(screen.getByRole("region", { name: "share dialog" })).toBeInTheDocument();
        expect(screen.getByText("open for theory/theory-3 titled The Golden Truth")).toBeInTheDocument();
    });

    it("closes the dialog when the dialog asks to be closed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ShareButton contentId="post-1" contentType="post" />, { user: sharer });
        await user.click(screen.getByRole("button", { name: "Share" }));

        // when
        await user.click(screen.getByRole("button", { name: "dismiss dialog" }));

        // then
        expect(screen.queryByRole("region", { name: "share dialog" })).not.toBeInTheDocument();
    });

    it("passes the shared callback straight through to the dialog", async () => {
        // given
        const onShared = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<ShareButton contentId="post-1" contentType="post" onShared={onShared} />, {
            user: sharer,
        });
        await user.click(screen.getByRole("button", { name: "Share" }));

        // when
        await user.click(screen.getByRole("button", { name: "confirm share" }));

        // then
        expect(onShared).toHaveBeenCalledOnce();
    });
});
