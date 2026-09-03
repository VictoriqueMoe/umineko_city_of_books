import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ChatMessage } from "../../../../types/api";
import { makeChatMessage } from "../../../../test-utils/fixtures";
import { renderWithProviders } from "../../../../test-utils/render";
import { LinksTab } from "./LinksTab";

const { useChatRoomAttachments } = vi.hoisted(() => ({ useChatRoomAttachments: vi.fn() }));

vi.mock("../../../../hooks/queries/chat", () => ({ useChatRoomAttachments }));

const roomId = "room-1";

function stub(messages: ChatMessage[], loading = false) {
    useChatRoomAttachments.mockReturnValue({ messages, loading, refresh: vi.fn() });
}

function withBody(id: string, body: string): ChatMessage {
    return makeChatMessage({ id, room_id: roomId, body, created_at: "2026-08-01T09:00:00Z" });
}

function renderTab(overrides: Partial<Parameters<typeof LinksTab>[0]> = {}) {
    const onJump = vi.fn();
    const onJumped = vi.fn();
    renderWithProviders(<LinksTab roomId={roomId} isActive onJump={onJump} onJumped={onJumped} {...overrides} />);

    return { onJump, onJumped };
}

describe("LinksTab", () => {
    it("issues no request while the tab is not the active one", () => {
        // given
        stub([]);

        // when
        renderTab({ isActive: false });

        // then
        expect(useChatRoomAttachments).toHaveBeenLastCalledWith(roomId, "links", false);
    });

    it("shows an empty state when nothing has been shared", () => {
        // given
        stub([]);

        // when
        renderTab();

        // then
        expect(screen.getByText("No links have been shared here yet.")).toBeInTheDocument();
    });

    it("renders one row per link in a message", () => {
        // given
        stub([withBody("m1", "see https://a.example/one and https://b.example/two")]);

        // when
        renderTab();

        // then
        expect(screen.getByText("https://a.example/one")).toBeInTheDocument();
        expect(screen.getByText("https://b.example/two")).toBeInTheDocument();
    });

    it("shows the site a link came from", () => {
        // given
        stub([withBody("m1", "https://www.example.com/page")]);

        // when
        renderTab();

        // then
        expect(screen.getByText("example.com")).toBeInTheDocument();
    });

    it("renders a link repeated within one message only once", () => {
        // given
        stub([withBody("m1", "https://a.example/x and again https://a.example/x")]);

        // when
        renderTab();

        // then
        expect(screen.getAllByText("https://a.example/x")).toHaveLength(1);
    });

    it("includes inline media links, which the timeline embeds rather than links", () => {
        // given
        stub([withBody("m1", "https://waifuvault.moe/f/1/cat.png")]);

        // when
        renderTab();

        // then
        expect(screen.getByText("https://waifuvault.moe/f/1/cat.png")).toBeInTheDocument();
    });

    it("leaves a sentence's full stop out of the link", () => {
        // given
        stub([withBody("m1", "have a look at https://example.com/page.")]);

        // when
        renderTab();

        // then
        expect(screen.getByText("https://example.com/page")).toBeInTheDocument();
    });

    it("jumps back to the message the link came from", async () => {
        // given
        stub([withBody("m1", "https://a.example/x")]);
        const user = userEvent.setup();
        const { onJump, onJumped } = renderTab();

        // when
        await user.click(screen.getByRole("button", { name: "Jump to message" }));

        // then
        expect(onJump).toHaveBeenCalledWith("m1", "2026-08-01T09:00:00Z");
        expect(onJumped).toHaveBeenCalled();
    });
});
