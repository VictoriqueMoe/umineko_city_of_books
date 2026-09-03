import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Room } from "livekit-client";
import { roomCapabilities } from "../../domain/chat/roomPolicy";
import type { RoomController } from "../../hooks/useRoomController";
import { makeRoomController } from "../../hooks/useRoomController.fixture";
import {
    makeChatRoom,
    makePublicUser,
    makeRoomMember,
    makeUser,
    makeWatchPartySession,
} from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { ChatRoom, ChatRoomMember, User, UserProfile } from "../../types/api";
import { RoomPage } from "./RoomPage";

const mocks = vi.hoisted(() => ({
    useRoomController: vi.fn(),
    useIsMobile: vi.fn(),
    forceMute: vi.fn(),
}));

vi.mock("../../hooks/useRoomController", () => ({ useRoomController: mocks.useRoomController }));

vi.mock("../../hooks/useIsMobile", () => ({ useIsMobile: mocks.useIsMobile }));

vi.mock("../../hooks/mutations/chat", async importOriginal => {
    const actual = await importOriginal<typeof import("../../hooks/mutations/chat")>();
    return {
        ...actual,
        useForceMuteVoiceParticipant: (roomId: string | null | undefined) => ({
            mutate: (variables: { userId: string; muted: boolean }, options?: { onError?: (err: unknown) => void }) =>
                mocks.forceMute(roomId, variables, options),
        }),
    };
});

vi.mock("../../components/chat/mobile/MobileRoomView", () => ({
    MobileRoomView: () => <div data-testid="mobile-room-view" />,
}));

vi.mock("../../components/chat/MessageList/MessageList", () => ({
    MessageList: () => <div data-testid="room-messages" />,
}));

vi.mock("../../components/chat/ChatComposer/ChatComposer", () => ({
    ChatComposer: (props: { roomId: string | null; extraActions?: React.ReactNode }) => (
        <div data-testid="composer" data-room={String(props.roomId)}>
            {props.extraActions}
        </div>
    ),
}));

vi.mock("../../components/chat/EditRoomProfileDialog/EditRoomProfileDialog", () => ({
    EditRoomProfileDialog: (props: { isOpen: boolean }) =>
        props.isOpen ? <div data-testid="edit-profile-dialog" /> : null,
}));

vi.mock("../../components/chat/RoomModerationDialog/RoomModerationDialog", () => ({
    RoomModerationDialog: (props: { isOpen: boolean }) =>
        props.isOpen ? <div data-testid="moderation-dialog" /> : null,
}));

vi.mock("../../components/chat/InviteMembersModal/InviteMembersModal", () => ({
    InviteMembersModal: (props: { isOpen: boolean }) => (props.isOpen ? <div data-testid="invite-modal" /> : null),
}));

vi.mock("../../components/chat/RoomInfoPanel/RoomInfoPanel", () => ({
    RoomInfoPanel: (props: { tab: string | null }) => (props.tab ? <div data-testid={`${props.tab}-panel`} /> : null),
}));

vi.mock("../../components/chat/WatchParty/WatchPartyButton", () => ({
    WatchPartyButton: (props: { enabled: boolean }) => (
        <div data-testid="watch-party-button">{String(props.enabled)}</div>
    ),
}));

vi.mock("../../components/chat/WatchParty/WatchPartyModal", () => ({
    WatchPartyModal: (props: { isStarter: boolean; voiceEnabled: boolean }) => (
        <div data-testid="watch-party-modal" data-voice={String(props.voiceEnabled)}>
            {String(props.isStarter)}
        </div>
    ),
}));

vi.mock("../../components/chat/Voice/VoiceBar", () => ({
    VoiceBar: (props: { canModerate: boolean; onForceMute: (id: string, muted: boolean) => void }) => (
        <button type="button" data-moderator={String(props.canModerate)} onClick={() => props.onForceMute("u9", true)}>
            voice bar
        </button>
    ),
}));

vi.mock("../../components/chat/Voice/VoiceButton", () => ({
    VoiceButton: (props: { enabled: boolean }) => <div data-testid="voice-button">{String(props.enabled)}</div>,
}));

vi.mock("../../components/Lightbox/Lightbox", () => ({
    Lightbox: (props: { src: string }) => <div data-testid="lightbox">{props.src}</div>,
}));

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });

function makeMemberUser(overrides: Partial<User> = {}): User {
    return makePublicUser({ id: "member-1", ...overrides });
}

function makeMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return makeRoomMember({ user: makeMemberUser(), ...overrides });
}

function makeActiveSession(startedBy: string): RoomController["watchParty"]["activeSession"] {
    return { session: makeWatchPartySession({ started_by: startedBy }), embedURL: "", hasControl: false };
}

interface ControllerOptions {
    user?: UserProfile | null;
    loading?: boolean;
    room?: ChatRoom | null;
    roomId?: string | null;
    members?: ChatRoomMember[];
    memberGroups?: { label: string; members: ChatRoomMember[] }[];
    presenceMapMerged?: RoomController["members"]["presence"];
    onlineIds?: string[];
    currentMember?: ChatRoomMember | null;
    sidebarCollapsed?: boolean;
    descExpanded?: boolean;
    typingNames?: string[];
    voiceStatus?: RoomController["voice"]["status"];
    voiceRoom?: RoomController["voice"]["room"];
    voiceEnabled?: boolean;
    watchPartyEnabled?: boolean;
    activeSession?: RoomController["watchParty"]["activeSession"];
    invitedPartyMissing?: boolean;
    lightboxSrc?: string | null;
    toast?: string | null;
    busy?: string | null;
    joining?: boolean;
    openMemberMenu?: string | null;
    editProfileOpen?: boolean;
    inviteModalOpen?: boolean;
    moderationDialogOpen?: boolean;
    panelTab?: "search" | "pins" | null;
    nicknameDialogTarget?: ChatRoomMember | null;
    nicknameDialogError?: string;
    nicknameDialogSaving?: boolean;
    timeoutDialogTarget?: ChatRoomMember | null;
    timeoutDialogError?: string;
    timeoutDialogSaving?: boolean;
}

function stubController(options: ControllerOptions = {}) {
    const handlers = {
        backToRooms: vi.fn(),
        setMobileView: vi.fn(),
        toggleSidebar: vi.fn(),
        toggleDescExpanded: vi.fn(),
        setReplyingTo: vi.fn(),
        setLightboxSrc: vi.fn(),
        setToast: vi.fn(),
        openPanel: vi.fn(),
        setEditProfileOpen: vi.fn(),
        setInviteModalOpen: vi.fn(),
        setModerationDialogOpen: vi.fn(),
        setOpenMemberMenu: vi.fn(),
        setRoom: vi.fn(),
        setMembers: vi.fn(),
        notifyTyping: vi.fn(),
        setNicknameDialogTarget: vi.fn(),
        setNicknameDialogValue: vi.fn(),
        setTimeoutDialogTarget: vi.fn(),
        setTimeoutDialogAmount: vi.fn(),
        setTimeoutDialogUnit: vi.fn(),
        openNicknameDialog: vi.fn(),
        openTimeoutDialog: vi.fn(),
        handleSentMessage: vi.fn(),
        handleJoin: vi.fn(),
        handleModSetNickname: vi.fn(),
        handleModUnlockNickname: vi.fn(),
        handleSetTimeout: vi.fn(),
        handleClearTimeout: vi.fn(),
        handleKick: vi.fn(),
        handleBan: vi.fn(),
        handleToggleMute: vi.fn(),
        handleLeave: vi.fn(),
        handleDelete: vi.fn(),
        handleJumpToMessage: vi.fn(),
        handleEditLast: vi.fn(),
        watchPartyStart: vi.fn(),
        watchPartyClose: vi.fn(),
    };

    const members = options.members ?? [];
    const onlineIds = new Set(options.onlineIds ?? []);
    const base = makeRoomController();
    const roomData = options.room === undefined ? makeChatRoom() : options.room;
    const viewerUser = options.user === undefined ? viewer : options.user;

    const controller = makeRoomController({
        capabilities: roomCapabilities(roomData, viewerUser),
        room: {
            ...base.room,
            data: roomData,
            id: options.roomId === undefined ? "room-1" : (options.roomId ?? undefined),
            loading: options.loading ?? false,
            joining: options.joining ?? false,
            set: handlers.setRoom,
            join: handlers.handleJoin,
            toggleMute: handlers.handleToggleMute,
            leave: handlers.handleLeave,
            remove: handlers.handleDelete,
            backToRooms: handlers.backToRooms,
        },
        session: {
            ...base.session,
            viewer: viewerUser,
            setReplyingTo: handlers.setReplyingTo,
            typingNames: options.typingNames ?? [],
            notifyTyping: handlers.notifyTyping,
            onSent: handlers.handleSentMessage,
            editLast: handlers.handleEditLast,
        },
        members: {
            ...base.members,
            list: members,
            groups: options.memberGroups ?? [{ label: "Members", members }],
            presence: options.presenceMapMerged ?? {},
            onlineWeight: (id: string) => (onlineIds.has(id) ? 0 : 1),
            current: options.currentMember ?? null,
            set: handlers.setMembers,
        },
        moderation: {
            ...base.moderation,
            busy: options.busy ?? null,
            openMemberMenu: options.openMemberMenu ?? null,
            setOpenMemberMenu: handlers.setOpenMemberMenu,
            nicknameDialogTarget: options.nicknameDialogTarget ?? null,
            setNicknameDialogTarget: handlers.setNicknameDialogTarget,
            setNicknameDialogValue: handlers.setNicknameDialogValue,
            nicknameDialogError: options.nicknameDialogError ?? "",
            nicknameDialogSaving: options.nicknameDialogSaving ?? false,
            timeoutDialogTarget: options.timeoutDialogTarget ?? null,
            setTimeoutDialogTarget: handlers.setTimeoutDialogTarget,
            setTimeoutDialogAmount: handlers.setTimeoutDialogAmount,
            setTimeoutDialogUnit: handlers.setTimeoutDialogUnit,
            timeoutDialogError: options.timeoutDialogError ?? "",
            timeoutDialogSaving: options.timeoutDialogSaving ?? false,
            formatTimeoutUntil: (value?: string) => `until ${value ?? "never"}`,
            openNicknameDialog: handlers.openNicknameDialog,
            openTimeoutDialog: handlers.openTimeoutDialog,
            handleModSetNickname: handlers.handleModSetNickname,
            handleModUnlockNickname: handlers.handleModUnlockNickname,
            handleSetTimeout: handlers.handleSetTimeout,
            handleClearTimeout: handlers.handleClearTimeout,
            handleKick: handlers.handleKick,
            handleBan: handlers.handleBan,
        },
        prefs: {
            ...base.prefs,
            sidebarCollapsed: options.sidebarCollapsed ?? false,
            toggleSidebar: handlers.toggleSidebar,
            descExpanded: options.descExpanded ?? false,
            toggleDescExpanded: handlers.toggleDescExpanded,
            setMobileView: handlers.setMobileView,
        },
        anchor: {
            ...base.anchor,
            jumpTo: handlers.handleJumpToMessage,
        },
        voice: {
            ...base.voice,
            status: options.voiceStatus ?? "idle",
            room: options.voiceRoom ?? null,
            enabled: options.voiceEnabled ?? true,
        },
        watchParty: {
            ...base.watchParty,
            enabled: options.watchPartyEnabled ?? true,
            activeSession: options.activeSession ?? null,
            start: handlers.watchPartyStart,
            close: handlers.watchPartyClose,
            invitedPartyMissing: options.invitedPartyMissing ?? false,
        },
        panels: {
            ...base.panels,
            panelTab: options.panelTab ?? null,
            openPanel: handlers.openPanel,
            lightboxSrc: options.lightboxSrc ?? null,
            setLightboxSrc: handlers.setLightboxSrc,
            editProfileOpen: options.editProfileOpen ?? false,
            setEditProfileOpen: handlers.setEditProfileOpen,
            inviteModalOpen: options.inviteModalOpen ?? false,
            setInviteModalOpen: handlers.setInviteModalOpen,
            moderationDialogOpen: options.moderationDialogOpen ?? false,
            setModerationDialogOpen: handlers.setModerationDialogOpen,
        },
        toast: {
            message: options.toast ?? null,
            show: handlers.setToast,
        },
    });

    mocks.useRoomController.mockReturnValue(controller);

    return handlers;
}

function renderRoom(options: ControllerOptions = {}) {
    const handlers = stubController(options);
    const result = renderWithProviders(<RoomPage />, { user: options.user === undefined ? viewer : options.user });

    return { ...result, ...handlers };
}

beforeEach(() => {
    mocks.useIsMobile.mockReturnValue(false);
    mocks.forceMute.mockReset();
});

describe("RoomPage gates", () => {
    it("shows nothing at all to a signed out visitor", () => {
        // given
        const user = null;

        // when
        const { container } = renderRoom({ user });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("waits while the room is being loaded", () => {
        // given
        const loading = true;

        // when
        renderRoom({ loading });

        // then
        expect(screen.getByText("Loading room...")).toBeInTheDocument();
    });

    it("offers a way in when the viewer is not a member", () => {
        // given
        const room = null;

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText("You're not a member of this room.")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Try to Join" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Back to Rooms" })).toBeInTheDocument();
    });

    it("tries to join when the outsider asks to", async () => {
        // given
        const user = userEvent.setup();
        const { handleJoin } = renderRoom({ room: null });

        // when
        await user.click(screen.getByRole("button", { name: "Try to Join" }));

        // then
        expect(handleJoin).toHaveBeenCalledTimes(1);
    });

    it("says it is joining while the request is in flight", () => {
        // given
        const joining = true;

        // when
        renderRoom({ room: null, joining });

        // then
        expect(screen.getByRole("button", { name: "Joining..." })).toBeDisabled();
    });

    it("hides the join button when there is no room in the address at all", () => {
        // given
        const roomId = null;

        // when
        renderRoom({ room: null, roomId });

        // then
        expect(screen.queryByRole("button", { name: "Try to Join" })).not.toBeInTheDocument();
    });

    it("passes on why the join failed", () => {
        // given
        const toast = "You are banned from this room.";

        // when
        renderRoom({ room: null, toast });

        // then
        expect(screen.getByText("You are banned from this room.")).toBeInTheDocument();
    });

    it("hands the room to the mobile view on a small screen", () => {
        // given
        mocks.useIsMobile.mockReturnValue(true);

        // when
        renderRoom();

        // then
        expect(screen.getByTestId("mobile-room-view")).toBeInTheDocument();
        expect(screen.queryByTestId("room-messages")).not.toBeInTheDocument();
    });
});

describe("RoomPage header", () => {
    it("names the room and describes who can see it", () => {
        // given
        const room = makeChatRoom({ name: "Rokkenjima", member_count: 7, is_public: true });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        expect(screen.getByText(/7 members/)).toBeInTheDocument();
        expect(screen.getByText(/public/)).toBeInTheDocument();
    });

    it("marks a private room as private", () => {
        // given
        const room = makeChatRoom({ is_public: false });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText(/private/)).toBeInTheDocument();
    });

    it("badges a staff room and a roleplay room", () => {
        // given
        const room = makeChatRoom({ is_system: true, is_rp: true });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText("Staff")).toBeInTheDocument();
        expect(screen.getByText("RP")).toBeInTheDocument();
    });

    it("opens the message search when asked", async () => {
        // given
        const user = userEvent.setup();
        const { openPanel } = renderRoom();

        // when
        await user.click(screen.getByRole("button", { name: "Search messages" }));

        // then
        expect(openPanel).toHaveBeenCalledWith("search");
    });

    it("opens the pinned messages when asked", async () => {
        // given
        const user = userEvent.setup();
        const { openPanel } = renderRoom();

        // when
        await user.click(screen.getByRole("button", { name: "Pinned messages" }));

        // then
        expect(openPanel).toHaveBeenCalledWith("pins");
    });

    it("keeps the search panel closed until it is opened", () => {
        // given
        const panelTab = null;

        // when
        renderRoom({ panelTab });

        // then
        expect(screen.queryByTestId("search-panel")).not.toBeInTheDocument();
    });

    it("shows the search panel once it is open", () => {
        // given
        const panelTab = "search" as const;

        // when
        renderRoom({ panelTab });

        // then
        expect(screen.getByTestId("search-panel")).toBeInTheDocument();
    });
});

describe("RoomPage room info", () => {
    it("leaves the info panel out when there is nothing to say", () => {
        // given
        const room = makeChatRoom({ description: "", tags: [] });

        // when
        renderRoom({ room });

        // then
        expect(screen.queryByRole("button", { name: /Show info/ })).not.toBeInTheDocument();
    });

    it("offers the info panel when the room has a description", () => {
        // given
        const room = makeChatRoom({ description: "Where the witches take tea" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "Show info ▼" })).toBeInTheDocument();
        expect(screen.getByText("Where the witches take tea")).toBeInTheDocument();
    });

    it("lists the room's tags", () => {
        // given
        const room = makeChatRoom({ tags: ["horror", "spoilers"] });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText("#horror")).toBeInTheDocument();
        expect(screen.getByText("#spoilers")).toBeInTheDocument();
    });

    it("collapses the info panel when it is already expanded", async () => {
        // given
        const user = userEvent.setup();
        const { toggleDescExpanded } = renderRoom({ room: makeChatRoom({ description: "Tea" }), descExpanded: true });

        // when
        await user.click(screen.getByRole("button", { name: "Hide info ▲" }));

        // then
        expect(toggleDescExpanded).toHaveBeenCalledTimes(1);
    });
});

describe("RoomPage members", () => {
    it("counts the members in the sidebar", () => {
        // given
        const members = [makeMember(), makeMember({ user: makeMemberUser({ id: "member-2", username: "ange" }) })];

        // when
        renderRoom({ members });

        // then
        expect(screen.getByText("2")).toBeInTheDocument();
    });

    it("groups the members under an online heading when they are around", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members, onlineIds: ["member-1"] });

        // then
        expect(screen.getByText("Online")).toBeInTheDocument();
    });

    it("groups the members under an offline heading when they are away", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members });

        // then
        expect(screen.getByText("Offline")).toBeInTheDocument();
    });

    it("skips the status heading inside the voice group", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members, memberGroups: [{ label: "In Voice", members }] });

        // then
        expect(screen.getByText("In Voice")).toBeInTheDocument();
        expect(screen.queryByText("Offline")).not.toBeInTheDocument();
    });

    it("says whether a member is watching the room right now", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members, presenceMapMerged: { "member-1": "active" } });

        // then
        expect(screen.getByLabelText("Active in this room")).toBeInTheDocument();
    });

    it("says when a member has the tab in the background", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members, presenceMapMerged: { "member-1": "idle" } });

        // then
        expect(screen.getByLabelText("Idle or tab in background")).toBeInTheDocument();
    });

    it("badges the host and a ghost member", () => {
        // given
        const members = [makeMember({ role: "host", ghost: true })];

        // when
        renderRoom({ members });

        // then
        expect(screen.getByText("Host")).toBeInTheDocument();
        expect(screen.getByTitle(/Ghost member/)).toBeInTheDocument();
    });

    it("marks a member who is timed out", () => {
        // given
        const members = [makeMember({ timeout_until: "2026-02-01T13:00:00Z" })];

        // when
        renderRoom({ members });

        // then
        expect(screen.getByLabelText("Timed out until until 2026-02-01T13:00:00Z")).toBeInTheDocument();
    });

    it("lets the viewer edit their own profile in the room", async () => {
        // given
        const user = userEvent.setup();
        const members = [makeMember({ user: makeMemberUser({ id: viewer.id, username: "battler" }) })];
        const { setEditProfileOpen } = renderRoom({ members });

        // when
        await user.click(screen.getByRole("button", { name: "Edit profile in this room" }));

        // then
        expect(setEditProfileOpen).toHaveBeenCalledWith(true);
    });

    it("shows a member's nickname in place of their display name", () => {
        // given
        const members = [makeMember({ nickname: "The Golden Witch" })];

        // when
        renderRoom({ members });

        // then
        expect(screen.getByText("The Golden Witch")).toBeInTheDocument();
        expect(screen.queryByText("Beatrice")).not.toBeInTheDocument();
    });
});

describe("RoomPage moderation", () => {
    it("gives an ordinary member no moderator actions", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members, room: makeChatRoom({ viewer_role: "member" }) });

        // then
        expect(screen.queryByRole("button", { name: "Moderator actions" })).not.toBeInTheDocument();
    });

    it("gives the host moderator actions over an ordinary member", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members, room: makeChatRoom({ viewer_role: "host" }) });

        // then
        expect(screen.getByRole("button", { name: "Moderator actions" })).toBeInTheDocument();
    });

    it("opens the moderator menu when it is clicked", async () => {
        // given
        const user = userEvent.setup();
        const { setOpenMemberMenu } = renderRoom({
            members: [makeMember()],
            room: makeChatRoom({ viewer_role: "host" }),
        });

        // when
        await user.click(screen.getByRole("button", { name: "Moderator actions" }));

        // then
        expect(setOpenMemberMenu).toHaveBeenCalledTimes(1);
    });

    it("offers the host kick and ban but no nickname change", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members, room: makeChatRoom({ viewer_role: "host" }), openMemberMenu: "member-1" });

        // then
        expect(screen.getByRole("button", { name: "Kick member" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Ban from room" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Change nickname" })).not.toBeInTheDocument();
    });

    it("offers site staff the nickname controls too", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: "moderator" });

        // when
        renderRoom({
            user,
            members: [makeMember({ nickname_locked: true })],
            room: makeChatRoom({ viewer_role: "member" }),
            openMemberMenu: "member-1",
        });

        // then
        expect(screen.getByRole("button", { name: "Change nickname" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Reset/unlock nickname" })).toBeInTheDocument();
    });

    it("kicks a member when the host chooses to", async () => {
        // given
        const user = userEvent.setup();
        const { handleKick } = renderRoom({
            members: [makeMember()],
            room: makeChatRoom({ viewer_role: "host" }),
            openMemberMenu: "member-1",
        });

        // when
        await user.click(screen.getByRole("button", { name: "Kick member" }));

        // then
        expect(handleKick).toHaveBeenCalledWith("member-1");
    });

    it("bans a member when the host chooses to", async () => {
        // given
        const user = userEvent.setup();
        const { handleBan } = renderRoom({
            members: [makeMember()],
            room: makeChatRoom({ viewer_role: "host" }),
            openMemberMenu: "member-1",
        });

        // when
        await user.click(screen.getByRole("button", { name: "Ban from room" }));

        // then
        expect(handleBan).toHaveBeenCalledWith("member-1");
    });

    it("opens the timeout dialog for a member", async () => {
        // given
        const user = userEvent.setup();
        const { openTimeoutDialog } = renderRoom({
            members: [makeMember()],
            room: makeChatRoom({ viewer_role: "host" }),
            openMemberMenu: "member-1",
        });

        // when
        await user.click(screen.getByRole("button", { name: "Set timeout" }));

        // then
        expect(openTimeoutDialog).toHaveBeenCalledTimes(1);
    });

    it("clears an existing timeout", async () => {
        // given
        const user = userEvent.setup();
        const { handleClearTimeout } = renderRoom({
            members: [makeMember({ timeout_until: "2026-02-01T13:00:00Z" })],
            room: makeChatRoom({ viewer_role: "host" }),
            openMemberMenu: "member-1",
        });

        // when
        await user.click(screen.getByRole("button", { name: "Remove timeout" }));

        // then
        expect(handleClearTimeout).toHaveBeenCalledWith("member-1");
    });

    it("leaves a system room unmoderated", () => {
        // given
        const room = makeChatRoom({ is_system: true, viewer_role: "host" });

        // when
        renderRoom({ members: [makeMember()], room });

        // then
        expect(screen.queryByRole("button", { name: "Moderator actions" })).not.toBeInTheDocument();
    });
});

describe("RoomPage sidebar actions", () => {
    it("offers to mute the room's notifications", async () => {
        // given
        const user = userEvent.setup();
        const { handleToggleMute } = renderRoom();

        // when
        await user.click(screen.getByRole("button", { name: "Mute notifications" }));

        // then
        expect(handleToggleMute).toHaveBeenCalledTimes(1);
    });

    it("offers to unmute a room that is already muted", () => {
        // given
        const room = makeChatRoom({ viewer_muted: true });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "Unmute notifications" })).toBeInTheDocument();
    });

    it("gives the host moderation and deletion", () => {
        // given
        const room = makeChatRoom({ viewer_role: "host" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "Moderation" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete Room" })).toBeInTheDocument();
    });

    it("offers an ordinary member the door instead", () => {
        // given
        const room = makeChatRoom({ viewer_role: "member" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "Leave Room" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete Room" })).not.toBeInTheDocument();
    });

    it("keeps deletion and leaving away from a system room", () => {
        // given
        const room = makeChatRoom({ is_system: true, viewer_role: "host" });

        // when
        renderRoom({ room });

        // then
        expect(screen.queryByRole("button", { name: "Delete Room" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Leave Room" })).not.toBeInTheDocument();
    });

    it("shows the invite button to the host of an ordinary room", () => {
        // given
        const room = makeChatRoom({ viewer_role: "host" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "+ Invite" })).toBeInTheDocument();
    });

    it("hides the invite button from a plain member", () => {
        // given
        const room = makeChatRoom({ viewer_role: "member" });

        // when
        renderRoom({ room });

        // then
        expect(screen.queryByRole("button", { name: "+ Invite" })).not.toBeInTheDocument();
    });

    it("shows the invite button to site staff who do not host the room", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: "moderator" });
        const room = makeChatRoom({ viewer_role: "member" });

        // when
        renderRoom({ user, room });

        // then
        expect(screen.getByRole("button", { name: "+ Invite" })).toBeInTheDocument();
    });

    it("hides the invite button from site staff in a system room", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: "moderator" });
        const room = makeChatRoom({ viewer_role: "member", is_system: true });

        // when
        renderRoom({ user, room });

        // then
        expect(screen.queryByRole("button", { name: "+ Invite" })).not.toBeInTheDocument();
    });

    it("collapses the member sidebar when asked", async () => {
        // given
        const user = userEvent.setup();
        const { toggleSidebar } = renderRoom();

        // when
        await user.click(screen.getByRole("button", { name: "Hide members" }));

        // then
        expect(toggleSidebar).toHaveBeenCalledTimes(1);
    });

    it("offers a rail to bring the sidebar back once it is collapsed", () => {
        // given
        const sidebarCollapsed = true;

        // when
        renderRoom({ sidebarCollapsed });

        // then
        expect(screen.getByRole("button", { name: "Show members" })).toBeInTheDocument();
    });

    it("goes back to the rooms list from the sidebar", async () => {
        // given
        const user = userEvent.setup();
        const { backToRooms } = renderRoom();

        // when
        await user.click(screen.getByRole("button", { name: "Back to rooms" }));

        // then
        expect(backToRooms).toHaveBeenCalledOnce();
    });

    it("says it is deleting while the room is being removed", () => {
        // given
        const busy = "delete";

        // when
        renderRoom({ room: makeChatRoom({ viewer_role: "host" }), busy });

        // then
        expect(screen.getByRole("button", { name: "Deleting..." })).toBeDisabled();
    });
});

describe("RoomPage voice and watch party", () => {
    it("keeps the voice bar away until the call is connected", () => {
        // given
        const voiceStatus = "connecting";

        // when
        renderRoom({ voiceStatus, voiceRoom: new Room() });

        // then
        expect(screen.queryByRole("button", { name: "voice bar" })).not.toBeInTheDocument();
    });

    it("shows the voice bar once the call is connected", async () => {
        // given
        const voiceStatus = "connected";

        // when
        renderRoom({ voiceStatus, voiceRoom: new Room(), room: makeChatRoom({ viewer_role: "host" }) });

        // then
        expect(await screen.findByRole("button", { name: "voice bar" })).toHaveAttribute("data-moderator", "true");
    });

    it("force mutes a voice participant in this room", async () => {
        // given
        const pointer = userEvent.setup();
        renderRoom({ voiceStatus: "connected", voiceRoom: new Room(), room: makeChatRoom({ viewer_role: "host" }) });
        const bar = await screen.findByRole("button", { name: "voice bar" });

        // when
        await pointer.click(bar);

        // then
        expect(mocks.forceMute).toHaveBeenCalledWith("room-1", { userId: "u9", muted: true }, expect.anything());
    });

    it("tells the moderator when a force mute did not take", async () => {
        // given
        mocks.forceMute.mockImplementation(
            (
                _roomId: string | null | undefined,
                _variables: { userId: string; muted: boolean },
                options?: { onError?: (err: unknown) => void },
            ) => options?.onError?.(new Error("LiveKit said no")),
        );
        const pointer = userEvent.setup();
        const { setToast } = renderRoom({
            voiceStatus: "connected",
            voiceRoom: new Room(),
            room: makeChatRoom({ viewer_role: "host" }),
        });
        const bar = await screen.findByRole("button", { name: "voice bar" });

        // when
        await pointer.click(bar);

        // then
        await waitFor(() => {
            expect(setToast).toHaveBeenCalledWith("LiveKit said no");
        });
    });

    it("gives an ordinary room the voice and watch party buttons", () => {
        // given
        const room = makeChatRoom({ is_system: false });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByTestId("voice-button")).toBeInTheDocument();
        expect(screen.getByTestId("watch-party-button")).toBeInTheDocument();
    });

    it("keeps the voice and watch party buttons out of a system room", () => {
        // given
        const room = makeChatRoom({ is_system: true });

        // when
        renderRoom({ room });

        // then
        expect(screen.queryByTestId("voice-button")).not.toBeInTheDocument();
        expect(screen.queryByTestId("watch-party-button")).not.toBeInTheDocument();
    });

    it("opens the watch party window for an active session", async () => {
        // given
        const activeSession = makeActiveSession(viewer.id);

        // when
        renderRoom({ activeSession });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveTextContent("true");
    });

    it("knows the viewer did not start somebody else's watch party", async () => {
        // given
        const activeSession = makeActiveSession("someone-else");

        // when
        renderRoom({ activeSession });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveTextContent("false");
    });

    it("offers watch party voice when the site allows voice and screen share is on", async () => {
        // given
        const activeSession = makeActiveSession(viewer.id);

        // when
        renderRoom({ activeSession, voiceEnabled: true });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-voice", "true");
    });

    it("obeys the site voice setting inside a watch party", async () => {
        // given
        const voiceEnabled = false;

        // when
        renderRoom({ activeSession: makeActiveSession(viewer.id), voiceEnabled });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-voice", "false");
    });

    it("says so when the invited watch party has already ended", () => {
        // given
        const invitedPartyMissing = true;

        // when
        renderRoom({ invitedPartyMissing });

        // then
        expect(screen.getByText("That watch party has ended.")).toBeInTheDocument();
    });
});

describe("RoomPage dialogs", () => {
    it("keeps the nickname dialog shut until a target is chosen", () => {
        // given
        const nicknameDialogTarget = null;

        // when
        renderRoom({ nicknameDialogTarget });

        // then
        expect(screen.queryByRole("heading", { name: /Change nickname for/ })).not.toBeInTheDocument();
    });

    it("names the member whose nickname is being changed", () => {
        // given
        const nicknameDialogTarget = makeMember();

        // when
        renderRoom({ nicknameDialogTarget });

        // then
        expect(screen.getByRole("heading", { name: "Change nickname for Beatrice" })).toBeInTheDocument();
    });

    it("saves the new nickname when asked", async () => {
        // given
        const user = userEvent.setup();
        const { handleModSetNickname } = renderRoom({ nicknameDialogTarget: makeMember() });

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(handleModSetNickname).toHaveBeenCalledTimes(1);
    });

    it("reports why a nickname could not be saved", () => {
        // given
        const nicknameDialogError = "That name is taken.";

        // when
        renderRoom({ nicknameDialogTarget: makeMember(), nicknameDialogError });

        // then
        expect(screen.getByText("That name is taken.")).toBeInTheDocument();
    });

    it("names the member being timed out and offers the units", () => {
        // given
        const timeoutDialogTarget = makeMember();

        // when
        renderRoom({ timeoutDialogTarget });

        // then
        expect(screen.getByRole("heading", { name: "Set timeout for Beatrice" })).toBeInTheDocument();
        expect(screen.getByRole("option", { name: "centuries" })).toBeInTheDocument();
    });

    it("sets the timeout when asked", async () => {
        // given
        const user = userEvent.setup();
        const { handleSetTimeout } = renderRoom({ timeoutDialogTarget: makeMember() });

        // when
        await user.click(screen.getByRole("button", { name: "Set timeout" }));

        // then
        expect(handleSetTimeout).toHaveBeenCalledTimes(1);
    });

    it("shows the invite modal only once it is opened", () => {
        // given
        const inviteModalOpen = true;

        // when
        renderRoom({ inviteModalOpen, room: makeChatRoom({ viewer_role: "host" }) });

        // then
        expect(screen.getByTestId("invite-modal")).toBeInTheDocument();
    });

    it("shows the moderation dialog only once it is opened", () => {
        // given
        const moderationDialogOpen = true;

        // when
        renderRoom({ moderationDialogOpen, room: makeChatRoom({ viewer_role: "host" }) });

        // then
        expect(screen.getByTestId("moderation-dialog")).toBeInTheDocument();
    });

    it("shows the pinned messages panel only once it is opened", () => {
        // given
        const panelTab = "pins" as const;

        // when
        renderRoom({ panelTab });

        // then
        expect(screen.getByTestId("pins-panel")).toBeInTheDocument();
    });
});

describe("RoomPage notices", () => {
    it("says who is typing", () => {
        // given
        const typingNames = ["Beatrice", "Ange"];

        // when
        renderRoom({ typingNames });

        // then
        expect(screen.getByText(/are typing/)).toHaveTextContent("Beatrice and Ange are typing...");
    });

    it("shows a passing toast", () => {
        // given
        const toast = "1 member invited";

        // when
        renderRoom({ toast });

        // then
        expect(screen.getByText("1 member invited")).toBeInTheDocument();
    });

    it("keeps the lightbox shut until an image is opened", () => {
        // given
        const lightboxSrc = null;

        // when
        renderRoom({ lightboxSrc });

        // then
        expect(screen.queryByTestId("lightbox")).not.toBeInTheDocument();
    });

    it("shows the image the viewer opened", () => {
        // given
        const lightboxSrc = "/media/witch.png";

        // when
        renderRoom({ lightboxSrc });

        // then
        expect(screen.getByTestId("lightbox")).toHaveTextContent("/media/witch.png");
    });

    it("wires the composer to the open room", () => {
        // given
        const room = makeChatRoom({ id: "room-42" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByTestId("composer")).toHaveAttribute("data-room", "room-42");
        expect(screen.getByTestId("room-messages")).toBeInTheDocument();
    });
});
