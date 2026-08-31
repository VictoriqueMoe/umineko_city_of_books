import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeSpectatorMessage, makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { emitRealtimeEvent, makeWSHarness, type WSHarness } from "../../../test-utils/ws";
import { GameChat } from "./GameChat";

const endpoints = vi.hoisted(() => ({
    getSpectatorChat: vi.fn(),
    postSpectatorChat: vi.fn(),
    getPlayerChat: vi.fn(),
    postPlayerChat: vi.fn(),
}));

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));

vi.mock("../../../api/endpoints/gameRoom", () => endpoints);

vi.mock("../../../api/realtime/pipeline", () => ({
    ensureRealtimePipeline: () => {},
    getRealtimeEpoch: () => holder.ws.getEpoch(),
    subscribeRealtimeEpoch: (listener: () => void) => holder.ws.subscribeEpoch(listener),
}));

const SPECTATOR_PLACEHOLDER = "Chat with other watchers...";
const PLAYER_PLACEHOLDER = "Message your opponent...";

const viewer = makeUser({ id: "u-one", username: "battler", display_name: "Battler" });

describe("GameChat", () => {
    beforeEach(() => {
        holder.ws = makeWSHarness();
        endpoints.getSpectatorChat.mockResolvedValue({ messages: [] });
        endpoints.getPlayerChat.mockResolvedValue({ messages: [] });
        endpoints.postSpectatorChat.mockResolvedValue(makeSpectatorMessage({ id: "m-sent", body: "hello" }));
        endpoints.postPlayerChat.mockResolvedValue(makeSpectatorMessage({ id: "m-sent", body: "hello" }));
    });

    it("heads the spectator chat with the number of watchers", async () => {
        // given
        const watcherCount = 7;

        // when
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" watcherCount={watcherCount} />, {
            user: viewer,
        });

        // then
        expect(screen.getByText("Spectator chat")).toBeInTheDocument();
        expect(screen.getByText("7 watching")).toBeInTheDocument();
        await waitFor(() => {
            expect(endpoints.getSpectatorChat).toHaveBeenCalledWith("room-1");
        });
    });

    it("tells watchers the room is quiet before anyone speaks", async () => {
        // given
        endpoints.getSpectatorChat.mockResolvedValue({ messages: [] });

        // when
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });

        // then
        expect(await screen.findByText("No messages yet. Say hello.")).toBeInTheDocument();
    });

    it("marks the player chat as private and keeps it to the two of them", async () => {
        // given
        endpoints.getPlayerChat.mockResolvedValue({ messages: [] });

        // when
        renderWithProviders(<GameChat roomId="room-1" variant="player" />, { user: viewer });

        // then
        expect(screen.getByText("Player chat")).toBeInTheDocument();
        expect(screen.getByText("Private")).toBeInTheDocument();
        expect(screen.getByText("Only you and your opponent can see this chat.")).toBeInTheDocument();
        await waitFor(() => {
            expect(endpoints.getPlayerChat).toHaveBeenCalledWith("room-1");
        });
        expect(endpoints.getSpectatorChat).not.toHaveBeenCalled();
    });

    it("lists the messages the room already has", async () => {
        // given
        endpoints.getSpectatorChat.mockResolvedValue({
            messages: [
                makeSpectatorMessage(),
                makeSpectatorMessage({ id: "m2", body: "without love it cannot be seen" }),
            ],
        });

        // when
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });

        // then
        expect(await screen.findByText("the golden truth")).toBeInTheDocument();
        expect(screen.getByText("without love it cannot be seen")).toBeInTheDocument();
    });

    it("stays quiet when the history cannot be loaded", async () => {
        // given
        endpoints.getSpectatorChat.mockRejectedValue(new Error("gone"));

        // when
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });

        // then
        expect(await screen.findByText("No messages yet. Say hello.")).toBeInTheDocument();
    });

    it("fetches the history again once the socket has reconnected", async () => {
        // given
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });
        await waitFor(() => {
            expect(endpoints.getSpectatorChat).toHaveBeenCalledTimes(1);
        });

        // when
        holder.ws.reconnect();

        // then
        await waitFor(() => {
            expect(endpoints.getSpectatorChat).toHaveBeenCalledTimes(2);
        });
    });

    it("asks a signed out visitor to sign in first", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user });

        // then
        expect(screen.getByText("Sign in to join the chat.")).toBeInTheDocument();
        expect(screen.queryByPlaceholderText(SPECTATOR_PLACEHOLDER)).not.toBeInTheDocument();
    });

    it("keeps the send control shut until something has been typed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });

        // when
        await user.type(screen.getByPlaceholderText(SPECTATOR_PLACEHOLDER), "   ");

        // then
        expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
        expect(endpoints.postSpectatorChat).not.toHaveBeenCalled();
    });

    it("posts the trimmed message and clears the box", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });

        // when
        await user.type(screen.getByPlaceholderText(SPECTATOR_PLACEHOLDER), "  hello  ");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(endpoints.postSpectatorChat).toHaveBeenCalledWith("room-1", "hello");
        expect(await screen.findByText("hello")).toBeInTheDocument();
        expect(screen.getByPlaceholderText(SPECTATOR_PLACEHOLDER)).toHaveValue("");
    });

    it("posts a player message down the private channel", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GameChat roomId="room-1" variant="player" />, { user: viewer });

        // when
        await user.type(screen.getByPlaceholderText(PLAYER_PLACEHOLDER), "hello");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(endpoints.postPlayerChat).toHaveBeenCalledWith("room-1", "hello");
        expect(endpoints.postSpectatorChat).not.toHaveBeenCalled();
    });

    it("sends the message when Enter is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });

        // when
        await user.type(screen.getByPlaceholderText(SPECTATOR_PLACEHOLDER), "hello{Enter}");

        // then
        await waitFor(() => {
            expect(endpoints.postSpectatorChat).toHaveBeenCalledWith("room-1", "hello");
        });
    });

    it("leaves the message alone for Shift and Enter", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });

        // when
        await user.type(screen.getByPlaceholderText(SPECTATOR_PLACEHOLDER), "hello{Shift>}{Enter}{/Shift}");

        // then
        expect(endpoints.postSpectatorChat).not.toHaveBeenCalled();
    });

    it("shows why the message would not send", async () => {
        // given
        const user = userEvent.setup();
        endpoints.postSpectatorChat.mockRejectedValue(new Error("you are timed out"));
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });

        // when
        await user.type(screen.getByPlaceholderText(SPECTATOR_PLACEHOLDER), "hello");
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(await screen.findByText("you are timed out")).toBeInTheDocument();
        expect(screen.getByPlaceholderText(SPECTATOR_PLACEHOLDER)).toHaveValue("hello");
    });

    it("adds a message that arrives over the socket", async () => {
        // given
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });
        await screen.findByText("No messages yet. Say hello.");

        // when
        emitRealtimeEvent({
            type: "spectator_chat_message",
            data: { room_id: "room-1", message: makeSpectatorMessage({ id: "m-ws", body: "beato is watching" }) },
        });

        // then
        expect(screen.getByText("beato is watching")).toBeInTheDocument();
    });

    it("ignores a socket message meant for another room", async () => {
        // given
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });
        await screen.findByText("No messages yet. Say hello.");

        // when
        emitRealtimeEvent({
            type: "spectator_chat_message",
            data: { room_id: "room-2", message: makeSpectatorMessage({ id: "m-ws", body: "somewhere else" }) },
        });

        // then
        expect(screen.queryByText("somewhere else")).not.toBeInTheDocument();
    });

    it("ignores a socket message of another kind", async () => {
        // given
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });
        await screen.findByText("No messages yet. Say hello.");

        // when
        emitRealtimeEvent({
            type: "player_chat_message",
            data: { room_id: "room-1", message: makeSpectatorMessage({ id: "m-ws", body: "wrong channel" }) },
        });

        // then
        expect(screen.queryByText("wrong channel")).not.toBeInTheDocument();
    });

    it("never shows the same message twice", async () => {
        // given
        endpoints.getSpectatorChat.mockResolvedValue({
            messages: [makeSpectatorMessage({ id: "m-dup", body: "only once" })],
        });
        renderWithProviders(<GameChat roomId="room-1" variant="spectator" />, { user: viewer });
        await screen.findByText("only once");

        // when
        emitRealtimeEvent({
            type: "spectator_chat_message",
            data: { room_id: "room-1", message: makeSpectatorMessage({ id: "m-dup", body: "only once" }) },
        });

        // then
        expect(screen.getAllByText("only once")).toHaveLength(1);
    });
});
