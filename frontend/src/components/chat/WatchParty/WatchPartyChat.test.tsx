import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { User, WatchPartyMessage } from "../../../types/api";
import { WatchPartyChat } from "./WatchPartyChat";

interface NodeProcess {
    on(event: "unhandledRejection", handler: (reason: unknown) => void): void;
    off(event: "unhandledRejection", handler: (reason: unknown) => void): void;
}

const nodeProcess = (globalThis as unknown as { process: NodeProcess }).process;

const viewerId = "user-viewer";

function makeChatUser(overrides: Partial<User> = {}): User {
    return { id: viewerId, username: "beatrice", display_name: "Beatrice", ...overrides };
}

function makeMessage(overrides: Partial<WatchPartyMessage> = {}): WatchPartyMessage {
    return {
        id: "msg-1",
        session_id: "session-1",
        kind: "user",
        sender: makeChatUser(),
        body: "without love it cannot be seen",
        created_at: "2026-08-01T10:05:00Z",
        ...overrides,
    };
}

interface ChatOptions {
    messages?: WatchPartyMessage[];
    onSend?: (body: string) => Promise<void>;
}

function renderChat(options: ChatOptions = {}) {
    const onSend = vi.fn(options.onSend ?? (() => Promise.resolve()));
    const result = renderWithProviders(
        <WatchPartyChat messages={options.messages ?? []} viewerUserId={viewerId} onSend={onSend} />,
    );

    return { ...result, onSend };
}

describe("WatchPartyChat", () => {
    it("invites the first message when the party chat is empty", () => {
        // given
        const messages: WatchPartyMessage[] = [];

        // when
        renderChat({ messages });

        // then
        expect(screen.getByText("No messages yet. Say hi.")).toBeInTheDocument();
    });

    it("shows who wrote each message and what they said", () => {
        // given
        const messages = [makeMessage({ body: "the golden truth" })];

        // when
        renderChat({ messages });

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
        expect(screen.queryByText("No messages yet. Say hi.")).not.toBeInTheDocument();
    });

    it("shows a system notice without attributing it to anybody", () => {
        // given
        const messages = [makeMessage({ id: "sys-1", kind: "system", sender: undefined, body: "Battler joined" })];

        // when
        renderChat({ messages });

        // then
        expect(screen.getByText("Battler joined")).toBeInTheDocument();
        expect(screen.queryByText("Beatrice")).not.toBeInTheDocument();
    });

    it("skips a message that has lost its sender", () => {
        // given
        const messages = [
            makeMessage({ id: "orphan", sender: undefined, body: "an orphaned line" }),
            makeMessage({ id: "msg-2", body: "a line with an author" }),
        ];

        // when
        renderChat({ messages });

        // then
        expect(screen.queryByText("an orphaned line")).not.toBeInTheDocument();
        expect(screen.getByText("a line with an author")).toBeInTheDocument();
    });

    it("keeps the send control unusable until something has been typed", async () => {
        // given
        const user = userEvent.setup();
        renderChat();
        const disabledWhileEmpty = (screen.getByRole("button", { name: "Send" }) as HTMLButtonElement).disabled;

        // when
        await user.type(screen.getByPlaceholderText("Type a message..."), "kihihi");

        // then
        expect(disabledWhileEmpty).toBe(true);
        expect(screen.getByRole("button", { name: "Send" })).toBeEnabled();
    });

    it("treats a draft of only whitespace as nothing to send", async () => {
        // given
        const user = userEvent.setup();
        const { onSend } = renderChat();

        // when
        await user.type(screen.getByPlaceholderText("Type a message..."), "   ");

        // then
        expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
        expect(onSend).not.toHaveBeenCalled();
    });

    it("sends the trimmed draft and empties the composer", async () => {
        // given
        const user = userEvent.setup();
        const { onSend } = renderChat();

        // when
        await user.type(screen.getByPlaceholderText("Type a message..."), "  kihihi  ");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(onSend).toHaveBeenCalledWith("kihihi");
        await waitFor(() => {
            expect(screen.getByPlaceholderText("Type a message...")).toHaveValue("");
        });
    });

    it("sends the draft when Enter is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { onSend } = renderChat();

        // when
        await user.type(screen.getByPlaceholderText("Type a message..."), "kihihi");
        await user.keyboard("{Enter}");

        // then
        expect(onSend).toHaveBeenCalledWith("kihihi");
    });

    it("adds a newline instead of sending on Shift and Enter", async () => {
        // given
        const user = userEvent.setup();
        const { onSend } = renderChat();
        const composer = screen.getByPlaceholderText("Type a message...");

        // when
        await user.type(composer, "first");
        await user.keyboard("{Shift>}{Enter}{/Shift}");
        await user.type(composer, "second");

        // then
        expect(onSend).not.toHaveBeenCalled();
        expect(composer).toHaveValue("first\nsecond");
    });

    it("refuses to send an empty draft even when Enter is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { onSend } = renderChat();

        // when
        await user.click(screen.getByPlaceholderText("Type a message..."));
        await user.keyboard("{Enter}");

        // then
        expect(onSend).not.toHaveBeenCalled();
    });

    it("shows that a message is on its way while the request is in flight", async () => {
        // given
        let release: () => void = () => {};
        const user = userEvent.setup();
        renderChat({
            onSend: () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        });

        // when
        await user.type(screen.getByPlaceholderText("Type a message..."), "kihihi");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(screen.getByRole("button", { name: "..." })).toBeDisabled();
        release();
        await waitFor(() => {
            expect(screen.getByPlaceholderText("Type a message...")).toHaveValue("");
        });
    });

    it("swallows a refused send instead of leaving the rejection unhandled", async () => {
        // given
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        nodeProcess.on("unhandledRejection", record);
        const user = userEvent.setup();
        renderChat({ onSend: () => Promise.reject(new Error("you are muted")) });

        // when
        await user.type(screen.getByPlaceholderText("Type a message..."), "kihihi");
        await user.click(screen.getByRole("button", { name: "Send" }));
        await new Promise(resolve => setTimeout(resolve, 0));
        nodeProcess.off("unhandledRejection", record);

        // then
        expect(unhandled).toEqual([]);
        expect(screen.getByPlaceholderText("Type a message...")).toHaveValue("kihihi");
    });

    it("caps how much can be typed into a single message", () => {
        // given
        const messages: WatchPartyMessage[] = [];

        // when
        renderChat({ messages });

        // then
        expect(screen.getByPlaceholderText("Type a message...")).toHaveAttribute("maxlength", "2000");
    });
});
