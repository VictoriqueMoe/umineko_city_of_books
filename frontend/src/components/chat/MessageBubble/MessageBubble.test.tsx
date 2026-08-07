import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatMessage, ReactionGroup } from "../../../types/api";
import { MessageBubble } from "./MessageBubble";

const HEART = "❤";
const STAR = "⭐";

vi.mock("../EmojiPicker/EmojiPicker", () => ({
    EmojiPicker: ({ onPick, onClose }: { onPick: (emoji: string) => void; onClose: () => void }) => (
        <div>
            <button onClick={() => onPick("⭐")}>choose star</button>
            <button onClick={onClose}>dismiss emoji</button>
        </div>
    ),
}));

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "m1",
        room_id: "room-1",
        sender: { id: "u1", username: "beatrice", display_name: "Beatrice" },
        body: "the golden truth",
        is_system: false,
        created_at: "2026-01-01T00:00:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}

function makeReaction(overrides: Partial<ReactionGroup> = {}): ReactionGroup {
    return {
        emoji: HEART,
        count: 1,
        viewer_reacted: false,
        display_names: [],
        ...overrides,
    };
}

function chipFor(emoji: string): HTMLElement {
    const chip = screen.getByText(emoji).closest("button");
    if (!chip) {
        throw new Error(`no reaction chip for ${emoji}`);
    }

    return chip;
}

describe("MessageBubble", () => {
    it("renders another person's message with their name and body", () => {
        // given
        const message = makeMessage();

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });

    it("prefers the room nickname over the profile display name", () => {
        // given
        const message = makeMessage({ sender_nickname: "The Golden Witch" });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />);

        // then
        expect(screen.getByText("The Golden Witch")).toBeInTheDocument();
        expect(screen.queryByText("Beatrice")).not.toBeInTheDocument();
    });

    it("falls back to the username when the sender has no display name", () => {
        // given
        const message = makeMessage({ sender: { id: "u1", username: "beatrice", display_name: "   " } });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />);

        // then
        expect(screen.getByText("beatrice")).toBeInTheDocument();
    });

    it("renders a system message as bare text with no sender or controls", () => {
        // given
        const message = makeMessage({ is_system: true, body: "Battler joined the room" });

        // when
        renderWithProviders(
            <MessageBubble message={message} isOwn={false} onReply={() => {}} onReactionToggle={() => {}} />,
        );

        // then
        expect(screen.getByText("Battler joined the room")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Reply" })).not.toBeInTheDocument();
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
    });

    it("hides a blocked sender's message until it is revealed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<MessageBubble message={makeMessage()} isOwn={false} senderBlocked />);
        expect(screen.getByText("Message from a blocked user")).toBeInTheDocument();
        expect(screen.queryByText("the golden truth")).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Show" }));

        // then
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
        expect(screen.queryByText("Message from a blocked user")).not.toBeInTheDocument();
    });

    it("marks an edited message and titles the marker with the edit time", () => {
        // given
        const message = makeMessage({ edited_at: "2026-01-01T01:00:00Z" });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn />);

        // then
        const marker = screen.getByText("(edited)");
        expect(marker).toBeInTheDocument();
        expect(marker.getAttribute("title")).toMatch(/^Edited /);
    });

    it("leaves an unedited message unmarked", () => {
        // given
        const message = makeMessage();

        // when
        renderWithProviders(<MessageBubble message={message} isOwn />);

        // then
        expect(screen.queryByText("(edited)")).not.toBeInTheDocument();
    });

    it("shows the seen label when one is supplied", () => {
        // given
        const seenLabel = "Seen by Battler";

        // when
        renderWithProviders(<MessageBubble message={makeMessage()} isOwn seenLabel={seenLabel} />);

        // then
        expect(screen.getByText(/Seen by Battler/)).toBeInTheDocument();
    });

    it("shows the quoted message a reply was aimed at", () => {
        // given
        const message = makeMessage({
            reply_to: { id: "m0", sender_id: "u2", sender_name: "Battler", body_preview: "an earlier claim" },
        });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />);

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.getByText("an earlier claim")).toBeInTheDocument();
    });

    it("scrolls to the quoted message when the quote is clicked", async () => {
        // given
        const scrollIntoView = vi.spyOn(Element.prototype, "scrollIntoView").mockImplementation(() => {});
        const user = userEvent.setup();
        const quoted = makeMessage({ id: "m0", body: "the earlier statement" });
        const reply = makeMessage({
            id: "m1",
            reply_to: { id: "m0", sender_id: "u2", sender_name: "Battler", body_preview: "an earlier claim" },
        });
        renderWithProviders(
            <>
                <MessageBubble message={quoted} isOwn={false} />
                <MessageBubble message={reply} isOwn={false} />
            </>,
        );

        // when
        await user.click(screen.getByText("an earlier claim"));

        // then
        expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "smooth", block: "center" });
        scrollIntoView.mockRestore();
    });

    it("renders image and video attachments and opens the lightbox for an image", async () => {
        // given
        const onLightbox = vi.fn();
        const user = userEvent.setup();
        const message = makeMessage({
            media: [
                { id: 1, media_url: "https://cdn.example/photo.png", media_type: "image", sort_order: 0 },
                { id: 2, media_url: "https://cdn.example/clip.mp4", media_type: "video", sort_order: 1 },
            ],
        });
        const { container } = renderWithProviders(
            <MessageBubble message={message} isOwn={false} onLightbox={onLightbox} />,
        );

        // when
        const image = container.querySelector('img[src="https://cdn.example/photo.png"]');
        await user.click(image as Element);

        // then
        expect(container.querySelector('video[src="https://cdn.example/clip.mp4"]')).toBeInTheDocument();
        expect(onLightbox).toHaveBeenCalledWith("https://cdn.example/photo.png");
    });

    it("embeds a Giphy link instead of showing the raw url", async () => {
        // given
        const onLightbox = vi.fn();
        const user = userEvent.setup();
        const gif = "https://media.giphy.com/media/abc/giphy.gif";
        renderWithProviders(
            <MessageBubble message={makeMessage({ body: gif })} isOwn={false} onLightbox={onLightbox} />,
        );

        // when
        await user.click(screen.getByAltText("GIF"));

        // then
        expect(screen.getByAltText("GIF")).toHaveAttribute("src", gif);
        expect(screen.queryByText(gif)).not.toBeInTheDocument();
        expect(onLightbox).toHaveBeenCalledWith(gif);
    });

    it("leaves a giphy lookalike host as plain text", () => {
        // given
        const lookalike = "https://media.giphy.com.evil.test/media/abc/giphy.gif";

        // when
        renderWithProviders(<MessageBubble message={makeMessage({ body: lookalike })} isOwn={false} />);

        // then
        expect(screen.queryByAltText("GIF")).not.toBeInTheDocument();
        expect(screen.getByText(lookalike)).toBeInTheDocument();
    });

    it("embeds videos linked from YouTube alongside the message text", () => {
        // given
        const body = "watch this https://www.youtube.com/watch?v=dQw4w9WgXcQ";

        // when
        renderWithProviders(<MessageBubble message={makeMessage({ body })} isOwn={false} />);

        // then
        expect(screen.getByTitle("YouTube video")).toHaveAttribute(
            "src",
            "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
        );
    });

    it("links a mention once the mentioned user is known", () => {
        // given
        const message = makeMessage({ body: "@battler is wrong" });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />, {
            mentionResolver: { isKnown: () => true, request: () => {} },
        });

        // then
        expect(screen.getByRole("link", { name: "@battler" })).toHaveAttribute("href", "/user/battler");
    });

    it("leaves a mention as plain text while the user is unresolved", () => {
        // given
        const message = makeMessage({ body: "@battler is wrong" });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />, {
            mentionResolver: { isKnown: () => false, request: () => {} },
        });

        // then
        expect(screen.queryByRole("link", { name: "@battler" })).not.toBeInTheDocument();
        expect(screen.getByText(/@battler is wrong/)).toBeInTheDocument();
    });

    it("labels a pinned message and offers to unpin it", () => {
        // given
        const message = makeMessage({ pinned: true });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} canPin onPinToggle={() => {}} />);

        // then
        expect(screen.getByText("Pinned")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Unpin message" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Pin message" })).not.toBeInTheDocument();
    });

    it("withholds the pin control from someone who cannot pin", () => {
        // given
        const canPin = false;

        // when
        renderWithProviders(
            <MessageBubble message={makeMessage()} isOwn={false} canPin={canPin} onPinToggle={vi.fn()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Pin message" })).not.toBeInTheDocument();
    });

    it("pins the message through the supplied handler", async () => {
        // given
        const onPinToggle = vi.fn();
        const user = userEvent.setup();
        const message = makeMessage();
        renderWithProviders(<MessageBubble message={message} isOwn={false} canPin onPinToggle={onPinToggle} />);

        // when
        await user.click(screen.getByRole("button", { name: "Pin message" }));

        // then
        expect(onPinToggle).toHaveBeenCalledWith(message);
    });

    it("passes the message back when reply is used", async () => {
        // given
        const onReply = vi.fn();
        const user = userEvent.setup();
        const message = makeMessage();
        renderWithProviders(<MessageBubble message={message} isOwn={false} onReply={onReply} />);

        // when
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // then
        expect(onReply).toHaveBeenCalledWith(message);
    });

    it("only offers editing on your own message", () => {
        // given
        const isOwn = false;

        // when
        renderWithProviders(<MessageBubble message={makeMessage()} isOwn={isOwn} onEdit={() => Promise.resolve()} />);

        // then
        expect(screen.queryByRole("button", { name: "Edit message" })).not.toBeInTheDocument();
    });

    it("withdraws editing when the room no longer allows it", () => {
        // given
        const canEdit = false;

        // when
        renderWithProviders(
            <MessageBubble message={makeMessage()} isOwn canEdit={canEdit} onEdit={() => Promise.resolve()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Edit message" })).not.toBeInTheDocument();
    });

    it("announces the start of an edit to the parent", async () => {
        // given
        const onEditStart = vi.fn();
        const user = userEvent.setup();
        const message = makeMessage();
        renderWithProviders(
            <MessageBubble message={message} isOwn onEdit={() => Promise.resolve()} onEditStart={onEditStart} />,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Edit message" }));

        // then
        expect(onEditStart).toHaveBeenCalledWith(message);
    });

    it("lets a member delete their own message after confirming", async () => {
        // given
        const onDelete = vi.fn();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        const message = makeMessage();
        renderWithProviders(<MessageBubble message={message} isOwn onDelete={onDelete} />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete message" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this message?");
        expect(onDelete).toHaveBeenCalledWith(message);
        confirm.mockRestore();
    });

    it("keeps the message when the delete confirmation is declined", async () => {
        // given
        const onDelete = vi.fn();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderWithProviders(<MessageBubble message={makeMessage()} isOwn onDelete={onDelete} />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete message" }));

        // then
        expect(onDelete).not.toHaveBeenCalled();
        confirm.mockRestore();
    });

    it("hides the delete control from an ordinary member reading someone else's message", () => {
        // given
        const canModerate = false;

        // when
        renderWithProviders(
            <MessageBubble message={makeMessage()} isOwn={false} canModerate={canModerate} onDelete={vi.fn()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Delete message" })).not.toBeInTheDocument();
    });

    it("lets a moderator delete another member's message", () => {
        // given
        const canModerate = true;

        // when
        renderWithProviders(
            <MessageBubble message={makeMessage()} isOwn={false} canModerate={canModerate} onDelete={vi.fn()} />,
        );

        // then
        expect(screen.getByRole("button", { name: "Delete message" })).toBeInTheDocument();
    });

    it("stops a moderator deleting a staff member's message", () => {
        // given
        const senderIsStaff = true;

        // when
        renderWithProviders(
            <MessageBubble
                message={makeMessage()}
                isOwn={false}
                canModerate
                senderIsStaff={senderIsStaff}
                onDelete={vi.fn()}
            />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Delete message" })).not.toBeInTheDocument();
    });

    it("renders each reaction with its own count", () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 3 }), makeReaction({ emoji: STAR, count: 1 })];

        // when
        renderWithProviders(<MessageBubble message={makeMessage({ reactions })} isOwn={false} />);

        // then
        expect(chipFor(HEART)).toHaveTextContent("3");
        expect(chipFor(STAR)).toHaveTextContent("1");
    });

    it("toggles an existing reaction when its chip is clicked", async () => {
        // given
        const onReactionToggle = vi.fn();
        const user = userEvent.setup();
        const message = makeMessage({ reactions: [makeReaction({ emoji: HEART, count: 3 })] });
        renderWithProviders(<MessageBubble message={message} isOwn={false} onReactionToggle={onReactionToggle} />);

        // when
        await user.click(chipFor(HEART));

        // then
        expect(onReactionToggle).toHaveBeenCalledWith(message, HEART);
    });

    it("tells the viewer their own reaction can be removed", () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 3, viewer_reacted: true })];

        // when
        renderWithProviders(
            <MessageBubble message={makeMessage({ reactions })} isOwn={false} onReactionToggle={vi.fn()} />,
        );

        // then
        expect(chipFor(HEART)).toHaveAttribute("title", "Click to remove your reaction");
    });

    it("refuses reaction toggles while the viewer is timed out", async () => {
        // given
        const onReactionToggle = vi.fn();
        const user = userEvent.setup();
        const reactions = [makeReaction({ emoji: HEART, count: 3, display_names: ["Battler"] })];
        renderWithProviders(
            <MessageBubble
                message={makeMessage({ reactions })}
                isOwn={false}
                canReact={false}
                onReactionToggle={onReactionToggle}
            />,
        );

        // when
        await user.click(chipFor(HEART));

        // then
        expect(onReactionToggle).not.toHaveBeenCalled();
        expect(chipFor(HEART)).toHaveAttribute("title", "You are timed out");
    });

    it("disables an untouched chip and hides the react control while the viewer is timed out", () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 3 })];

        // when
        renderWithProviders(
            <MessageBubble
                message={makeMessage({ reactions })}
                isOwn={false}
                canReact={false}
                onReactionToggle={vi.fn()}
            />,
        );

        // then
        expect(chipFor(HEART)).toBeDisabled();
        expect(screen.queryByRole("button", { name: "React" })).not.toBeInTheDocument();
    });

    it("forwards the emoji chosen from the picker and closes it", async () => {
        // given
        const onReactionToggle = vi.fn();
        const user = userEvent.setup();
        const message = makeMessage();
        renderWithProviders(<MessageBubble message={message} isOwn={false} onReactionToggle={onReactionToggle} />);

        // when
        await user.click(screen.getByRole("button", { name: "React" }));
        await user.click(screen.getByRole("button", { name: "choose star" }));

        // then
        expect(onReactionToggle).toHaveBeenCalledWith(message, STAR);
        expect(screen.queryByRole("button", { name: "choose star" })).not.toBeInTheDocument();
    });

    it("lists who reacted when a chip is right clicked", async () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 2, display_names: ["Beatrice", "Battler"] })];
        renderWithProviders(
            <MessageBubble message={makeMessage({ reactions })} isOwn={false} onReactionToggle={vi.fn()} />,
        );

        // when
        fireEvent.contextMenu(chipFor(HEART));

        // then
        const popover = await screen.findByRole("dialog", { name: "Reactors" });
        expect(popover).toHaveTextContent("2 reacted");
        expect(popover).toHaveTextContent("Beatrice");
        expect(popover).toHaveTextContent("Battler");
    });

    it("says so when no reactor names came back from the server", async () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 4, display_names: [] })];
        renderWithProviders(
            <MessageBubble message={makeMessage({ reactions })} isOwn={false} onReactionToggle={vi.fn()} />,
        );

        // when
        fireEvent.contextMenu(chipFor(HEART));

        // then
        expect(await screen.findByText("No reactor names available.")).toBeInTheDocument();
    });

    it("closes the reactor list when the page is clicked elsewhere", async () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 2, display_names: ["Beatrice", "Battler"] })];
        renderWithProviders(
            <MessageBubble message={makeMessage({ reactions })} isOwn={false} onReactionToggle={vi.fn()} />,
        );
        fireEvent.contextMenu(chipFor(HEART));
        await screen.findByRole("dialog", { name: "Reactors" });

        // when
        fireEvent.mouseDown(document.body);

        // then
        await waitFor(() => expect(screen.queryByRole("dialog", { name: "Reactors" })).not.toBeInTheDocument());
    });

    it("starts the editor from the existing body", () => {
        // given
        const message = makeMessage({ body: "the golden truth" });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn editing onEdit={() => Promise.resolve()} />);

        // then
        expect(screen.getByRole("textbox")).toHaveValue("the golden truth");
        expect(screen.getByText("Enter to save · Esc to cancel")).toBeInTheDocument();
    });

    it("commits the new body on Enter and then leaves edit mode", async () => {
        // given
        const onEdit = vi.fn(() => Promise.resolve());
        const onEditCancel = vi.fn();
        const user = userEvent.setup();
        const message = makeMessage();
        renderWithProviders(
            <MessageBubble message={message} isOwn editing onEdit={onEdit} onEditCancel={onEditCancel} />,
        );

        // when
        const editor = screen.getByRole("textbox");
        await user.clear(editor);
        await user.type(editor, "the red truth");
        await user.keyboard("{Enter}");

        // then
        await waitFor(() => expect(onEdit).toHaveBeenCalledWith(message, "the red truth"));
        expect(onEditCancel).toHaveBeenCalled();
    });

    it("abandons the edit on Escape without saving", async () => {
        // given
        const onEdit = vi.fn(() => Promise.resolve());
        const onEditCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <MessageBubble message={makeMessage()} isOwn editing onEdit={onEdit} onEditCancel={onEditCancel} />,
        );

        // when
        await user.type(screen.getByRole("textbox"), " and more");
        await user.keyboard("{Escape}");

        // then
        expect(onEditCancel).toHaveBeenCalled();
        expect(onEdit).not.toHaveBeenCalled();
    });

    it("treats an unchanged edit as a cancellation", async () => {
        // given
        const onEdit = vi.fn(() => Promise.resolve());
        const onEditCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <MessageBubble message={makeMessage()} isOwn editing onEdit={onEdit} onEditCancel={onEditCancel} />,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(onEdit).not.toHaveBeenCalled();
        expect(onEditCancel).toHaveBeenCalled();
    });

    it("blocks saving an emptied edit", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<MessageBubble message={makeMessage()} isOwn editing onEdit={() => Promise.resolve()} />);

        // when
        await user.clear(screen.getByRole("textbox"));

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("cancels the edit through the cancel control", async () => {
        // given
        const onEditCancel = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <MessageBubble
                message={makeMessage()}
                isOwn
                editing
                onEdit={() => Promise.resolve()}
                onEditCancel={onEditCancel}
            />,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onEditCancel).toHaveBeenCalledOnce();
    });

    it("hides the edit control while the message is already being edited", () => {
        // given
        const editing = true;

        // when
        renderWithProviders(
            <MessageBubble message={makeMessage()} isOwn editing={editing} onEdit={() => Promise.resolve()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Edit message" })).not.toBeInTheDocument();
    });
});
