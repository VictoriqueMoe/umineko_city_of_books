import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../../../api/client";
import type { SiteInfo } from "../../../api/endpoints";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatMessage, ChatRoom } from "../../../types/api";
import { ChatComposer } from "./ChatComposer";

const mocks = vi.hoisted(() => ({
    sendChatMessage: vi.fn(),
    sendFirstDM: vi.fn(),
}));

vi.mock("../../../api/mutations/chat", () => ({
    useSendChatMessage: () => ({ mutateAsync: mocks.sendChatMessage }),
    useSendFirstDMMessage: () => ({ mutateAsync: mocks.sendFirstDM }),
}));

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

interface ComposerOptions {
    roomId?: string | null;
    draftRecipientId?: string | null;
    onSent?: (message: ChatMessage, room?: ChatRoom) => void;
    replyingTo?: { id: string; senderName: string; bodyPreview: string } | null;
    onCancelReply?: () => void;
    onTyping?: () => void;
    onEditLast?: () => void;
    timeoutUntil?: string;
    sendOnEnter?: boolean;
    compact?: boolean;
    siteInfo?: Partial<SiteInfo>;
}

function renderComposer(options: ComposerOptions = {}) {
    return renderWithProviders(
        <ChatComposer
            roomId={options.roomId === undefined ? "room-1" : options.roomId}
            draftRecipientId={options.draftRecipientId ?? null}
            onSent={options.onSent ?? (() => {})}
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
}

describe("ChatComposer", () => {
    beforeEach(() => {
        mocks.sendChatMessage.mockResolvedValue(makeMessage());
        mocks.sendFirstDM.mockResolvedValue({ message: makeMessage(), room: { id: "room-9" } });
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
        renderComposer();

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "   ");

        // then
        expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
        await user.keyboard("{Enter}");
        expect(mocks.sendChatMessage).not.toHaveBeenCalled();
    });

    it("sends the trimmed body when Enter is pressed", async () => {
        // given
        const onSent = vi.fn();
        const user = userEvent.setup();
        renderComposer({ onSent });

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "  without love it cannot be seen  ");
        await user.keyboard("{Enter}");

        // then
        await waitFor(() => expect(mocks.sendChatMessage).toHaveBeenCalledTimes(1));
        expect(mocks.sendChatMessage).toHaveBeenCalledWith({
            body: "without love it cannot be seen",
            reply_to_id: undefined,
            files: [],
        });
        expect(onSent).toHaveBeenCalledWith(makeMessage());
    });

    it("clears the composer once the message has been accepted", async () => {
        // given
        const user = userEvent.setup();
        renderComposer();
        const input = screen.getByPlaceholderText(ENTER_PLACEHOLDER);

        // when
        await user.type(input, "beehive");
        await user.keyboard("{Enter}");

        // then
        await waitFor(() => expect(input).toHaveValue(""));
    });

    it("inserts a newline instead of sending on Shift and Enter", async () => {
        // given
        const user = userEvent.setup();
        renderComposer();
        const input = screen.getByPlaceholderText(ENTER_PLACEHOLDER);

        // when
        await user.type(input, "first");
        await user.keyboard("{Shift>}{Enter}{/Shift}");
        await user.type(input, "second");

        // then
        expect(mocks.sendChatMessage).not.toHaveBeenCalled();
        expect(input).toHaveValue("first\nsecond");
    });

    it("leaves Enter alone when sending on Enter is switched off", async () => {
        // given
        const user = userEvent.setup();
        renderComposer({ sendOnEnter: false });
        const input = screen.getByPlaceholderText("Type a message...");

        // when
        await user.type(input, "a note");
        await user.keyboard("{Enter}");

        // then
        expect(mocks.sendChatMessage).not.toHaveBeenCalled();
        expect(input).toHaveValue("a note\n");
    });

    it("attaches the reply target to the outgoing message and clears the reply", async () => {
        // given
        const onCancelReply = vi.fn();
        const user = userEvent.setup();
        renderComposer({
            replyingTo: { id: "m-earlier", senderName: "Battler", bodyPreview: "an earlier claim" },
            onCancelReply,
        });

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "I deny it");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        await waitFor(() => expect(mocks.sendChatMessage).toHaveBeenCalledTimes(1));
        expect(mocks.sendChatMessage).toHaveBeenCalledWith({
            body: "I deny it",
            reply_to_id: "m-earlier",
            files: [],
        });
        expect(onCancelReply).toHaveBeenCalled();
    });

    it("shows who is being replied to and lets the reply be abandoned", async () => {
        // given
        const onCancelReply = vi.fn();
        const user = userEvent.setup();
        renderComposer({
            replyingTo: { id: "m-earlier", senderName: "Battler", bodyPreview: "an earlier claim" },
            onCancelReply,
        });

        // when
        await user.click(screen.getByRole("button", { name: "Cancel reply" }));

        // then
        expect(screen.getByText("Replying to Battler")).toBeInTheDocument();
        expect(screen.getByText("an earlier claim")).toBeInTheDocument();
        expect(onCancelReply).toHaveBeenCalledOnce();
    });

    it("opens the conversation through the first direct message when no room exists yet", async () => {
        // given
        const onSent = vi.fn();
        const user = userEvent.setup();
        renderComposer({ roomId: null, draftRecipientId: "u-battler", onSent });

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "are you there");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        await waitFor(() => expect(mocks.sendFirstDM).toHaveBeenCalledTimes(1));
        expect(mocks.sendFirstDM).toHaveBeenCalledWith({
            recipientId: "u-battler",
            body: "are you there",
            files: [],
        });
        expect(onSent).toHaveBeenCalledWith(makeMessage(), { id: "room-9" });
        expect(mocks.sendChatMessage).not.toHaveBeenCalled();
    });

    it("sends nothing when there is neither a room nor a recipient", async () => {
        // given
        const user = userEvent.setup();
        renderComposer({ roomId: null, draftRecipientId: null });

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "into the void");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(mocks.sendChatMessage).not.toHaveBeenCalled();
        expect(mocks.sendFirstDM).not.toHaveBeenCalled();
    });

    it("lets an attachment stand in for a body and sends it with the message", async () => {
        // given
        const user = userEvent.setup();
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const { container } = renderComposer();
        expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();

        // when
        await user.upload(container.querySelector('input[type="file"]') as HTMLInputElement, file);
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        await waitFor(() => expect(mocks.sendChatMessage).toHaveBeenCalledTimes(1));
        expect(mocks.sendChatMessage).toHaveBeenCalledWith({ body: "", reply_to_id: undefined, files: [file] });
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

    it("explains a banned word rejection including the kick", async () => {
        // given
        const user = userEvent.setup();
        mocks.sendChatMessage.mockRejectedValue(
            new ApiError(422, "blocked", { code: "banned_word", pattern: "goats", action: "kick" }),
        );
        renderComposer();

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "goats");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(
            await screen.findByText(
                'Message blocked by banned-word rule "goats". You have been kicked from this room.',
            ),
        ).toBeInTheDocument();
    });

    it("omits the kick notice when the rule only blocks the message", async () => {
        // given
        const user = userEvent.setup();
        mocks.sendChatMessage.mockRejectedValue(
            new ApiError(422, "blocked", { code: "banned_word", pattern: "goats" }),
        );
        renderComposer();

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "goats");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(await screen.findByText('Message blocked by banned-word rule "goats".')).toBeInTheDocument();
    });

    it("surfaces the error field the server returned", async () => {
        // given
        const user = userEvent.setup();
        mocks.sendChatMessage.mockRejectedValue(new ApiError(403, "nope", { error: "You are muted in this room" }));
        renderComposer();

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "hello");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(await screen.findByText("You are muted in this room")).toBeInTheDocument();
    });

    it("falls back to the thrown error message for a plain failure", async () => {
        // given
        const user = userEvent.setup();
        mocks.sendChatMessage.mockRejectedValue(new Error("network is down"));
        renderComposer();

        // when
        await user.type(screen.getByPlaceholderText(ENTER_PLACEHOLDER), "hello");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(await screen.findByText("network is down")).toBeInTheDocument();
        expect(screen.getByPlaceholderText(ENTER_PLACEHOLDER)).toHaveValue("hello");
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
            replyingTo: { id: "m-earlier", senderName: "Battler", bodyPreview: "an earlier claim" },
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

    it("sends the chosen GIF url as the message body", async () => {
        // given
        const user = userEvent.setup();
        renderComposer();

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "choose gif" }));

        // then
        await waitFor(() => expect(mocks.sendChatMessage).toHaveBeenCalledTimes(1));
        expect(mocks.sendChatMessage).toHaveBeenCalledWith({
            body: "https://media.giphy.com/media/g1/giphy.gif",
            reply_to_id: undefined,
        });
        expect(screen.queryByRole("button", { name: "choose gif" })).not.toBeInTheDocument();
    });

    it("reports a failure to send the chosen GIF", async () => {
        // given
        const user = userEvent.setup();
        mocks.sendChatMessage.mockRejectedValue(new Error("giphy is unreachable"));
        renderComposer();

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "choose gif" }));

        // then
        expect(await screen.findByText("giphy is unreachable")).toBeInTheDocument();
    });
});
