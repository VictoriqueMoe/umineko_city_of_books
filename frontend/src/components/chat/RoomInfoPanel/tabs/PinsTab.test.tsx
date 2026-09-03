import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ChatMessage, User } from "../../../../types/api";
import { makeChatMessage } from "../../../../test-utils/fixtures";
import { renderWithProviders } from "../../../../test-utils/render";
import { PinsTab } from "./PinsTab";

const { useChatRoomPinnedMessages, useUnpinChatMessage } = vi.hoisted(() => ({
    useChatRoomPinnedMessages: vi.fn(),
    useUnpinChatMessage: vi.fn(),
}));

vi.mock("../../../../hooks/queries/chat", () => ({ useChatRoomPinnedMessages }));
vi.mock("../../../../hooks/mutations/chat", () => ({ useUnpinChatMessage }));

const roomId = "room-1";

function makeSender(overrides: Partial<User> = {}): User {
    return {
        id: "user-1",
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "",
        ...overrides,
    };
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({
        id: "msg-1",
        room_id: roomId,
        sender: makeSender(),
        body: "Without love it cannot be seen",
        created_at: "2026-08-01T09:00:00Z",
        pinned: true,
        pinned_at: "2026-08-01T10:00:00Z",
        ...overrides,
    });
}

function stubPinned(messages: ChatMessage[], loading = false) {
    const refresh = vi.fn(() => Promise.resolve());
    useChatRoomPinnedMessages.mockReturnValue({ messages, loading, refresh });

    return refresh;
}

interface UnpinCallbacks {
    onSettled?: () => void;
}

function stubUnpin(impl: (messageId: string) => Promise<unknown> = () => Promise.resolve()) {
    const mutate = vi.fn((messageId: string, callbacks?: UnpinCallbacks) => {
        impl(messageId).then(
            () => callbacks?.onSettled?.(),
            () => callbacks?.onSettled?.(),
        );
    });
    useUnpinChatMessage.mockReturnValue({ mutate });

    return mutate;
}

function renderPanel(overrides: Partial<ComponentProps<typeof PinsTab>> = {}) {
    const onJumped = vi.fn();
    const onJump = vi.fn();
    const onLightbox = vi.fn();
    const result = renderWithProviders(
        <PinsTab
            roomId={roomId}
            isActive
            onJumped={onJumped}
            onJump={onJump}
            onLightbox={onLightbox}
            canUnpin={false}
            {...overrides}
        />,
    );

    return { ...result, onClose: onJumped, onJump, onLightbox };
}

describe("PinsTab", () => {
    it("leaves the pinned query disabled while the tab is not the active one", () => {
        // given
        stubPinned([]);
        stubUnpin();

        // when
        renderPanel({ isActive: false });

        // then
        expect(useChatRoomPinnedMessages).toHaveBeenLastCalledWith(roomId, false);
    });

    it("shows a loading notice before the pins arrive", () => {
        // given
        stubPinned([], true);
        stubUnpin();

        // when
        renderPanel();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("shows an empty state when nothing has been pinned", () => {
        // given
        stubPinned([]);
        stubUnpin();

        // when
        renderPanel();

        // then
        expect(screen.getByText("No pinned messages yet.")).toBeInTheDocument();
    });

    it("orders the pins with the most recently pinned first", () => {
        // given
        stubPinned([
            makeMessage({ id: "old", body: "pinned first", pinned_at: "2026-08-01T10:00:00Z" }),
            makeMessage({ id: "new", body: "pinned last", pinned_at: "2026-08-01T12:00:00Z" }),
        ]);
        stubUnpin();

        // when
        renderPanel();

        // then
        const bodies = screen.getAllByText(/^pinned /).map(node => node.textContent);
        expect(bodies).toEqual(["pinned last", "pinned first"]);
    });

    it("sinks pins with no pinned timestamp to the bottom of the list", () => {
        // given
        stubPinned([
            makeMessage({ id: "unstamped", body: "pinned without a stamp", pinned_at: undefined }),
            makeMessage({ id: "stamped", body: "pinned with a stamp", pinned_at: "2026-08-01T12:00:00Z" }),
        ]);
        stubUnpin();

        // when
        renderPanel();

        // then
        const bodies = screen.getAllByText(/^pinned /).map(node => node.textContent);
        expect(bodies).toEqual(["pinned with a stamp", "pinned without a stamp"]);
    });

    it("prefers the room nickname and room avatar over the profile ones", () => {
        // given
        stubPinned([
            makeMessage({
                sender: makeSender({ display_name: "Beatrice", avatar_url: "/profile.png" }),
                sender_nickname: "Golden Witch",
                sender_member_avatar_url: "/member.png",
            }),
        ]);
        stubUnpin();

        // when
        const { container } = renderPanel();

        // then
        expect(screen.getByText("Golden Witch")).toBeInTheDocument();
        expect(container.querySelector("img")).toHaveAttribute("src", "/member.png");
    });

    it("falls back to the profile name and an initial when the room profile is blank", () => {
        // given
        stubPinned([
            makeMessage({
                sender: makeSender({ display_name: "Beatrice", avatar_url: "" }),
                sender_nickname: "   ",
                sender_member_avatar_url: "   ",
            }),
        ]);
        stubUnpin();

        // when
        const { container } = renderPanel();

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("B")).toBeInTheDocument();
        expect(container.querySelector("img")).toBeNull();
    });

    it("hides the unpin control from a viewer who is not allowed to unpin", () => {
        // given
        stubPinned([makeMessage()]);
        stubUnpin();

        // when
        renderPanel({ canUnpin: false });

        // then
        expect(screen.queryByRole("button", { name: "Unpin" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Jump to message" })).toBeInTheDocument();
    });

    it("offers the unpin control to a viewer who is allowed to unpin", () => {
        // given
        stubPinned([makeMessage()]);
        stubUnpin();

        // when
        renderPanel({ canUnpin: true });

        // then
        expect(screen.getByRole("button", { name: "Unpin" })).toBeInTheDocument();
    });

    it("jumps to the pinned message and closes the panel", async () => {
        // given
        stubPinned([makeMessage()]);
        stubUnpin();
        const user = userEvent.setup();
        const { onClose, onJump } = renderPanel();

        // when
        await user.click(screen.getByRole("button", { name: "Jump to message" }));

        // then
        expect(onJump).toHaveBeenCalledWith("msg-1", "2026-08-01T09:00:00Z");
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("hands the unpin to the mutation and leaves the cache to it", async () => {
        // given
        stubPinned([makeMessage({ id: "msg-1", body: "first pin" })]);
        const mutate = stubUnpin();
        const user = userEvent.setup();
        renderPanel({ canUnpin: true });

        // when
        await user.click(screen.getByRole("button", { name: "Unpin" }));

        // then
        expect(mutate.mock.calls[0][0]).toBe("msg-1");
    });

    it("shows a busy label on only the row being unpinned", async () => {
        // given
        let release: () => void = () => {};
        stubPinned([
            makeMessage({ id: "msg-1", pinned_at: "2026-08-01T12:00:00Z" }),
            makeMessage({ id: "msg-2", pinned_at: "2026-08-01T10:00:00Z" }),
        ]);
        stubUnpin(
            () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        );
        const user = userEvent.setup();
        renderPanel({ canUnpin: true });

        // when
        await user.click(screen.getAllByRole("button", { name: "Unpin" })[0]);

        // then
        expect(screen.getByRole("button", { name: "Unpinning..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Unpin" })).toBeEnabled();
        await act(async () => {
            release();
        });
        expect(screen.getAllByRole("button", { name: "Unpin" })).toHaveLength(2);
    });

    it("releases the busy row when the unpin request fails", async () => {
        // given
        stubPinned([makeMessage({ id: "msg-1" })]);
        stubUnpin(() => Promise.reject(new Error("the witch forbids it")));
        const user = userEvent.setup();
        renderPanel({ canUnpin: true });

        // when
        await user.click(screen.getByRole("button", { name: "Unpin" }));

        // then
        await waitFor(() => {
            expect(screen.getByRole("button", { name: "Unpin" })).toBeEnabled();
        });
        expect(screen.getByText("Without love it cannot be seen")).toBeInTheDocument();
    });

    it("opens the lightbox when a pinned image is clicked", async () => {
        // given
        stubPinned([
            makeMessage({
                body: "",
                media: [{ id: 1, media_url: "/pinned.png", media_type: "image", sort_order: 0 }],
            }),
        ]);
        stubUnpin();
        const user = userEvent.setup();
        const { container, onLightbox } = renderPanel();

        // when
        await user.click(container.querySelector("img") as HTMLElement);

        // then
        expect(onLightbox).toHaveBeenCalledWith("/pinned.png");
    });

    it("renders pinned video media as a player rather than an image", () => {
        // given
        stubPinned([
            makeMessage({
                body: "",
                media: [
                    {
                        id: 2,
                        media_url: "/pinned.mp4",
                        media_type: "video",
                        thumbnail_url: "/poster.png",
                        sort_order: 0,
                    },
                ],
            }),
        ]);
        stubUnpin();

        // when
        const { container } = renderPanel();

        // then
        const video = container.querySelector("video");
        expect(video).toHaveAttribute("src", "/pinned.mp4");
        expect(video).toHaveAttribute("poster", "/poster.png");
        expect(container.querySelector("img")).toBeNull();
    });

    it("never refetches the pins by hand", () => {
        // given
        const refresh = stubPinned([makeMessage()]);
        stubUnpin();

        // when
        const { rerender } = renderPanel();
        rerender(<PinsTab roomId={roomId} isActive onJumped={() => {}} onJump={() => {}} canUnpin={false} />);

        // then
        expect(refresh).not.toHaveBeenCalled();
    });
});
