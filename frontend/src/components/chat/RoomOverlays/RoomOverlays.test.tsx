import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ChatRoom, ChatRoomMember, UserProfile, WatchPartySession } from "../../../types/api";
import type { ActiveWatchPartySession } from "../../../hooks/useWatchParty";
import {
    makeChatRoom,
    makePublicUser,
    makeRoomMember,
    makeUser,
    makeWatchPartySession,
} from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { RoomOverlays, type RoomOverlaysProps } from "./RoomOverlays";

vi.mock("../RoomInfoPanel/RoomInfoPanel", () => ({
    RoomInfoPanel: (props: {
        roomId: string;
        tab: string | null;
        canUnpin: boolean;
        onLightbox?: (src: string) => void;
    }) =>
        props.tab === null ? null : (
            <button
                type="button"
                data-testid="info-panel"
                data-tab={props.tab}
                data-can-unpin={String(props.canUnpin)}
                onClick={() => props.onLightbox?.("/img.png")}
            >
                {props.roomId}
            </button>
        ),
}));

vi.mock("../EditRoomProfileDialog/EditRoomProfileDialog", () => ({
    EditRoomProfileDialog: (props: { isOpen: boolean; currentMember: ChatRoomMember | null }) => (
        <div data-testid="edit-profile" data-open={String(props.isOpen)}>
            {props.currentMember?.user.id ?? "none"}
        </div>
    ),
}));

vi.mock("../RoomModerationDialog/RoomModerationDialog", () => ({
    RoomModerationDialog: (props: { isOpen: boolean; room: ChatRoom }) => (
        <div data-testid="room-moderation" data-open={String(props.isOpen)}>
            {props.room.name}
        </div>
    ),
}));

vi.mock("../WatchParty/WatchPartyModal", () => ({
    WatchPartyModal: (props: { isStarter: boolean; voiceEnabled: boolean; viewerIsStaff: boolean }) => (
        <div
            data-testid="watch-party-modal"
            data-starter={String(props.isStarter)}
            data-voice={String(props.voiceEnabled)}
            data-staff={String(props.viewerIsStaff)}
        >
            watch party
        </div>
    ),
}));

vi.mock("../InviteMembersModal/InviteMembersModal", () => ({
    InviteMembersModal: (props: {
        isOpen: boolean;
        existingMemberIds: Set<string>;
        onInvited?: (result: { invited_count: number; skipped_count: number }) => void;
    }) => (
        <button
            type="button"
            data-testid="invite-modal"
            data-open={String(props.isOpen)}
            data-existing={[...props.existingMemberIds].join(",")}
            onClick={() => props.onInvited?.({ invited_count: 2, skipped_count: 0 })}
        >
            invite
        </button>
    ),
}));

const viewer: UserProfile = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });

function makeMember(id: string): ChatRoomMember {
    return makeRoomMember({ user: makePublicUser({ id }) });
}

function makeSession(overrides: Partial<WatchPartySession> = {}): WatchPartySession {
    return makeWatchPartySession({
        id: "wp-1",
        started_by: viewer.id,
        controller_id: viewer.id,
        title: "Ep 1",
        started_at: "2026-01-01T00:00:00Z",
        ...overrides,
    });
}

function makeActiveSession(overrides: Partial<WatchPartySession> = {}): ActiveWatchPartySession {
    return { session: makeSession(overrides), embedURL: "https://example.test/embed", hasControl: true };
}

function makeProps(overrides: Partial<RoomOverlaysProps> = {}): RoomOverlaysProps {
    return {
        room: makeChatRoom(),
        viewer,
        viewerIsStaff: false,
        canModerateRoom: false,
        voiceEnabled: true,
        onJump: vi.fn(),
        onLightbox: vi.fn(),
        infoPanel: { tab: null, onTabChange: vi.fn(), onClose: vi.fn() },
        editProfile: { open: false, currentMember: null, onClose: vi.fn(), onSaved: vi.fn() },
        roomModeration: { open: false, onClose: vi.fn(), onSaved: vi.fn() },
        watchParty: {
            active: null,
            screenShareEnabled: true,
            onClose: vi.fn(),
            onLeave: vi.fn(),
            onEnd: vi.fn(),
            onTransferControl: vi.fn(),
            onKick: vi.fn(),
            onIdentify: vi.fn(),
        },
        invite: { open: false, existingMemberIds: new Set<string>(), onClose: vi.fn(), onInvited: vi.fn() },
        ...overrides,
    };
}

function renderOverlays(overrides: Partial<RoomOverlaysProps> = {}) {
    const props = makeProps(overrides);
    const result = renderWithProviders(<RoomOverlays {...props} />);

    return { ...result, props };
}

describe("RoomOverlays panels", () => {
    it("keeps the search panel out of the tree until it is opened", () => {
        // given
        const infoPanel = { tab: null, onTabChange: vi.fn(), onClose: vi.fn() };

        // when
        renderOverlays({ infoPanel });

        // then
        expect(screen.queryByTestId("info-panel")).not.toBeInTheDocument();
    });

    it("mounts the info panel against the current room once a tab is opened", () => {
        // given
        const infoPanel = { tab: "search" as const, onTabChange: vi.fn(), onClose: vi.fn() };

        // when
        renderOverlays({ infoPanel });

        // then
        expect(screen.getByTestId("info-panel")).toHaveTextContent("room-1");
    });

    it("opens the info panel on the tab it was asked for", () => {
        // given
        const infoPanel = { tab: "pins" as const, onTabChange: vi.fn(), onClose: vi.fn() };

        // when
        renderOverlays({ infoPanel });

        // then
        expect(screen.getByTestId("info-panel")).toHaveAttribute("data-tab", "pins");
    });

    it("lets a moderator unpin from the pins tab", () => {
        // when
        renderOverlays({ canModerateRoom: true, infoPanel: { tab: "pins", onTabChange: vi.fn(), onClose: vi.fn() } });

        // then
        expect(screen.getByTestId("info-panel")).toHaveAttribute("data-can-unpin", "true");
    });

    it("sends a pinned image up to the lightbox", async () => {
        // given
        const user = userEvent.setup();
        const onLightbox = vi.fn();

        // when
        renderOverlays({ onLightbox, infoPanel: { tab: "pins", onTabChange: vi.fn(), onClose: vi.fn() } });
        await user.click(screen.getByTestId("info-panel"));

        // then
        expect(onLightbox).toHaveBeenCalledWith("/img.png");
    });

    it("hands the room moderation dialog the room it moderates", () => {
        // given
        const roomModeration = { open: true, onClose: vi.fn(), onSaved: vi.fn() };

        // when
        renderOverlays({ roomModeration });

        // then
        expect(screen.getByTestId("room-moderation")).toHaveTextContent("Tea Parlour");
        expect(screen.getByTestId("room-moderation")).toHaveAttribute("data-open", "true");
    });
});

describe("RoomOverlays profile dialog identity", () => {
    it("rebuilds the profile dialog when the member behind it changes", () => {
        // given
        const first = makeProps({
            editProfile: { open: true, currentMember: makeMember("m-1"), onClose: vi.fn(), onSaved: vi.fn() },
        });
        const { rerender } = renderWithProviders(<RoomOverlays {...first} />);
        const before = screen.getByTestId("edit-profile");

        // when
        rerender(<RoomOverlays {...first} editProfile={{ ...first.editProfile, currentMember: makeMember("m-2") }} />);

        // then
        expect(screen.getByTestId("edit-profile")).not.toBe(before);
        expect(screen.getByTestId("edit-profile")).toHaveTextContent("m-2");
    });

    it("keeps the same profile dialog when nothing behind it changed", () => {
        // given
        const member = makeMember("m-1");
        const props = makeProps({
            editProfile: { open: true, currentMember: member, onClose: vi.fn(), onSaved: vi.fn() },
        });
        const { rerender } = renderWithProviders(<RoomOverlays {...props} />);
        const before = screen.getByTestId("edit-profile");

        // when
        rerender(<RoomOverlays {...props} />);

        // then
        expect(screen.getByTestId("edit-profile")).toBe(before);
    });
});

describe("RoomOverlays watch party", () => {
    it("stays out of the tree while no session is open", () => {
        // when
        renderOverlays();

        // then
        expect(screen.queryByTestId("watch-party-modal")).not.toBeInTheDocument();
    });

    it("knows the viewer started the party they opened", async () => {
        // given
        const active = makeActiveSession({ started_by: viewer.id });

        // when
        renderOverlays({ watchParty: { ...makeProps().watchParty, active } });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-starter", "true");
    });

    it("knows the viewer only joined somebody else's party", async () => {
        // given
        const active = makeActiveSession({ started_by: "someone-else" });

        // when
        renderOverlays({ watchParty: { ...makeProps().watchParty, active } });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-starter", "false");
    });

    it("passes the staff flag through to the party", async () => {
        // given
        const active = makeActiveSession();

        // when
        renderOverlays({ viewerIsStaff: true, watchParty: { ...makeProps().watchParty, active } });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-staff", "true");
    });
});

describe("RoomOverlays watch party voice gate", () => {
    it("offers watch party voice when the site allows voice and screen share is on", async () => {
        // given
        const active = makeActiveSession();

        // when
        renderOverlays({
            voiceEnabled: true,
            watchParty: { ...makeProps().watchParty, active, screenShareEnabled: true },
        });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-voice", "true");
    });

    it("obeys the site voice setting inside a watch party", async () => {
        // given
        const active = makeActiveSession();

        // when
        renderOverlays({
            voiceEnabled: false,
            watchParty: { ...makeProps().watchParty, active, screenShareEnabled: true },
        });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-voice", "false");
    });

    it("keeps watch party voice off when screen share is off", async () => {
        // given
        const active = makeActiveSession();

        // when
        renderOverlays({
            voiceEnabled: true,
            watchParty: { ...makeProps().watchParty, active, screenShareEnabled: false },
        });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-voice", "false");
    });
});

describe("RoomOverlays invites", () => {
    it("tells the invite modal who is already in the room", () => {
        // given
        const invite = {
            open: true,
            existingMemberIds: new Set(["m-1", "m-2"]),
            onClose: vi.fn(),
            onInvited: vi.fn(),
        };

        // when
        renderOverlays({ invite });

        // then
        expect(screen.getByTestId("invite-modal")).toHaveAttribute("data-existing", "m-1,m-2");
    });

    it("reports the invite result back to the page", async () => {
        // given
        const user = userEvent.setup();
        const onInvited = vi.fn();
        const invite = { open: true, existingMemberIds: new Set<string>(), onClose: vi.fn(), onInvited };

        // when
        renderOverlays({ invite });
        await user.click(screen.getByTestId("invite-modal"));

        // then
        expect(onInvited).toHaveBeenCalledWith({ invited_count: 2, skipped_count: 0 });
    });
});
