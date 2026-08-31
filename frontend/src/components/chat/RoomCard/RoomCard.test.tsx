import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { makeChatRoom } from "../../../test-utils/fixtures";
import type { ChatRoom } from "../../../types/api";
import { RoomCard, type RoomCardVariant } from "./RoomCard";

const ROOM_PAGE_MARKER = "the room itself";

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({ member_count: 4, ...overrides });
}

function renderRouted(ui: ReactNode) {
    return render(
        <MemoryRouter initialEntries={["/rooms"]}>
            <Routes>
                <Route path="/rooms" element={ui} />
                <Route path="/rooms/:roomId" element={<div>{ROOM_PAGE_MARKER}</div>} />
            </Routes>
        </MemoryRouter>,
    );
}

function renderCard(variant: RoomCardVariant, room: ChatRoom, actions?: ReactNode) {
    const onTagClick = vi.fn();
    const result = renderRouted(<RoomCard variant={variant} room={room} onTagClick={onTagClick} actions={actions} />);

    return { ...result, onTagClick };
}

describe("RoomCard member variant", () => {
    it("links the card to the room's own page and counts its members", () => {
        // given
        const room = makeRoom({ id: "room-9", name: "Rokkenjima", member_count: 6 });

        // when
        renderCard("member", room);

        // then
        expect(screen.getByRole("link", { name: /Rokkenjima/ })).toHaveAttribute("href", "/rooms/room-9");
        expect(screen.getByText(/6 members/)).toBeInTheDocument();
    });

    it("falls back to the loaded member list when the server sent no count", () => {
        // given
        const room = makeRoom({
            member_count: undefined as unknown as number,
            members: [
                { id: "u1", username: "battler", display_name: "Battler" },
                { id: "u2", username: "beatrice", display_name: "Beatrice" },
            ],
        });

        // when
        renderCard("member", room);

        // then
        expect(screen.getByText(/2 members/)).toBeInTheDocument();
    });

    it("opens the room when the card itself is clicked", async () => {
        // given
        const user = userEvent.setup();
        renderCard("member", makeRoom({ name: "Rokkenjima" }));

        // when
        await user.click(screen.getByRole("link", { name: /Rokkenjima/ }));

        // then
        expect(screen.getByText(ROOM_PAGE_MARKER)).toBeInTheDocument();
    });

    it("badges the room the viewer hosts", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        renderCard("member", room);

        // then
        expect(screen.getByText("Host")).toBeInTheDocument();
    });

    it("badges a roleplay room, a private room and an archived one", () => {
        // given
        const room = makeRoom({ is_rp: true, is_public: false, archived_at: "2026-01-02T00:00:00Z" });

        // when
        renderCard("member", room);

        // then
        expect(screen.getByText("RP")).toBeInTheDocument();
        expect(screen.getByText("Private")).toBeInTheDocument();
        expect(screen.getByText("Archived")).toBeInTheDocument();
    });

    it("marks a room the viewer joined silently or muted", () => {
        // given
        const room = makeRoom({ viewer_ghost: true, viewer_muted: true });

        // when
        renderCard("member", room);

        // then
        expect(screen.getByTitle("You joined silently as a ghost")).toBeInTheDocument();
        expect(screen.getByTitle("Notifications muted")).toBeInTheDocument();
    });

    it("badges a system room and leaves its openness unstated", () => {
        // given
        const room = makeRoom({ is_system: true });

        // when
        renderCard("member", room);

        // then
        expect(screen.getByText("System")).toBeInTheDocument();
        expect(screen.queryByText("Public")).not.toBeInTheDocument();
        expect(screen.queryByText("Private")).not.toBeInTheDocument();
    });

    it("filters by a tag without opening the room", async () => {
        // given
        const user = userEvent.setup();
        const { onTagClick } = renderCard("member", makeRoom({ tags: ["horror"] }));

        // when
        await user.click(screen.getByRole("button", { name: "#horror" }));

        // then
        expect(onTagClick).toHaveBeenCalledWith("horror");
        expect(screen.queryByText(ROOM_PAGE_MARKER)).not.toBeInTheDocument();
    });
});

describe("RoomCard discover variant", () => {
    it("never links a room the viewer has not joined", () => {
        // given
        const room = makeRoom({ name: "Rokkenjima" });

        // when
        renderCard("discover", room);

        // then
        expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
    });

    it("always calls a discovered room public", () => {
        // given
        const room = makeRoom({ is_public: false });

        // when
        renderCard("discover", room);

        // then
        expect(screen.getByText("Public")).toBeInTheDocument();
        expect(screen.queryByText("Private")).not.toBeInTheDocument();
    });

    it("filters by a tag on a discovered room", async () => {
        // given
        const user = userEvent.setup();
        const { onTagClick } = renderCard("discover", makeRoom({ tags: ["horror"] }));

        // when
        await user.click(screen.getByRole("button", { name: "#horror" }));

        // then
        expect(onTagClick).toHaveBeenCalledWith("horror");
    });

    it("carries the actions the caller put under the card", () => {
        // given
        const actions = <button type="button">Join Room</button>;

        // when
        renderCard("discover", makeRoom(), actions);

        // then
        expect(screen.getByRole("button", { name: "Join Room" })).toBeInTheDocument();
    });

    it("leaves the action row out when the caller gave none", () => {
        // given
        const room = makeRoom();

        // when
        renderCard("discover", room);

        // then
        expect(screen.queryByRole("button", { name: "Join Room" })).not.toBeInTheDocument();
    });
});

describe("RoomCard shared badges", () => {
    const variants: RoomCardVariant[] = ["member", "discover"];

    for (const variant of variants) {
        it(`marks a busy ${variant} room as hot`, () => {
            // given
            const room = makeRoom({ hot_score: 51 });

            // when
            renderCard(variant, room);

            // then
            expect(screen.getByText("Hot")).toBeInTheDocument();
        });

        it(`never calls an archived ${variant} room hot`, () => {
            // given
            const room = makeRoom({ hot_score: 99, archived_at: "2026-01-02T00:00:00Z" });

            // when
            renderCard(variant, room);

            // then
            expect(screen.getByText("Archived")).toBeInTheDocument();
            expect(screen.queryByText("Hot")).not.toBeInTheDocument();
        });

        it(`shows how many people are in voice on a ${variant} room`, () => {
            // given
            const room = makeRoom({ voice_count: 3 });

            // when
            renderCard(variant, room);

            // then
            expect(screen.getByTitle("Voice chat active")).toHaveTextContent("3");
        });

        it(`badges a roleplay ${variant} room`, () => {
            // given
            const room = makeRoom({ is_rp: true });

            // when
            renderCard(variant, room);

            // then
            expect(screen.getByText("RP")).toBeInTheDocument();
        });

        it(`shows the description of a ${variant} room that has one`, () => {
            // given
            const room = makeRoom({ description: "Tea and murder" });

            // when
            renderCard(variant, room);

            // then
            expect(screen.getByText("Tea and murder")).toBeInTheDocument();
        });

        it(`leaves the tag row out of a ${variant} room with no tags`, () => {
            // given
            const room = makeRoom({ tags: [] });

            // when
            renderCard(variant, room);

            // then
            expect(screen.queryByRole("button", { name: /^#/ })).not.toBeInTheDocument();
        });
    }
});
