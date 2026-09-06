import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ChatMessage } from "../../../../types/api";
import { makeChatMessage } from "../../../../test-utils/fixtures";
import { renderWithProviders } from "../../../../test-utils/render";
import { MediaTab } from "./MediaTab";

const { useChatRoomAttachments } = vi.hoisted(() => ({ useChatRoomAttachments: vi.fn() }));

vi.mock("../../../../hooks/queries/chat", () => ({ useChatRoomAttachments }));

const roomId = "room-1";

function stub(messages: ChatMessage[], loading = false, more: Partial<AttachmentStub> = {}) {
    useChatRoomAttachments.mockReturnValue({
        messages,
        loading,
        loadingMore: false,
        hasMore: false,
        loadMore: vi.fn(),
        refresh: vi.fn(),
        ...more,
    });
}

interface AttachmentStub {
    loadingMore: boolean;
    hasMore: boolean;
    loadMore: () => void;
}

function withMedia(id: string, media: ChatMessage["media"]): ChatMessage {
    return makeChatMessage({ id, room_id: roomId, body: "", created_at: "2026-08-01T09:00:00Z", media });
}

function renderTab(overrides: Partial<Parameters<typeof MediaTab>[0]> = {}) {
    const onJump = vi.fn();
    const onJumped = vi.fn();
    const onLightbox = vi.fn();
    renderWithProviders(
        <MediaTab
            roomId={roomId}
            isActive
            onJump={onJump}
            onJumped={onJumped}
            onLightbox={onLightbox}
            {...overrides}
        />,
    );

    return { onJump, onJumped, onLightbox };
}

describe("MediaTab", () => {
    it("issues no request while the tab is not the active one", () => {
        // given
        stub([]);

        // when
        renderTab({ isActive: false });

        // then
        expect(useChatRoomAttachments).toHaveBeenLastCalledWith(roomId, "media", false);
    });

    it("shows an empty state when the room holds no media", () => {
        // given
        stub([]);

        // when
        renderTab();

        // then
        expect(screen.getByText("Nothing has been posted here yet.")).toBeInTheDocument();
    });

    it("flattens every media item across messages into its own cell", () => {
        // given
        stub([
            withMedia("m1", [
                { id: 1, media_url: "/a.png", media_type: "image", sort_order: 0 },
                { id: 2, media_url: "/b.png", media_type: "image", sort_order: 1 },
            ]),
            withMedia("m2", [{ id: 3, media_url: "/c.mp4", media_type: "video", sort_order: 0 }]),
        ]);

        // when
        renderTab();

        // then
        expect(screen.getAllByRole("button", { name: "Jump to message" })).toHaveLength(3);
    });

    it("prefers the thumbnail for an image cell", () => {
        // given
        stub([
            withMedia("m1", [
                { id: 1, media_url: "/full.png", media_type: "image", thumbnail_url: "/thumb.png", sort_order: 0 },
            ]),
        ]);

        // when
        const { container } = { container: document.body };
        renderTab();

        // then
        expect(container.querySelector("img")).toHaveAttribute("src", "/thumb.png");
    });

    it("keeps a spoiler covered in the grid until it is clicked", async () => {
        // given a spoiler image in the room's media grid
        stub([
            withMedia("m1", [
                {
                    id: 1,
                    media_url: "/full.png",
                    media_type: "image",
                    thumbnail_url: "/thumb.png",
                    sort_order: 0,
                    is_spoiler: true,
                },
            ]),
        ]);
        const user = userEvent.setup();
        const { onLightbox } = renderTab();

        // when the viewer clicks the covered tile
        await user.click(screen.getByRole("button", { name: "Reveal spoiler" }));

        // then it only revealed, rather than jumping straight to full size
        expect(onLightbox).not.toHaveBeenCalled();

        // when they click the revealed tile
        await user.click(screen.getByRole("button", { name: "Open full size" }));

        // then it opens
        expect(onLightbox).toHaveBeenCalledWith("/full.png");
    });

    it("opens an image in the lightbox at full size rather than jumping", async () => {
        // given
        stub([
            withMedia("m1", [
                { id: 1, media_url: "/full.png", media_type: "image", thumbnail_url: "/thumb.png", sort_order: 0 },
            ]),
        ]);
        const user = userEvent.setup();
        const { onLightbox, onJump } = renderTab();

        // when
        await user.click(screen.getByRole("button", { name: "Open full size" }));

        // then
        expect(onLightbox).toHaveBeenCalledWith("/full.png");
        expect(onJump).not.toHaveBeenCalled();
    });

    it("jumps to the message from the button under any thumbnail", async () => {
        // given
        stub([withMedia("m2", [{ id: 3, media_url: "/c.mp4", media_type: "video", sort_order: 0 }])]);
        const user = userEvent.setup();
        const { onJump, onJumped } = renderTab();

        // when
        await user.click(screen.getByRole("button", { name: "Jump to message" }));

        // then
        expect(onJump).toHaveBeenCalledWith("m2", "2026-08-01T09:00:00Z");
        expect(onJumped).toHaveBeenCalled();
    });

    it("offers a jump for an image as well as the full size view", async () => {
        // given
        stub([withMedia("m1", [{ id: 1, media_url: "/full.png", media_type: "image", sort_order: 0 }])]);
        const user = userEvent.setup();
        const { onJump, onJumped, onLightbox } = renderTab();

        // when
        await user.click(screen.getByRole("button", { name: "Jump to message" }));

        // then
        expect(onJump).toHaveBeenCalledWith("m1", "2026-08-01T09:00:00Z");
        expect(onJumped).toHaveBeenCalled();
        expect(onLightbox).not.toHaveBeenCalled();
    });

    it("leaves the thumbnail inert for media with no full size view", () => {
        // given
        stub([withMedia("m2", [{ id: 3, media_url: "/c.mp4", media_type: "video", sort_order: 0 }])]);

        // when
        renderTab();

        // then
        expect(screen.getByRole("button", { name: "Open full size" })).toBeDisabled();
    });

    it("names the file on an audio cell", () => {
        // given
        stub([
            withMedia("m3", [
                { id: 4, media_url: "/x.mp3", media_type: "audio", filename: "beato.mp3", sort_order: 0 },
            ]),
        ]);

        // when
        renderTab();

        // then
        expect(screen.getByText("beato.mp3")).toBeInTheDocument();
    });
});
