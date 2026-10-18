import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import type { OutgoingChatMessage } from "../../../domain/chat/optimisticSend";
import type { ChatSender, DraftSender } from "../../../hooks/chat/useOptimisticSends";
import { renderWithProviders } from "../../../test-utils/render";
import type { SiteInfo } from "../../../types/api";
import { ChatComposer, type ComposerTarget } from "./ChatComposer";

const mocks = vi.hoisted(() => ({
    useIsMobile: vi.fn(),
}));

vi.mock("../../../hooks/useIsMobile", () => ({ useIsMobile: mocks.useIsMobile }));

vi.mock("../GifPicker/GifPicker", () => ({
    GifPicker: ({ onPick, onClose }: { onPick: (gif: { id: string; url: string }) => void; onClose: () => void }) => (
        <div>
            <button onClick={() => onPick({ id: "g1", url: "https://media.giphy.com/media/g1/giphy.gif" })}>
                choose gif
            </button>
            <button onClick={onClose}>dismiss gifs</button>
        </div>
    ),
}));

const ENTER_PLACEHOLDER = "Type a message... (Enter to send, Shift+Enter for newline)";

interface ComposerOptions {
    target?: ComposerTarget | null;
    replyingTo?: { id: string; senderName: string; body: string } | null;
    onCancelReply?: () => void;
    onTyping?: () => void;
    onEditLast?: () => void;
    timeoutUntil?: string;
    sendOnEnter?: boolean;
    compact?: boolean;
    siteInfo?: Partial<SiteInfo>;
}

type SendMock = Mock<(outgoing: OutgoingChatMessage) => void>;

function makeSender(roomId = "room-1", adoptedFrom?: string): ChatSender & { send: SendMock } {
    return { roomId, adoptedFrom, send: vi.fn<(outgoing: OutgoingChatMessage) => void>() };
}

function makeDraftSender(recipientId = "u-battler"): DraftSender & { send: SendMock } {
    return { recipientId, send: vi.fn<(outgoing: OutgoingChatMessage) => void>() };
}

function renderComposer(options: ComposerOptions = {}) {
    const sender = makeSender();
    const target = options.target === undefined ? { kind: "room" as const, sender } : options.target;
    const rendered = renderWithProviders(
        <ChatComposer
            target={target}
            replyingTo={options.replyingTo}
            onCancelReply={options.onCancelReply}
            onTyping={options.onTyping}
            onEditLast={options.onEditLast}
            timeoutUntil={options.timeoutUntil}
            sendOnEnter={options.sendOnEnter}
            compact={options.compact}
        />,
        { siteInfo: options.siteInfo },
    );

    return { ...rendered, sender };
}

describe("ChatComposer in a room", () => {
    beforeEach(() => {
        mocks.useIsMobile.mockReturnValue(false);
    });

    it("keeps the send control disabled until there is something to send", async () => {
        // given
        const user = userEvent.setup();
        renderComposer();

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "hello");

        // then
        expect(screen.getByRole("button", { name: "Send" })).toBeEnabled();
    });

    it("refuses to send a body that is only whitespace", async () => {
        // given
        const user = userEvent.setup();
        const { sender } = renderComposer();

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "   ");

        // then
        expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
        await user.keyboard("{Enter}");
        expect(sender.send).not.toHaveBeenCalled();
    });

    it("hands the trimmed body to the room's sender when Enter is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { sender } = renderComposer();

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "  without love it cannot be seen  ");
        await user.keyboard("{Enter}");

        // then
        expect(sender.send).toHaveBeenCalledExactlyOnceWith({
            body: "without love it cannot be seen",
            replyTo: null,
            attachments: [],
        });
    });

    it("clears the composer the moment the message is handed over, without waiting for the server", async () => {
        // given
        const user = userEvent.setup();
        renderComposer();
        const input = screen.getByPlaceholderText(ENTER_PLACEHOLDER);

        // when
        await user.type(input, "beehive");
        await user.keyboard("{Enter}");

        // then
        expect(input).toHaveValue("");
    });

    it("never locks the composer, so a second message can follow the first straight away", async () => {
        // given
        const user = userEvent.setup();
        const { sender } = renderComposer();
        const input = screen.getByPlaceholderText(ENTER_PLACEHOLDER);
        await user.type(input, "hi");
        await user.keyboard("{Enter}");

        // when
        await user.type(input, "you there?");

        // then
        expect(screen.getByRole("button", { name: "Send" })).toBeEnabled();
        await user.keyboard("{Enter}");
        expect(sender.send).toHaveBeenCalledTimes(2);
        expect(sender.send).toHaveBeenLastCalledWith({ body: "you there?", replyTo: null, attachments: [] });
        expect(screen.getByRole("button", { name: "+ GIF" })).toBeEnabled();
    });

    it("inserts a newline instead of sending on Shift and Enter", async () => {
        // given
        const user = userEvent.setup();
        const { sender } = renderComposer();
        const input = screen.getByPlaceholderText(ENTER_PLACEHOLDER);

        // when
        await user.type(input, "first");
        await user.keyboard("{Shift>}{Enter}{/Shift}");
        await user.type(input, "second");

        // then
        expect(sender.send).not.toHaveBeenCalled();
        expect(input).toHaveValue("first\nsecond");
    });

    it("leaves Enter alone when sending on Enter is switched off", async () => {
        // given
        const user = userEvent.setup();
        const { sender } = renderComposer({ sendOnEnter: false });
        const input = screen.getByPlaceholderText("Type a message...");

        // when
        await user.type(input, "a note");
        await user.keyboard("{Enter}");

        // then
        expect(sender.send).not.toHaveBeenCalled();
        expect(input).toHaveValue("a note\n");
    });

    it("drops the keyboard hint from the placeholder on a phone", () => {
        // given
        mocks.useIsMobile.mockReturnValue(true);

        // when
        renderComposer();

        // then
        expect(screen.getByPlaceholderText("Type a message...")).toBeInTheDocument();
        expect(screen.queryByPlaceholderText(ENTER_PLACEHOLDER)).not.toBeInTheDocument();
    });

    it("hands the reply target over with the message and clears the reply", async () => {
        // given
        const onCancelReply = vi.fn();
        const user = userEvent.setup();
        const replyingTo = { id: "m-earlier", senderName: "Battler", body: "an earlier claim" };
        const { sender } = renderComposer({ replyingTo, onCancelReply });

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "I deny it");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(sender.send).toHaveBeenCalledExactlyOnceWith({
            body: "I deny it",
            replyTo: replyingTo,
            attachments: [],
        });
        expect(onCancelReply).toHaveBeenCalled();
    });

    it("shows who is being replied to and lets the reply be abandoned", async () => {
        // given
        const onCancelReply = vi.fn();
        const user = userEvent.setup();
        renderComposer({
            replyingTo: { id: "m-earlier", senderName: "Battler", body: "an earlier claim" },
            onCancelReply,
        });

        // when
        await user.click(screen.getByRole("button", { name: "Cancel reply" }));

        // then
        expect(screen.getByText(/Replying to/)).toHaveTextContent("Replying to Battler");
        expect(screen.getByText("an earlier claim")).toBeInTheDocument();
        expect(onCancelReply).toHaveBeenCalledOnce();
    });

    it("keeps a spoiler in the message being replied to hidden and renders its formatting", () => {
        // given
        const body = "||Does that mean [purple]Lambdadelta[/purple] is jealous?||";

        // when
        renderComposer({ replyingTo: { id: "m-earlier", senderName: "Battler", body } });

        // then
        expect(screen.getByTitle("Hover to reveal")).toHaveTextContent("Does that mean Lambdadelta is jealous?");
        expect(screen.queryByText(/\|\||\[purple\]/)).not.toBeInTheDocument();
    });

    it("hands over an attachment marked as a spoiler with its flag set", async () => {
        // given a picked image the sender has marked as a spoiler
        const user = userEvent.setup();
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const { container, sender } = renderComposer();
        await user.upload(container.querySelector('input[type="file"]') as HTMLInputElement, file);

        // when
        await user.click(screen.getByRole("button", { name: "Mark as spoiler" }));
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(sender.send).toHaveBeenCalledExactlyOnceWith({
            body: "",
            replyTo: null,
            attachments: [{ file, spoiler: true }],
        });
    });

    it("keeps the spoiler flag with its own attachment when an earlier one is removed", async () => {
        // given two attachments where only the second is a spoiler
        const user = userEvent.setup();
        const first = new File(["one"], "one.png", { type: "image/png" });
        const second = new File(["two"], "two.png", { type: "image/png" });
        const { container, sender } = renderComposer();
        await user.upload(container.querySelector('input[type="file"]') as HTMLInputElement, [first, second]);
        await user.click(screen.getAllByRole("button", { name: "Mark as spoiler" })[1]);

        // when the first attachment is removed
        await user.click(screen.getAllByRole("button", { name: "Remove" })[0]);
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then the flag has moved down with its own file rather than staying on index 1
        expect(sender.send).toHaveBeenCalledExactlyOnceWith({
            body: "",
            replyTo: null,
            attachments: [{ file: second, spoiler: true }],
        });
    });

    it("lets an attachment stand in for a body and clears it once handed over", async () => {
        // given
        const user = userEvent.setup();
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const { container, sender } = renderComposer();
        expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();

        // when
        await user.upload(container.querySelector('input[type="file"]') as HTMLInputElement, file);
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(sender.send).toHaveBeenCalledExactlyOnceWith({
            body: "",
            replyTo: null,
            attachments: [{ file, spoiler: false }],
        });
        expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
    });

    it("drops an attachment again when its remove control is used", async () => {
        // given
        const user = userEvent.setup();
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const { container } = renderComposer();
        await user.upload(container.querySelector('input[type="file"]') as HTMLInputElement, file);
        expect(screen.getByRole("button", { name: "Send" })).toBeEnabled();

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
    });

    it("refuses an attachment that exceeds the site image limit", async () => {
        // given
        const user = userEvent.setup();
        const file = new File([new Uint8Array(64)], "huge.png", { type: "image/png" });
        const { container } = renderComposer({ siteInfo: { max_image_size: 8 } });

        // when
        await user.upload(container.querySelector('input[type="file"]') as HTMLInputElement, file);

        // then
        expect(await screen.findByText(/huge\.png is too large/)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
    });

    it("replaces the whole composer with a notice while the sender is timed out", () => {
        // given
        const timeoutUntil = new Date(Date.now() + 60_000).toISOString();

        // when
        renderComposer({ timeoutUntil });

        // then
        expect(screen.getByText(/You are timed out until/)).toBeInTheDocument();
        expect(screen.queryByPlaceholderText(ENTER_PLACEHOLDER)).not.toBeInTheDocument();
    });

    it("restores the composer once the timeout has already lapsed", () => {
        // given
        const timeoutUntil = new Date(Date.now() - 60_000).toISOString();

        // when
        renderComposer({ timeoutUntil });

        // then
        expect(screen.queryByText(/You are timed out until/)).not.toBeInTheDocument();
        expect(screen.getByPlaceholderText(ENTER_PLACEHOLDER)).toBeInTheDocument();
    });

    it("throttles the typing notification to once per burst of keystrokes", async () => {
        // given
        const onTyping = vi.fn();
        const user = userEvent.setup();
        renderComposer({ onTyping });

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "typing away");

        // then
        expect(onTyping).toHaveBeenCalledTimes(1);
    });

    it("recalls the previous message when ArrowUp is pressed in an empty composer", async () => {
        // given
        const onEditLast = vi.fn();
        const user = userEvent.setup();
        renderComposer({ onEditLast });

        // when
        await user.click(screen.getByPlaceholderText(ENTER_PLACEHOLDER));
        await user.keyboard("{ArrowUp}");

        // then
        expect(onEditLast).toHaveBeenCalledOnce();
    });

    it("leaves ArrowUp alone once the composer has text in it", async () => {
        // given
        const onEditLast = vi.fn();
        const user = userEvent.setup();
        renderComposer({ onEditLast });

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "half written");
        await user.keyboard("{ArrowUp}");

        // then
        expect(onEditLast).not.toHaveBeenCalled();
    });

    it("leaves ArrowUp alone while a reply is being composed", async () => {
        // given
        const onEditLast = vi.fn();
        const user = userEvent.setup();
        renderComposer({
            onEditLast,
            replyingTo: { id: "m-earlier", senderName: "Battler", body: "an earlier claim" },
        });

        // when
        await user.click(screen.getByPlaceholderText(ENTER_PLACEHOLDER));
        await user.keyboard("{ArrowUp}");

        // then
        expect(onEditLast).not.toHaveBeenCalled();
    });

    it("hides the extra actions behind a more options control in compact mode", async () => {
        // given
        const user = userEvent.setup();
        renderComposer({ compact: true });
        expect(screen.queryByRole("button", { name: "+ GIF" })).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "More options" }));

        // then
        expect(screen.getByRole("button", { name: "+ GIF" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "+ Media" })).toBeInTheDocument();
    });

    it("hands the chosen GIF url to the room's sender as the message body", async () => {
        // given
        const user = userEvent.setup();
        const { sender } = renderComposer();

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "choose gif" }));

        // then
        expect(sender.send).toHaveBeenCalledExactlyOnceWith({
            body: "https://media.giphy.com/media/g1/giphy.gif",
            replyTo: null,
            attachments: [],
        });
        expect(screen.queryByRole("button", { name: "choose gif" })).not.toBeInTheDocument();
    });

    it("leaves an unsent draft behind when the reader opens another conversation", async () => {
        // given a message typed but not sent
        const user = userEvent.setup();
        const { rerender } = renderComposer();
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "meet me at the rose garden");

        // when the reader switches to a different conversation
        rerender(<ChatComposer target={{ kind: "room", sender: makeSender("room-2") }} />);

        // then the draft does not follow them into it
        expect(screen.getByPlaceholderText(ENTER_PLACEHOLDER)).toHaveValue("");
    });

    it("keeps what was typed when nothing can be sent because there is neither a room nor a recipient", async () => {
        // given
        const user = userEvent.setup();
        renderComposer({ target: null });

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "into the void");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(screen.getByPlaceholderText(ENTER_PLACEHOLDER)).toHaveValue("into the void");
    });
});

describe("ChatComposer for a first direct message", () => {
    beforeEach(() => {
        mocks.useIsMobile.mockReturnValue(false);
    });

    it("hands the first message to the draft's sender and clears at once, without waiting for the conversation", async () => {
        // given
        const sender = makeDraftSender();
        const user = userEvent.setup();
        renderComposer({ target: { kind: "draft", sender } });
        const input = screen.getByPlaceholderText(ENTER_PLACEHOLDER);

        // when
        await user.type(input, "  are you there  ");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(sender.send).toHaveBeenCalledExactlyOnceWith({ body: "are you there", replyTo: null, attachments: [] });
        expect(input).toHaveValue("");
        expect(screen.queryByRole("button", { name: "..." })).not.toBeInTheDocument();
    });

    it("never locks the composer, so a second message can follow the first before the conversation exists", async () => {
        // given
        const sender = makeDraftSender();
        const user = userEvent.setup();
        renderComposer({ target: { kind: "draft", sender } });
        const input = screen.getByPlaceholderText(ENTER_PLACEHOLDER);
        await user.type(input, "hi");
        await user.keyboard("{Enter}");

        // when
        await user.type(input, "you there?");
        await user.keyboard("{Enter}");

        // then
        expect(sender.send.mock.calls.map(([outgoing]) => outgoing.body)).toEqual(["hi", "you there?"]);
        expect(screen.getByRole("button", { name: "+ GIF" })).toBeEnabled();
    });

    it("hands the chosen GIF to the draft's sender as the first message", async () => {
        // given
        const sender = makeDraftSender();
        const user = userEvent.setup();
        renderComposer({ target: { kind: "draft", sender } });

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "choose gif" }));

        // then
        expect(sender.send).toHaveBeenCalledExactlyOnceWith({
            body: "https://media.giphy.com/media/g1/giphy.gif",
            replyTo: null,
            attachments: [],
        });
    });

    it("keeps a half-typed message when the draft becomes the room it opened", async () => {
        // given a message being typed while the first one opens the conversation
        const user = userEvent.setup();
        const { rerender } = renderComposer({ target: { kind: "draft", sender: makeDraftSender("u-battler") } });
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "and another thing");

        // when the conversation lands as a room adopted from that draft
        rerender(<ChatComposer target={{ kind: "room", sender: makeSender("room-9", "u-battler") }} />);

        // then
        expect(screen.getByPlaceholderText(ENTER_PLACEHOLDER)).toHaveValue("and another thing");
    });

    it("leaves the draft's text behind when the reader opens an unrelated room instead", async () => {
        // given
        const user = userEvent.setup();
        const { rerender } = renderComposer({ target: { kind: "draft", sender: makeDraftSender("u-battler") } });
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "for battler only");

        // when
        rerender(<ChatComposer target={{ kind: "room", sender: makeSender("room-2") }} />);

        // then
        expect(screen.getByPlaceholderText(ENTER_PLACEHOLDER)).toHaveValue("");
    });
});
