import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatMessage, ReactionGroup } from "../../../types/api";
import { MessageBubble } from "./MessageBubble";

const HEART = "❤";
const STAR = "⭐";
const BODY = "the golden truth";

const EVERY_ACTION = ["React", "Reply", "Pin message", "Edit message", "Delete message"];

const ALL_LABELS = ["React", "Reply", "Pin message", "Unpin message", "Edit message", "Delete message"];

type HandlerName = "onReply" | "onReactionToggle" | "onPinToggle" | "onDelete" | "onEditStart";

const HANDLER_NAMES: HandlerName[] = ["onReply", "onReactionToggle", "onPinToggle", "onDelete", "onEditStart"];

vi.mock("../EmojiPicker/EmojiPicker", () => ({
    EmojiPicker: ({ onPick, onClose }: { onPick: (emoji: string) => void; onClose: () => void }) => (
        <div>
            <button onClick={() => onPick("⭐")}>choose star</button>
            <button onClick={onClose}>dismiss emoji</button>
        </div>
    ),
}));

afterEach(() => {
    vi.restoreAllMocks();
});

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

function bubbleOf(): HTMLElement {
    const bubble = document.getElementById("chat-msg-m1");
    if (!bubble) {
        throw new Error("no message bubble");
    }

    return bubble;
}

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

interface FullyWiredBubble {
    message: ChatMessage;
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

interface SenderNameCase {
    name: string;
    overrides: Partial<ChatMessage>;
    shown: string;
    hidden: string[];
}

const senderNameCases: SenderNameCase[] = [
    { name: "their profile display name", overrides: {}, shown: "Beatrice", hidden: [] },
    {
        name: "their room nickname in place of the profile display name",
        overrides: { sender_nickname: "The Golden Witch" },
        shown: "The Golden Witch",
        hidden: ["Beatrice"],
    },
    {
        name: "their username when they have no display name",
        overrides: { sender: { id: "u1", username: "beatrice", display_name: "   " } },
        shown: "beatrice",
        hidden: [],
    },
];

interface ChipCase {
    name: string;
    reaction: Partial<ReactionGroup>;
    canReact: boolean;
    withHandler: boolean;
    chipEnabled: boolean;
    reactControl: boolean;
    title: string | null;
    toggles: boolean;
}

const chipCases: ChipCase[] = [
    {
        name: "a member whose list wired the handler",
        reaction: {},
        canReact: true,
        withHandler: true,
        chipEnabled: true,
        reactControl: true,
        title: "Click to react",
        toggles: true,
    },
    {
        name: "a member removing their own reaction",
        reaction: { viewer_reacted: true },
        canReact: true,
        withHandler: true,
        chipEnabled: true,
        reactControl: true,
        title: "Click to remove your reaction",
        toggles: true,
    },
    {
        name: "a list that forgot the handler",
        reaction: {},
        canReact: true,
        withHandler: false,
        chipEnabled: false,
        reactControl: false,
        title: null,
        toggles: false,
    },
    {
        name: "a timed out member",
        reaction: {},
        canReact: false,
        withHandler: true,
        chipEnabled: false,
        reactControl: false,
        title: "You are timed out",
        toggles: false,
    },
    {
        name: "a timed out member who can still see who reacted",
        reaction: { display_names: ["Battler"] },
        canReact: false,
        withHandler: true,
        chipEnabled: true,
        reactControl: false,
        title: "You are timed out",
        toggles: false,
    },
];

interface AbandonCase {
    name: string;
    button: "Save" | "Cancel" | null;
}

const abandonCases: AbandonCase[] = [
    { name: "Escape", button: null },
    { name: "saving an unchanged body", button: "Save" },
    { name: "the cancel control", button: "Cancel" },
];

describe("MessageBubble", () => {
    it.each(senderNameCases)("renders another person's message under $name", ({ overrides, shown, hidden }) => {
        // given
        const message = makeChatMessage(overrides);

        // when
        renderWithProviders(<MessageBubble message={message} isOwn={false} />);

        // then
        expect(screen.getByText(shown)).toBeInTheDocument();
        for (const absent of hidden) {
            expect(screen.queryByText(absent)).not.toBeInTheDocument();
        }
        expect(screen.getByText(BODY)).toBeInTheDocument();
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
        expect(screen.queryByText(BODY)).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Show" }));

        // then
        expect(screen.getByText(BODY)).toBeInTheDocument();
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

    it("shows the seen label and leaves an unedited message unmarked", () => {
        // given
        const seenLabel = "Seen by Battler";

        // when
        renderWithProviders(<MessageBubble message={makeChatMessage()} isOwn seenLabel={seenLabel} />);

        // then
        expect(screen.getByText(/Seen by Battler/)).toBeInTheDocument();
        expect(screen.queryByText("(edited)")).not.toBeInTheDocument();
    });

    it("shows the quoted message a reply was aimed at and scrolls to it when the quote is clicked", async () => {
        // given
        const scrollIntoView = vi.spyOn(Element.prototype, "scrollIntoView").mockImplementation(() => {});
        const user = userEvent.setup();
        const quoted = makeChatMessage({ id: "m0", body: "the earlier statement" });
        const reply = makeChatMessage({
            id: "m1",
            reply_to: { id: "m0", sender_id: "u2", sender_name: "Battler", body_preview: "an earlier claim" },
        });

        // when
        renderWithProviders(
            <>
                <MessageBubble message={quoted} isOwn={false} />
                <MessageBubble message={reply} isOwn={false} />
            </>,
        );

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.getByText("an earlier claim")).toBeInTheDocument();

        // when
        await user.click(screen.getByText("an earlier claim"));

        // then
        expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "smooth", block: "center" });
    });

    it("renders uncovered image and video attachments and opens the lightbox for an image", async () => {
        // given
        const onLightbox = vi.fn();
        const user = userEvent.setup();
        const message = makeChatMessage({
            media: [
                { id: 1, media_url: "https://cdn.example/photo.png", media_type: "image", sort_order: 0 },
                { id: 2, media_url: "https://cdn.example/clip.mp4", media_type: "video", sort_order: 1 },
            ],
        });

        // when
        const { container } = renderWithProviders(
            <MessageBubble message={message} isOwn={false} onLightbox={onLightbox} />,
        );

        // then
        expect(screen.queryByText("Spoiler")).not.toBeInTheDocument();

        // when
        const image = container.querySelector('img[src="https://cdn.example/photo.png"]');
        await user.click(image as Element);

        // then
        expect(container.querySelector('video[src="https://cdn.example/clip.mp4"]')).toBeInTheDocument();
        expect(onLightbox).toHaveBeenCalledWith("https://cdn.example/photo.png");
    });

    it("covers a spoiler attachment and keeps the first click away from the lightbox", async () => {
        // given an attachment the sender marked as a spoiler
        const onLightbox = vi.fn();
        const user = userEvent.setup();
        const message = makeChatMessage({
            media: [
                {
                    id: 1,
                    media_url: "https://cdn.example/ending.png",
                    media_type: "image",
                    sort_order: 0,
                    is_spoiler: true,
                },
            ],
        });
        const { container } = renderWithProviders(
            <MessageBubble message={message} isOwn={false} onLightbox={onLightbox} />,
        );
        const image = container.querySelector('img[src="https://cdn.example/ending.png"]') as Element;

        // when the viewer clicks it once
        await user.click(image);

        // then that click only revealed it
        expect(onLightbox).not.toHaveBeenCalled();

        // when they click again
        await user.click(image);

        // then it opens full size
        expect(onLightbox).toHaveBeenCalledWith("https://cdn.example/ending.png");
    });

    it("strips the controls from a spoiler video until it is revealed", async () => {
        // given a video marked as a spoiler
        const user = userEvent.setup();
        const message = makeChatMessage({
            media: [
                {
                    id: 1,
                    media_url: "https://cdn.example/ending.mp4",
                    media_type: "video",
                    sort_order: 0,
                    is_spoiler: true,
                },
            ],
        });
        const { container } = renderWithProviders(<MessageBubble message={message} isOwn={false} />);
        const video = container.querySelector('video[src="https://cdn.example/ending.mp4"]') as HTMLVideoElement;

        // then it cannot be played while covered
        expect(video.controls).toBe(false);
        expect(screen.getByText("Spoiler")).toBeInTheDocument();

        // when it is revealed
        await user.click(video);

        // then it becomes a normal player
        expect(video.controls).toBe(true);
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

    it("renders each reaction with its own count", () => {
        // given
        const reactions = [makeReaction({ emoji: HEART, count: 3 }), makeReaction({ emoji: STAR, count: 1 })];

        // when
        renderWithProviders(<MessageBubble message={makeChatMessage({ reactions })} isOwn={false} />);

        // then
        expect(chipFor(HEART)).toHaveTextContent("3");
        expect(chipFor(STAR)).toHaveTextContent("1");
    });

    it.each(chipCases)(
        "keeps the chip, its tooltip and the react control in agreement for $name",
        async ({ reaction, canReact, withHandler, chipEnabled, reactControl, title, toggles }) => {
            // given
            const onReactionToggle = vi.fn();
            const user = userEvent.setup();
            const message = makeChatMessage({ reactions: [makeReaction({ emoji: HEART, count: 3, ...reaction })] });

            // when
            renderWithProviders(
                <MessageBubble
                    message={message}
                    isOwn={false}
                    canReact={canReact}
                    onReactionToggle={withHandler ? onReactionToggle : undefined}
                />,
            );

            // then
            expect(chipFor(HEART).hasAttribute("disabled")).toBe(!chipEnabled);
            expect(Boolean(screen.queryByRole("button", { name: "React" }))).toBe(reactControl);

            // when
            await user.click(chipFor(HEART));

            // then
            expect(chipFor(HEART).getAttribute("title")).toBe(title);
            expect(onReactionToggle.mock.calls).toEqual(toggles ? [[message, HEART]] : []);
        },
    );

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

    it("lists who reacted on a right clicked chip instead of opening the message menu, until the page is clicked elsewhere", async () => {
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
        const popover = await screen.findByRole("dialog", { name: "Reactors" });
        expect(popover).toHaveTextContent("2 reacted");
        expect(popover).toHaveTextContent("Beatrice");
        expect(popover).toHaveTextContent("Battler");
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();

        // when
        fireEvent.mouseDown(document.body);

        // then
        await waitFor(() => expect(screen.queryByRole("dialog", { name: "Reactors" })).not.toBeInTheDocument());
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

    it("starts the editor from the existing body, blocks saving it emptied, and commits a new body on Enter before leaving edit mode", async () => {
        // given
        const onEdit = vi.fn(() => Promise.resolve());
        const onEditCancel = vi.fn();
        const user = userEvent.setup();
        const message = makeChatMessage();

        // when
        renderWithProviders(
            <MessageBubble message={message} isOwn editing onEdit={onEdit} onEditCancel={onEditCancel} />,
        );

        // then
        const editor = screen.getByRole("textbox");
        expect(editor).toHaveValue(BODY);
        expect(screen.getByText("Enter to save · Esc to cancel")).toBeInTheDocument();

        // when
        await user.clear(editor);

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();

        // when
        await user.type(editor, "the red truth");
        await user.keyboard("{Enter}");

        // then
        await waitFor(() => expect(onEdit).toHaveBeenCalledWith(message, "the red truth"));
        expect(onEditCancel).toHaveBeenCalled();
    });

    for (const abandonCase of abandonCases) {
        it(`abandons the edit without saving through ${abandonCase.name}`, async () => {
            // given
            const onEdit = vi.fn(() => Promise.resolve());
            const onEditCancel = vi.fn();
            const user = userEvent.setup();
            renderWithProviders(
                <MessageBubble message={makeChatMessage()} isOwn editing onEdit={onEdit} onEditCancel={onEditCancel} />,
            );

            // when
            if (abandonCase.button === null) {
                await user.type(screen.getByRole("textbox"), " and more");
                await user.keyboard("{Escape}");
            } else {
                await user.click(screen.getByRole("button", { name: abandonCase.button }));
            }

            // then
            expect(onEditCancel).toHaveBeenCalledOnce();
            expect(onEdit).not.toHaveBeenCalled();
        });
    }
});

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
    message?: ChatMessage;
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
        props: { ...everythingWired, isOwn: false, canPin: false, canModerate: false },
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
            const offersMenu = parityCase.want.length > 0;

            // when
            const notPrevented = fireEvent.contextMenu(bubbleOf());

            // then
            expect(hovered).toEqual(parityCase.want);
            expect(menuLabels()).toEqual(hovered);
            expect(notPrevented).toBe(!offersMenu);
            expect(screen.queryByRole("menu") !== null).toBe(offersMenu);
        });
    }
});

interface ActionSurface {
    name: string;
    role: "button" | "menuitem";
}

const actionSurfaces: ActionSurface[] = [
    { name: "hover bar", role: "button" },
    { name: "context menu", role: "menuitem" },
];

interface DispatchCase {
    name: string;
    label: string;
    answer: boolean;
    prompts: string[][];
    runs: HandlerName | null;
}

const dispatchCases: DispatchCase[] = [
    { name: "replies once and does nothing else", label: "Reply", answer: true, prompts: [], runs: "onReply" },
    { name: "pins once and does nothing else", label: "Pin message", answer: true, prompts: [], runs: "onPinToggle" },
    {
        name: "starts an edit once and does nothing else",
        label: "Edit message",
        answer: true,
        prompts: [],
        runs: "onEditStart",
    },
    {
        name: "asks, then deletes once and does nothing else",
        label: "Delete message",
        answer: true,
        prompts: [["Delete this message?"]],
        runs: "onDelete",
    },
    {
        name: "asks, then keeps the message when the delete is declined",
        label: "Delete message",
        answer: false,
        prompts: [["Delete this message?"]],
        runs: null,
    },
];

for (const surface of actionSurfaces) {
    describe(`MessageBubble ${surface.name} dispatch`, () => {
        for (const dispatchCase of dispatchCases) {
            it(`${dispatchCase.name} when that item is chosen`, async () => {
                // given
                const confirm = vi.spyOn(window, "confirm").mockReturnValue(dispatchCase.answer);
                const user = userEvent.setup();
                const wired = renderFullyWired();
                if (surface.role === "menuitem") {
                    fireEvent.contextMenu(screen.getByText(BODY));
                }

                // when
                await user.click(screen.getByRole(surface.role, { name: dispatchCase.label }));

                // then
                for (const handler of HANDLER_NAMES) {
                    expect(wired[handler].mock.calls).toEqual(handler === dispatchCase.runs ? [[wired.message]] : []);
                }
                expect(confirm.mock.calls).toEqual(dispatchCase.prompts);
                expect(screen.queryByRole("menu")).not.toBeInTheDocument();
            });
        }
    });
}

describe("MessageBubble context menu", () => {
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

    it("leaves the caret in the editor when editing starts from the menu", async () => {
        // given
        const user = userEvent.setup();
        const message = makeChatMessage();
        const onEditStart = vi.fn();
        const { rerender } = renderWithProviders(
            <MessageBubble message={message} isOwn onEdit={() => Promise.resolve()} onEditStart={onEditStart} />,
        );
        fireEvent.contextMenu(screen.getByText(BODY));

        // when
        await user.click(screen.getByRole("menuitem", { name: "Edit message" }));
        rerender(<MessageBubble message={message} isOwn editing onEdit={() => Promise.resolve()} />);

        // then
        expect(onEditStart).toHaveBeenCalledWith(message);
        expect(screen.getByRole("textbox")).toHaveFocus();
    });

    it("opens the emoji picker from the menu, leaves it holding focus, and forwards the emoji chosen", async () => {
        // given
        const user = userEvent.setup();
        const { message, onReactionToggle } = renderFullyWired();
        fireEvent.contextMenu(screen.getByText(BODY));

        // when
        await user.click(screen.getByRole("menuitem", { name: "React" }));

        // then
        expect(screen.getByRole("button", { name: "choose star" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "React" })).not.toHaveFocus();

        // when
        await user.click(screen.getByRole("button", { name: "choose star" }));

        // then
        expect(onReactionToggle).toHaveBeenCalledWith(message, STAR);
        expect(screen.queryByRole("button", { name: "choose star" })).not.toBeInTheDocument();
    });

    it("stays open while the message list scrolls underneath it and closes when the page is clicked elsewhere", async () => {
        // given
        renderFullyWired();
        fireEvent.contextMenu(screen.getByText(BODY));

        // when
        fireEvent.scroll(document, {});

        // then
        expect(screen.getByRole("menu", { name: "Message actions" })).toBeInTheDocument();

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
        fireEvent.pointerDown(screen.getByText(BODY), { pointerType: "touch", clientX: 40, clientY: 60 });
        act(() => {
            vi.advanceTimersByTime(450);
        });

        // then
        expect(menuLabels()).toEqual(EVERY_ACTION);
    });

    it("leaves a short touch alone", () => {
        // given
        vi.useFakeTimers();
        renderFullyWired();

        // when
        fireEvent.pointerDown(screen.getByText(BODY), { pointerType: "touch", clientX: 40, clientY: 60 });
        fireEvent.pointerUp(screen.getByText(BODY), { pointerType: "touch" });
        act(() => {
            vi.advanceTimersByTime(450);
        });

        // then
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });
});

interface DismissCase {
    name: string;
    keys: string;
}

const dismissCases: DismissCase[] = [
    { name: "Escape", keys: "{Escape}" },
    { name: "Tab", keys: "{Tab}" },
];

describe("MessageBubble context menu without a mouse", () => {
    it("opens from a keyboard menu key, which carries no pointer position, and runs the action a keyboard walked to", async () => {
        // given
        const user = userEvent.setup();
        const { message, onReply } = renderFullyWired();
        const opener = screen.getByRole("button", { name: "React" });
        opener.focus();

        // when
        fireEvent.contextMenu(opener);

        // then
        expect(screen.getByRole("menu", { name: "Message actions" })).toBeInTheDocument();
        expect(screen.getByRole("menuitem", { name: "React" })).toHaveFocus();

        // when
        await user.keyboard("{ArrowDown}{Enter}");

        // then
        expect(onReply).toHaveBeenCalledExactlyOnceWith(message);
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    for (const dismissCase of dismissCases) {
        it(`closes on ${dismissCase.name} and hands focus back to the message it came from`, async () => {
            // given
            const user = userEvent.setup();
            const { onReply } = renderFullyWired();
            fireEvent.contextMenu(screen.getByText(BODY));

            // when
            await user.keyboard(dismissCase.keys);

            // then
            expect(screen.queryByRole("menu")).not.toBeInTheDocument();
            expect(screen.getByRole("button", { name: "React" })).toHaveFocus();
            expect(bubbleOf().contains(document.activeElement)).toBe(true);
            expect(onReply).not.toHaveBeenCalled();
        });
    }
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
            vi.spyOn(window, "confirm").mockReturnValue(false);
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
        });
    }
});
