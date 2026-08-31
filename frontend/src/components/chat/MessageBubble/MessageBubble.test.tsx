import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ReactionGroup } from "../../../types/api";
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

function makeReaction(overrides: Partial<ReactionGroup> = {}): ReactionGroup {
    return {
        emoji: HEART,
        count: 1,
        viewer_reacted: false,
        display_names: [],
        ...overrides,
    };
}

interface ReactionGuardCase {
    name: string;
    canReact: boolean;
    withHandler: boolean;
    chipEnabled: boolean;
    reactControl: boolean;
}

const reactionGuardCases: ReactionGuardCase[] = [
    {
        name: "a member whose list wired the handler",
        canReact: true,
        withHandler: true,
        chipEnabled: true,
        reactControl: true,
    },
    {
        name: "a list that forgot the handler",
        canReact: true,
        withHandler: false,
        chipEnabled: false,
        reactControl: false,
    },
    { name: "a timed out member", canReact: false, withHandler: true, chipEnabled: false, reactControl: false },
];

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
        const message = makeChatMessage();

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });

    it("prefers the room nickname over the profile display name", () => {
        // given
        const message = makeChatMessage({ sender_nickname: "The Golden Witch" });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />);

        // then
        expect(screen.getByText("The Golden Witch")).toBeInTheDocument();
        expect(screen.queryByText("Beatrice")).not.toBeInTheDocument();
    });

    it("falls back to the username when the sender has no display name", () => {
        // given
        const message = makeChatMessage({ sender: { id: "u1", username: "beatrice", display_name: "   " } });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />);

        // then
        expect(screen.getByText("beatrice")).toBeInTheDocument();
    });

    it("renders a system message as bare text with no sender or controls", () => {
        // given
        const message = makeChatMessage({ is_system: true, body: "Battler joined the room" });

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
        renderWithProviders(<MessageBubble message={makeChatMessage()} isOwn={false} senderBlocked />);
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
        const message = makeChatMessage({ edited_at: "2026-01-01T01:00:00Z" });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn />);

        // then
        const marker = screen.getByText("(edited)");
        expect(marker).toBeInTheDocument();
        expect(marker.getAttribute("title")).toMatch(/^Edited /);
    });

    it("leaves an unedited message unmarked", () => {
        // given
        const message = makeChatMessage();

        // when
        renderWithProviders(<MessageBubble message={message} isOwn />);

        // then
        expect(screen.queryByText("(edited)")).not.toBeInTheDocument();
    });

    it("shows the seen label when one is supplied", () => {
        // given
        const seenLabel = "Seen by Battler";

        // when
        renderWithProviders(<MessageBubble message={makeChatMessage()} isOwn seenLabel={seenLabel} />);

        // then
        expect(screen.getByText(/Seen by Battler/)).toBeInTheDocument();
    });

    it("shows the quoted message a reply was aimed at", () => {
        // given
        const message = makeChatMessage({
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
        const quoted = makeChatMessage({ id: "m0", body: "the earlier statement" });
        const reply = makeChatMessage({
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
        const message = makeChatMessage({
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
            <MessageBubble message={makeChatMessage({ body: gif })} isOwn={false} onLightbox={onLightbox} />,
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
        renderWithProviders(<MessageBubble message={makeChatMessage({ body: lookalike })} isOwn={false} />);

        // then
        expect(screen.queryByAltText("GIF")).not.toBeInTheDocument();
        expect(screen.getByText(lookalike)).toBeInTheDocument();
    });

    it("embeds videos linked from YouTube alongside the message text", () => {
        // given
        const body = "watch this https://www.youtube.com/watch?v=dQw4w9WgXcQ";

        // when
        renderWithProviders(<MessageBubble message={makeChatMessage({ body })} isOwn={false} />);

        // then
        expect(screen.getByTitle("YouTube video")).toHaveAttribute(
            "src",
            "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
        );
    });

    it("links a mention once the mentioned user is known", () => {
        // given
        const message = makeChatMessage({ body: "@battler is wrong" });

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />, {
            mentionResolver: { isKnown: () => true, request: () => {} },
        });

        // then
        expect(screen.getByRole("link", { name: "@battler" })).toHaveAttribute("href", "/user/battler");
    });

    it("leaves a mention as plain text while the user is unresolved", () => {
        // given
        const message = makeChatMessage({ body: "@battler is wrong" });

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
        const message = makeChatMessage({ pinned: true });

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
            <MessageBubble message={makeChatMessage()} isOwn={false} canPin={canPin} onPinToggle={vi.fn()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Pin message" })).not.toBeInTheDocument();
    });

    it("pins the message through the supplied handler", async () => {
        // given
        const onPinToggle = vi.fn();
        const user = userEvent.setup();
        const message = makeChatMessage();
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
        const message = makeChatMessage();
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
        renderWithProviders(
            <MessageBubble message={makeChatMessage()} isOwn={isOwn} onEdit={() => Promise.resolve()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Edit message" })).not.toBeInTheDocument();
    });

    it("withdraws editing when the room no longer allows it", () => {
        // given
        const canEdit = false;

        // when
        renderWithProviders(
            <MessageBubble message={makeChatMessage()} isOwn canEdit={canEdit} onEdit={() => Promise.resolve()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Edit message" })).not.toBeInTheDocument();
    });

    it("announces the start of an edit to the parent", async () => {
        // given
        const onEditStart = vi.fn();
        const user = userEvent.setup();
        const message = makeChatMessage();
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
        const message = makeChatMessage();
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
        renderWithProviders(<MessageBubble message={makeChatMessage()} isOwn onDelete={onDelete} />);

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
            <MessageBubble message={makeChatMessage()} isOwn={false} canModerate={canModerate} onDelete={vi.fn()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Delete message" })).not.toBeInTheDocument();
    });

    it("lets a moderator delete another member's message", () => {
        // given
        const canModerate = true;

        // when
        renderWithProviders(
            <MessageBubble message={makeChatMessage()} isOwn={false} canModerate={canModerate} onDelete={vi.fn()} />,
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
                message={makeChatMessage()}
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
        renderWithProviders(<MessageBubble message={makeChatMessage({ reactions })} isOwn={false} />);

        // then
        expect(chipFor(HEART)).toHaveTextContent("3");
        expect(chipFor(STAR)).toHaveTextContent("1");
    });

    it.each(reactionGuardCases)(
        "keeps the chip and the react control in agreement for $name",
        ({ canReact, withHandler, chipEnabled, reactControl }) => {
            // given
            const reactions = [makeReaction({ emoji: HEART, count: 3 })];

            // when
            renderWithProviders(
                <MessageBubble
                    message={makeChatMessage({ reactions })}
                    isOwn={false}
                    canReact={canReact}
                    onReactionToggle={withHandler ? vi.fn() : undefined}
                />,
            );

            // then
            expect(chipFor(HEART).hasAttribute("disabled")).toBe(!chipEnabled);
            expect(Boolean(screen.queryByRole("button", { name: "React" }))).toBe(reactControl);
        },
    );

    it("stops promising a reaction the caller never wired up", () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 3 })];

        // when
        renderWithProviders(<MessageBubble message={makeChatMessage({ reactions })} isOwn={false} />);

        // then
        expect(chipFor(HEART)).not.toHaveAttribute("title");
    });

    it("toggles an existing reaction when its chip is clicked", async () => {
        // given
        const onReactionToggle = vi.fn();
        const user = userEvent.setup();
        const message = makeChatMessage({ reactions: [makeReaction({ emoji: HEART, count: 3 })] });
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
            <MessageBubble message={makeChatMessage({ reactions })} isOwn={false} onReactionToggle={vi.fn()} />,
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
                message={makeChatMessage({ reactions })}
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
                message={makeChatMessage({ reactions })}
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
        const message = makeChatMessage();
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
            <MessageBubble message={makeChatMessage({ reactions })} isOwn={false} onReactionToggle={vi.fn()} />,
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
            <MessageBubble message={makeChatMessage({ reactions })} isOwn={false} onReactionToggle={vi.fn()} />,
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
            <MessageBubble message={makeChatMessage({ reactions })} isOwn={false} onReactionToggle={vi.fn()} />,
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
        const message = makeChatMessage({ body: "the golden truth" });

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
        const message = makeChatMessage();
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
            <MessageBubble message={makeChatMessage()} isOwn editing onEdit={onEdit} onEditCancel={onEditCancel} />,
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
            <MessageBubble message={makeChatMessage()} isOwn editing onEdit={onEdit} onEditCancel={onEditCancel} />,
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
        renderWithProviders(
            <MessageBubble message={makeChatMessage()} isOwn editing onEdit={() => Promise.resolve()} />,
        );

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
                message={makeChatMessage()}
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
            <MessageBubble message={makeChatMessage()} isOwn editing={editing} onEdit={() => Promise.resolve()} />,
        );

        // then
        expect(screen.queryByRole("button", { name: "Edit message" })).not.toBeInTheDocument();
    });
});

const EVERY_ACTION = ["React", "Reply", "Pin message", "Edit message", "Delete message"];

function bodyOf(): HTMLElement {
    return screen.getByText("the golden truth");
}

function bubbleOf(): HTMLElement {
    const bubble = document.getElementById("chat-msg-m1");
    if (!bubble) {
        throw new Error("no message bubble");
    }

    return bubble;
}

interface FullyWiredBubble {
    message: ReturnType<typeof makeChatMessage>;
    onReply: ReturnType<typeof vi.fn>;
    onReactionToggle: ReturnType<typeof vi.fn>;
    onPinToggle: ReturnType<typeof vi.fn>;
    onDelete: ReturnType<typeof vi.fn>;
    onEditStart: ReturnType<typeof vi.fn>;
}

function renderFullyWired(): FullyWiredBubble {
    const message = makeChatMessage();
    const handlers = {
        onReply: vi.fn(),
        onReactionToggle: vi.fn(),
        onPinToggle: vi.fn(),
        onDelete: vi.fn(),
        onEditStart: vi.fn(),
    };

    renderWithProviders(
        <MessageBubble message={message} isOwn canPin canModerate onEdit={() => Promise.resolve()} {...handlers} />,
    );

    return { message, ...handlers };
}

describe("MessageBubble context menu", () => {
    afterEach(() => {
        vi.useRealTimers();
    });

    it("offers exactly the actions the hover bar offers", () => {
        // given
        renderFullyWired();
        for (const label of EVERY_ACTION) {
            expect(screen.getByRole("button", { name: label })).toBeInTheDocument();
        }

        // when
        fireEvent.contextMenu(bodyOf());

        // then
        expect(screen.getAllByRole("menuitem").map(item => item.textContent)).toHaveLength(EVERY_ACTION.length);
        for (const label of EVERY_ACTION) {
            expect(screen.getByRole("menuitem", { name: label })).toBeInTheDocument();
        }
    });

    it("offers only what a live stream chat panel wired up", () => {
        // given
        renderWithProviders(
            <MessageBubble
                message={makeChatMessage()}
                isOwn
                onReply={vi.fn()}
                onEdit={() => Promise.resolve()}
                onEditStart={vi.fn()}
            />,
        );

        // when
        fireEvent.contextMenu(bodyOf());

        // then
        expect(screen.getAllByRole("menuitem").map(item => item.textContent)).toHaveLength(2);
        expect(screen.getByRole("menuitem", { name: "Reply" })).toBeInTheDocument();
        expect(screen.getByRole("menuitem", { name: "Edit message" })).toBeInTheDocument();
    });

    it("leaves the browser its own menu when there is no action to offer", () => {
        // given
        renderWithProviders(<MessageBubble message={makeChatMessage()} isOwn={false} />);

        // when
        const notPrevented = fireEvent.contextMenu(bodyOf());

        // then
        expect(notPrevented).toBe(true);
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("leaves the browser its own menu over an image so it can still be saved", () => {
        // given
        const message = makeChatMessage({
            media: [{ id: 1, media_url: "https://cdn.example/photo.png", media_type: "image", sort_order: 0 }],
        });
        const { container } = renderWithProviders(
            <MessageBubble message={message} isOwn canPin canModerate onReply={vi.fn()} onDelete={vi.fn()} />,
        );

        // when
        const notPrevented = fireEvent.contextMenu(
            container.querySelector('img[src="https://cdn.example/photo.png"]') as Element,
        );

        // then
        expect(notPrevented).toBe(true);
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("keeps a right clicked reaction chip showing its reactors and nothing else", async () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 2, display_names: ["Beatrice", "Battler"] })];
        renderWithProviders(
            <MessageBubble
                message={makeChatMessage({ reactions })}
                isOwn
                onReply={vi.fn()}
                onReactionToggle={vi.fn()}
            />,
        );

        // when
        fireEvent.contextMenu(chipFor(HEART));

        // then
        expect(await screen.findByRole("dialog", { name: "Reactors" })).toBeInTheDocument();
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("replies through the same handler the hover bar uses", async () => {
        // given
        const user = userEvent.setup();
        const { message, onReply } = renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        await user.click(screen.getByRole("menuitem", { name: "Reply" }));

        // then
        expect(onReply).toHaveBeenCalledWith(message);
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("pins through the same handler the hover bar uses", async () => {
        // given
        const user = userEvent.setup();
        const { message, onPinToggle } = renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        await user.click(screen.getByRole("menuitem", { name: "Pin message" }));

        // then
        expect(onPinToggle).toHaveBeenCalledWith(message);
    });

    it("still asks before deleting when the delete comes from the menu", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        const { message, onDelete } = renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        await user.click(screen.getByRole("menuitem", { name: "Delete message" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this message?");
        expect(onDelete).toHaveBeenCalledWith(message);
        confirm.mockRestore();
    });

    it("keeps the message when the delete confirmation from the menu is declined", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        const { onDelete } = renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        await user.click(screen.getByRole("menuitem", { name: "Delete message" }));

        // then
        expect(onDelete).not.toHaveBeenCalled();
        confirm.mockRestore();
    });

    it("leaves the caret in the editor when editing starts from the menu", async () => {
        // given
        const user = userEvent.setup();
        const message = makeChatMessage();
        const onEditStart = vi.fn();
        const { rerender } = renderWithProviders(
            <MessageBubble message={message} isOwn onEdit={() => Promise.resolve()} onEditStart={onEditStart} />,
        );
        fireEvent.contextMenu(bodyOf());

        // when
        await user.click(screen.getByRole("menuitem", { name: "Edit message" }));
        rerender(<MessageBubble message={message} isOwn editing onEdit={() => Promise.resolve()} />);

        // then
        expect(onEditStart).toHaveBeenCalledWith(message);
        expect(screen.getByRole("textbox")).toHaveFocus();
    });

    it("opens the emoji picker from the menu and forwards the emoji chosen", async () => {
        // given
        const user = userEvent.setup();
        const { message, onReactionToggle } = renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        await user.click(screen.getByRole("menuitem", { name: "React" }));
        await user.click(screen.getByRole("button", { name: "choose star" }));

        // then
        expect(onReactionToggle).toHaveBeenCalledWith(message, STAR);
        expect(screen.queryByRole("button", { name: "choose star" })).not.toBeInTheDocument();
    });

    it("stays open while the message list scrolls underneath it", () => {
        // given
        renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        fireEvent.scroll(document, {});

        // then
        expect(screen.getByRole("menu", { name: "Message actions" })).toBeInTheDocument();
    });

    it("closes on Escape and hands focus back to the message it came from", async () => {
        // given
        const user = userEvent.setup();
        renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        await user.keyboard("{Escape}");

        // then
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: EVERY_ACTION[0] })).toHaveFocus();
        expect(bubbleOf().contains(document.activeElement)).toBe(true);
    });

    it("closes when the page is clicked elsewhere", async () => {
        // given
        renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        fireEvent.mouseDown(document.body);

        // then
        await waitFor(() => expect(screen.queryByRole("menu")).not.toBeInTheDocument());
    });

    it("opens on a long press so a touch reader reaches the same actions", () => {
        // given
        vi.useFakeTimers();
        renderFullyWired();

        // when
        fireEvent.pointerDown(bodyOf(), { pointerType: "touch", clientX: 40, clientY: 60 });
        act(() => {
            vi.advanceTimersByTime(450);
        });

        // then
        expect(screen.getAllByRole("menuitem")).toHaveLength(EVERY_ACTION.length);
    });

    it("leaves a short touch alone", () => {
        // given
        vi.useFakeTimers();
        renderFullyWired();

        // when
        fireEvent.pointerDown(bodyOf(), { pointerType: "touch", clientX: 40, clientY: 60 });
        fireEvent.pointerUp(bodyOf(), { pointerType: "touch" });
        act(() => {
            vi.advanceTimersByTime(450);
        });

        // then
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("never opens a second menu from a long press on a reaction chip", () => {
        // given
        vi.useFakeTimers();
        const reactions = [makeReaction({ emoji: HEART, count: 2, display_names: ["Beatrice"] })];
        renderWithProviders(
            <MessageBubble
                message={makeChatMessage({ reactions })}
                isOwn
                onReply={vi.fn()}
                onReactionToggle={vi.fn()}
            />,
        );

        // when
        fireEvent.pointerDown(chipFor(HEART), { pointerType: "touch" });
        act(() => {
            vi.advanceTimersByTime(450);
        });

        // then
        expect(screen.getByRole("dialog", { name: "Reactors" })).toBeInTheDocument();
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });
});

const ALL_LABELS = ["React", "Reply", "Pin message", "Unpin message", "Edit message", "Delete message"];

function labelOf(control: HTMLElement): string {
    const text = control.getAttribute("aria-label") ?? control.textContent ?? "";
    const label = ALL_LABELS.find(candidate => text.endsWith(candidate));
    if (!label) {
        throw new Error(`an action control carried no recognisable label: "${text}"`);
    }

    return label;
}

function hoverBarLabels(): string[] {
    return screen
        .queryAllByRole("button")
        .filter(button => button.hasAttribute("aria-label"))
        .map(labelOf);
}

function menuLabels(): string[] {
    return screen.queryAllByRole("menuitem").map(labelOf);
}

const everythingWired = {
    onReply: vi.fn(),
    onReactionToggle: vi.fn(),
    onPinToggle: vi.fn(),
    onDelete: vi.fn(),
    onEditStart: vi.fn(),
    onEdit: () => Promise.resolve(),
};

interface ParityCase {
    name: string;
    message?: ReturnType<typeof makeChatMessage>;
    props: Omit<ComponentProps<typeof MessageBubble>, "message">;
    want: string[];
}

const parityCases: ParityCase[] = [
    {
        name: "their own message in a room they host",
        props: { ...everythingWired, isOwn: true, canPin: true, canModerate: true },
        want: EVERY_ACTION,
    },
    {
        name: "somebody else's message read by an ordinary member",
        props: { ...everythingWired, isOwn: false },
        want: ["React", "Reply"],
    },
    {
        name: "somebody else's message read by a moderator",
        props: { ...everythingWired, isOwn: false, canModerate: true },
        want: ["React", "Reply", "Delete message"],
    },
    {
        name: "a staff member's message read by a moderator",
        props: { ...everythingWired, isOwn: false, canModerate: true, senderIsStaff: true },
        want: ["React", "Reply"],
    },
    {
        name: "somebody else's message where the reader may pin",
        props: { ...everythingWired, isOwn: false, canPin: true },
        want: ["React", "Reply", "Pin message"],
    },
    {
        name: "a message that is already pinned",
        message: makeChatMessage({ pinned: true }),
        props: { ...everythingWired, isOwn: false, canPin: true },
        want: ["React", "Reply", "Unpin message"],
    },
    {
        name: "a reader whose room withholds reacting",
        props: { ...everythingWired, isOwn: false, canReact: false },
        want: ["Reply"],
    },
    {
        name: "their own message while its editor is already open",
        props: { ...everythingWired, isOwn: true, editing: true, canPin: true, canModerate: true },
        want: ["React", "Reply", "Pin message", "Delete message"],
    },
    {
        name: "a timed out member looking at their own message",
        props: { ...everythingWired, isOwn: true, canReact: false, canEdit: false },
        want: ["Reply", "Delete message"],
    },
    {
        name: "a live stream panel that wired only replying and editing",
        props: { isOwn: true, onReply: vi.fn(), onEditStart: vi.fn(), onEdit: () => Promise.resolve() },
        want: ["Reply", "Edit message"],
    },
    {
        name: "a reader the caller gave nothing to do",
        props: { isOwn: false },
        want: [],
    },
];

describe("MessageBubble, one action source behind both surfaces", () => {
    for (const parityCase of parityCases) {
        it(`offers the right click the same actions as the hover bar for ${parityCase.name}`, () => {
            // given
            renderWithProviders(
                <MessageBubble message={parityCase.message ?? makeChatMessage()} {...parityCase.props} />,
            );
            const hovered = hoverBarLabels();

            // when
            fireEvent.contextMenu(bubbleOf());

            // then
            expect(hovered).toEqual(parityCase.want);
            expect(menuLabels()).toEqual(hovered);
        });
    }
});

type HandlerName = "onReply" | "onReactionToggle" | "onPinToggle" | "onDelete" | "onEditStart";

const HANDLER_NAMES: HandlerName[] = ["onReply", "onReactionToggle", "onPinToggle", "onDelete", "onEditStart"];

interface DispatchCase {
    name: string;
    label: string;
    handler: HandlerName;
}

const dispatchCases: DispatchCase[] = [
    { name: "replies", label: "Reply", handler: "onReply" },
    { name: "pins", label: "Pin message", handler: "onPinToggle" },
    { name: "starts an edit", label: "Edit message", handler: "onEditStart" },
    { name: "deletes", label: "Delete message", handler: "onDelete" },
];

describe("MessageBubble context menu dispatch", () => {
    for (const dispatchCase of dispatchCases) {
        it(`${dispatchCase.name} once and does nothing else when that item is chosen`, async () => {
            // given
            const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
            const user = userEvent.setup();
            const wired = renderFullyWired();
            fireEvent.contextMenu(bodyOf());

            // when
            await user.click(screen.getByRole("menuitem", { name: dispatchCase.label }));

            // then
            expect(wired[dispatchCase.handler]).toHaveBeenCalledExactlyOnceWith(wired.message);
            for (const other of HANDLER_NAMES.filter(name => name !== dispatchCase.handler)) {
                expect(wired[other]).not.toHaveBeenCalled();
            }
            expect(screen.queryByRole("menu")).not.toBeInTheDocument();
            confirm.mockRestore();
        });
    }
});

describe("MessageBubble context menu without a mouse", () => {
    it("opens from a keyboard menu key, which carries no pointer position", () => {
        // given
        renderFullyWired();
        const opener = screen.getByRole("button", { name: "React" });
        opener.focus();

        // when
        fireEvent.contextMenu(opener);

        // then
        expect(screen.getByRole("menu", { name: "Message actions" })).toBeInTheDocument();
        expect(screen.getByRole("menuitem", { name: "React" })).toHaveFocus();
    });

    it("runs the action a keyboard walked to", async () => {
        // given
        const user = userEvent.setup();
        const { message, onReply } = renderFullyWired();
        const opener = screen.getByRole("button", { name: "React" });
        opener.focus();
        fireEvent.contextMenu(opener);

        // when
        await user.keyboard("{ArrowDown}{Enter}");

        // then
        expect(onReply).toHaveBeenCalledExactlyOnceWith(message);
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("gives the message its focus back when Tab dismisses the menu", async () => {
        // given
        const user = userEvent.setup();
        const { onReply } = renderFullyWired();
        const opener = screen.getByRole("button", { name: "React" });
        opener.focus();
        fireEvent.contextMenu(opener);

        // when
        await user.tab();

        // then
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "React" })).toHaveFocus();
        expect(onReply).not.toHaveBeenCalled();
    });
});

interface FocusHandoverCase {
    name: string;
    label: string;
}

const focusHandoverCases: FocusHandoverCase[] = [
    { name: "replying", label: "Reply" },
    { name: "pinning", label: "Pin message" },
    { name: "declining a delete", label: "Delete message" },
];

describe("MessageBubble context menu focus handover", () => {
    for (const handoverCase of focusHandoverCases) {
        it(`never drops a keyboard reader onto the page body after ${handoverCase.name}`, async () => {
            // given
            const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
            const user = userEvent.setup();
            renderFullyWired();
            const opener = screen.getByRole("button", { name: "React" });
            opener.focus();
            fireEvent.contextMenu(opener);

            // when
            await user.click(screen.getByRole("menuitem", { name: handoverCase.label }));

            // then
            expect(document.activeElement).not.toBe(document.body);
            expect(bubbleOf().contains(document.activeElement)).toBe(true);
            confirm.mockRestore();
        });
    }

    it("leaves the caret in the editor rather than snatching it back after editing starts", async () => {
        // given
        const user = userEvent.setup();
        const message = makeChatMessage();
        const { rerender } = renderWithProviders(
            <MessageBubble message={message} isOwn onEdit={() => Promise.resolve()} onEditStart={vi.fn()} />,
        );
        fireEvent.contextMenu(bodyOf());

        // when
        await user.click(screen.getByRole("menuitem", { name: "Edit message" }));
        rerender(<MessageBubble message={message} isOwn editing onEdit={() => Promise.resolve()} />);

        // then
        expect(screen.getByRole("textbox")).toHaveFocus();
    });

    it("leaves the emoji picker holding focus rather than snatching it back", async () => {
        // given
        const user = userEvent.setup();
        renderFullyWired();
        fireEvent.contextMenu(bodyOf());

        // when
        await user.click(screen.getByRole("menuitem", { name: "React" }));

        // then
        expect(screen.getByRole("button", { name: "choose star" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "React" })).not.toHaveFocus();
    });
});
