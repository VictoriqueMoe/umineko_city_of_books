import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { RoomController } from "../../hooks/useRoomController";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { ChatRoom, ChatRoomMember, User, UserProfile } from "../../types/api";
import { RoomPage } from "./RoomPage";

const mocks = vi.hoisted(() => ({
    useRoomController: vi.fn(),
    useIsMobile: vi.fn(),
    forceMuteVoiceParticipant: vi.fn(),
}));

vi.mock("../../hooks/useRoomController", () => ({ useRoomController: mocks.useRoomController }));

vi.mock("../../hooks/useIsMobile", () => ({ useIsMobile: mocks.useIsMobile }));

vi.mock("../../api/endpoints", () => ({ forceMuteVoiceParticipant: mocks.forceMuteVoiceParticipant }));

vi.mock("../../components/chat/mobile/MobileRoomView", () => ({
    MobileRoomView: () => <div data-testid="mobile-room-view" />,
}));

vi.mock("../../components/chat/MessageList/RoomMessageList", () => ({
    RoomMessageList: () => <div data-testid="room-messages" />,
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

vi.mock("../../components/chat/PinnedMessagesPanel/PinnedMessagesPanel", () => ({
    PinnedMessagesPanel: (props: { isOpen: boolean }) => (props.isOpen ? <div data-testid="pinned-panel" /> : null),
}));

vi.mock("../../components/chat/MessageSearchPanel/MessageSearchPanel", () => ({
    MessageSearchPanel: () => <div data-testid="search-panel" />,
}));

vi.mock("../../components/chat/WatchParty/WatchPartyButton", () => ({
    WatchPartyButton: (props: { enabled: boolean }) => (
        <div data-testid="watch-party-button">{String(props.enabled)}</div>
    ),
}));

vi.mock("../../components/chat/WatchParty/WatchPartyModal", () => ({
    WatchPartyModal: (props: { isStarter: boolean }) => (
        <div data-testid="watch-party-modal">{String(props.isStarter)}</div>
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
    return { id: "member-1", username: "beatrice", display_name: "Beatrice", ...overrides };
}

function makeMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return {
        user: makeMemberUser(),
        role: "member",
        joined_at: "2026-01-01T00:00:00Z",
        nickname: "",
        member_avatar_url: "",
        nickname_locked: false,
        ...overrides,
    };
}

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "Tea Parlour",
        description: "",
        type: "group",
        is_public: true,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 2,
        hot_score: 0,
        members: [],
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

interface ControllerOptions {
    user?: UserProfile | null;
    loading?: boolean;
    room?: ChatRoom | null;
    roomId?: string | null;
    members?: ChatRoomMember[];
    memberGroups?: { label: string; members: ChatRoomMember[] }[];
    presenceMapMerged?: Record<string, "active" | "idle" | "">;
    onlineIds?: string[];
    currentMember?: ChatRoomMember | null;
    sidebarCollapsed?: boolean;
    descExpanded?: boolean;
    typingNames?: string[];
    voiceStatus?: string;
    voiceRoom?: object | null;
    voiceEnabled?: boolean;
    watchPartyEnabled?: boolean;
    activeSession?: object | null;
    invitedPartyMissing?: boolean;
    lightboxSrc?: string | null;
    toast?: string | null;
    busy?: string | null;
    joining?: boolean;
    openMemberMenu?: string | null;
    editProfileOpen?: boolean;
    inviteModalOpen?: boolean;
    moderationDialogOpen?: boolean;
    pinnedOpen?: boolean;
    searchOpen?: boolean;
    nicknameDialogTarget?: ChatRoomMember | null;
    nicknameDialogError?: string;
    nicknameDialogSaving?: boolean;
    timeoutDialogTarget?: ChatRoomMember | null;
    timeoutDialogError?: string;
    timeoutDialogSaving?: boolean;
}

function stubController(options: ControllerOptions = {}) {
    const handlers = {
        navigate: vi.fn(),
        setMobileView: vi.fn(),
        toggleSidebar: vi.fn(),
        toggleDescExpanded: vi.fn(),
        setReplyingTo: vi.fn(),
        setLightboxSrc: vi.fn(),
        setToast: vi.fn(),
        sendWSMessage: vi.fn(),
        setPinnedOpen: vi.fn(),
        setSearchOpen: vi.fn(),
        setEditProfileOpen: vi.fn(),
        setInviteModalOpen: vi.fn(),
        setModerationDialogOpen: vi.fn(),
        setOpenMemberMenu: vi.fn(),
        setMembers: vi.fn(),
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

    const controller = {
        user: options.user === undefined ? viewer : options.user,
        navigate: handlers.navigate,
        loading: options.loading ?? false,
        room: options.room === undefined ? makeRoom() : options.room,
        roomId: options.roomId === undefined ? "room-1" : options.roomId,
        members,
        memberGroups: options.memberGroups ?? [{ label: "Members", members }],
        presenceMapMerged: options.presenceMapMerged ?? {},
        memberOnlineWeight: (id: string) => (onlineIds.has(id) ? 0 : 1),
        currentMember: options.currentMember ?? null,
        mobileView: "chat",
        setMobileView: handlers.setMobileView,
        sidebarCollapsed: options.sidebarCollapsed ?? false,
        toggleSidebar: handlers.toggleSidebar,
        descExpanded: options.descExpanded ?? false,
        toggleDescExpanded: handlers.toggleDescExpanded,
        typingNames: options.typingNames ?? [],
        voice: {
            status: options.voiceStatus ?? "idle",
            room: options.voiceRoom ?? null,
            join: vi.fn(),
            leave: vi.fn(),
            presenceCount: 0,
        },
        voiceEnabled: options.voiceEnabled ?? true,
        watchParty: {
            enabled: options.watchPartyEnabled ?? true,
            screenShareEnabled: true,
            sessions: [],
            openSessionId: null,
            activeSession: options.activeSession ?? null,
            start: handlers.watchPartyStart,
            join: vi.fn(),
            openExisting: vi.fn(),
            close: handlers.watchPartyClose,
            leave: vi.fn(),
            end: vi.fn(),
            transferControl: vi.fn(),
            kick: vi.fn(),
            identify: vi.fn(),
        },
        invitedPartyMissing: options.invitedPartyMissing ?? false,
        replyingTo: null,
        setReplyingTo: handlers.setReplyingTo,
        viewerTimeoutUntil: undefined,
        lightboxSrc: options.lightboxSrc ?? null,
        setLightboxSrc: handlers.setLightboxSrc,
        toast: options.toast ?? null,
        setToast: handlers.setToast,
        busy: options.busy ?? null,
        joining: options.joining ?? false,
        sendWSMessage: handlers.sendWSMessage,
        pinnedOpen: options.pinnedOpen ?? false,
        setPinnedOpen: handlers.setPinnedOpen,
        searchOpen: options.searchOpen ?? false,
        setSearchOpen: handlers.setSearchOpen,
        pinnedRefreshKey: 0,
        editProfileOpen: options.editProfileOpen ?? false,
        setEditProfileOpen: handlers.setEditProfileOpen,
        inviteModalOpen: options.inviteModalOpen ?? false,
        setInviteModalOpen: handlers.setInviteModalOpen,
        moderationDialogOpen: options.moderationDialogOpen ?? false,
        setModerationDialogOpen: handlers.setModerationDialogOpen,
        openMemberMenu: options.openMemberMenu ?? null,
        setOpenMemberMenu: handlers.setOpenMemberMenu,
        setMembers: handlers.setMembers,
        nicknameDialogTarget: options.nicknameDialogTarget ?? null,
        setNicknameDialogTarget: handlers.setNicknameDialogTarget,
        nicknameDialogValue: "",
        setNicknameDialogValue: handlers.setNicknameDialogValue,
        nicknameDialogError: options.nicknameDialogError ?? "",
        nicknameDialogSaving: options.nicknameDialogSaving ?? false,
        timeoutDialogTarget: options.timeoutDialogTarget ?? null,
        setTimeoutDialogTarget: handlers.setTimeoutDialogTarget,
        timeoutDialogAmount: "10",
        setTimeoutDialogAmount: handlers.setTimeoutDialogAmount,
        timeoutDialogUnit: "seconds",
        setTimeoutDialogUnit: handlers.setTimeoutDialogUnit,
        timeoutDialogError: options.timeoutDialogError ?? "",
        timeoutDialogSaving: options.timeoutDialogSaving ?? false,
        formatTimeoutUntil: (value?: string) => `until ${value ?? "never"}`,
        openNicknameDialog: handlers.openNicknameDialog,
        openTimeoutDialog: handlers.openTimeoutDialog,
        handleSentMessage: handlers.handleSentMessage,
        handleJoin: handlers.handleJoin,
        handleModSetNickname: handlers.handleModSetNickname,
        handleModUnlockNickname: handlers.handleModUnlockNickname,
        handleSetTimeout: handlers.handleSetTimeout,
        handleClearTimeout: handlers.handleClearTimeout,
        handleKick: handlers.handleKick,
        handleBan: handlers.handleBan,
        handleToggleMute: handlers.handleToggleMute,
        handleLeave: handlers.handleLeave,
        handleDelete: handlers.handleDelete,
        handleJumpToMessage: handlers.handleJumpToMessage,
        handleEditLast: handlers.handleEditLast,
    };

    mocks.useRoomController.mockReturnValue(controller as unknown as RoomController);

    return handlers;
}

function renderRoom(options: ControllerOptions = {}) {
    const handlers = stubController(options);
    const result = renderWithProviders(<RoomPage />, { user: options.user === undefined ? viewer : options.user });

    return { ...result, ...handlers };
}

beforeEach(() => {
    mocks.useIsMobile.mockReturnValue(false);
    mocks.forceMuteVoiceParticipant.mockResolvedValue(undefined);
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
        const room = makeRoom({ name: "Rokkenjima", member_count: 7, is_public: true });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        expect(screen.getByText(/7 members/)).toBeInTheDocument();
        expect(screen.getByText(/public/)).toBeInTheDocument();
    });

    it("marks a private room as private", () => {
        // given
        const room = makeRoom({ is_public: false });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText(/private/)).toBeInTheDocument();
    });

    it("badges a staff room and a roleplay room", () => {
        // given
        const room = makeRoom({ is_system: true, is_rp: true });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText("Staff")).toBeInTheDocument();
        expect(screen.getByText("RP")).toBeInTheDocument();
    });

    it("opens the message search when asked", async () => {
        // given
        const user = userEvent.setup();
        const { setSearchOpen } = renderRoom();

        // when
        await user.click(screen.getByRole("button", { name: "Search messages" }));

        // then
        expect(setSearchOpen).toHaveBeenCalledWith(true);
    });

    it("opens the pinned messages when asked", async () => {
        // given
        const user = userEvent.setup();
        const { setPinnedOpen } = renderRoom();

        // when
        await user.click(screen.getByRole("button", { name: "Pinned messages" }));

        // then
        expect(setPinnedOpen).toHaveBeenCalledWith(true);
    });

    it("keeps the search panel closed until it is opened", () => {
        // given
        const searchOpen = false;

        // when
        renderRoom({ searchOpen });

        // then
        expect(screen.queryByTestId("search-panel")).not.toBeInTheDocument();
    });

    it("shows the search panel once it is open", () => {
        // given
        const searchOpen = true;

        // when
        renderRoom({ searchOpen });

        // then
        expect(screen.getByTestId("search-panel")).toBeInTheDocument();
    });
});

describe("RoomPage room info", () => {
    it("leaves the info panel out when there is nothing to say", () => {
        // given
        const room = makeRoom({ description: "", tags: [] });

        // when
        renderRoom({ room });

        // then
        expect(screen.queryByRole("button", { name: /Show info/ })).not.toBeInTheDocument();
    });

    it("offers the info panel when the room has a description", () => {
        // given
        const room = makeRoom({ description: "Where the witches take tea" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "Show info ▼" })).toBeInTheDocument();
        expect(screen.getByText("Where the witches take tea")).toBeInTheDocument();
    });

    it("lists the room's tags", () => {
        // given
        const room = makeRoom({ tags: ["horror", "spoilers"] });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByText("#horror")).toBeInTheDocument();
        expect(screen.getByText("#spoilers")).toBeInTheDocument();
    });

    it("collapses the info panel when it is already expanded", async () => {
        // given
        const user = userEvent.setup();
        const { toggleDescExpanded } = renderRoom({ room: makeRoom({ description: "Tea" }), descExpanded: true });

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
        renderRoom({ members, room: makeRoom({ viewer_role: "member" }) });

        // then
        expect(screen.queryByRole("button", { name: "Moderator actions" })).not.toBeInTheDocument();
    });

    it("gives the host moderator actions over an ordinary member", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoom({ members, room: makeRoom({ viewer_role: "host" }) });

        // then
        expect(screen.getByRole("button", { name: "Moderator actions" })).toBeInTheDocument();
    });

    it("opens the moderator menu when it is clicked", async () => {
        // given
        const user = userEvent.setup();
        const { setOpenMemberMenu } = renderRoom({
            members: [makeMember()],
            room: makeRoom({ viewer_role: "host" }),
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
        renderRoom({ members, room: makeRoom({ viewer_role: "host" }), openMemberMenu: "member-1" });

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
            room: makeRoom({ viewer_role: "member" }),
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
            room: makeRoom({ viewer_role: "host" }),
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
            room: makeRoom({ viewer_role: "host" }),
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
            room: makeRoom({ viewer_role: "host" }),
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
            room: makeRoom({ viewer_role: "host" }),
            openMemberMenu: "member-1",
        });

        // when
        await user.click(screen.getByRole("button", { name: "Remove timeout" }));

        // then
        expect(handleClearTimeout).toHaveBeenCalledWith("member-1");
    });

    it("leaves a system room unmoderated", () => {
        // given
        const room = makeRoom({ is_system: true, viewer_role: "host" });

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
        const room = makeRoom({ viewer_muted: true });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "Unmute notifications" })).toBeInTheDocument();
    });

    it("gives the host moderation and deletion", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "Moderation" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete Room" })).toBeInTheDocument();
    });

    it("offers an ordinary member the door instead", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "Leave Room" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete Room" })).not.toBeInTheDocument();
    });

    it("keeps deletion and leaving away from a system room", () => {
        // given
        const room = makeRoom({ is_system: true, viewer_role: "host" });

        // when
        renderRoom({ room });

        // then
        expect(screen.queryByRole("button", { name: "Delete Room" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Leave Room" })).not.toBeInTheDocument();
    });

    it("shows the invite button to the host of an ordinary room", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByRole("button", { name: "+ Invite" })).toBeInTheDocument();
    });

    it("hides the invite button from a plain member", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        renderRoom({ room });

        // then
        expect(screen.queryByRole("button", { name: "+ Invite" })).not.toBeInTheDocument();
    });

    it("shows the invite button to site staff who do not host the room", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: "moderator" });
        const room = makeRoom({ viewer_role: "member" });

        // when
        renderRoom({ user, room });

        // then
        expect(screen.getByRole("button", { name: "+ Invite" })).toBeInTheDocument();
    });

    it("hides the invite button from site staff in a system room", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: "moderator" });
        const room = makeRoom({ viewer_role: "member", is_system: true });

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
        const { navigate } = renderRoom();

        // when
        await user.click(screen.getByRole("button", { name: "Back to rooms" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/rooms");
    });

    it("says it is deleting while the room is being removed", () => {
        // given
        const busy = "delete";

        // when
        renderRoom({ room: makeRoom({ viewer_role: "host" }), busy });

        // then
        expect(screen.getByRole("button", { name: "Deleting..." })).toBeDisabled();
    });
});

describe("RoomPage voice and watch party", () => {
    it("keeps the voice bar away until the call is connected", () => {
        // given
        const voiceStatus = "connecting";

        // when
        renderRoom({ voiceStatus, voiceRoom: {} });

        // then
        expect(screen.queryByRole("button", { name: "voice bar" })).not.toBeInTheDocument();
    });

    it("shows the voice bar once the call is connected", async () => {
        // given
        const voiceStatus = "connected";

        // when
        renderRoom({ voiceStatus, voiceRoom: {}, room: makeRoom({ viewer_role: "host" }) });

        // then
        expect(await screen.findByRole("button", { name: "voice bar" })).toHaveAttribute("data-moderator", "true");
    });

    it("force mutes a voice participant in this room", async () => {
        // given
        const pointer = userEvent.setup();
        renderRoom({ voiceStatus: "connected", voiceRoom: {}, room: makeRoom({ viewer_role: "host" }) });
        const bar = await screen.findByRole("button", { name: "voice bar" });

        // when
        await pointer.click(bar);

        // then
        expect(mocks.forceMuteVoiceParticipant).toHaveBeenCalledWith("room-1", "u9", true);
    });

    it("gives an ordinary room the voice and watch party buttons", () => {
        // given
        const room = makeRoom({ is_system: false });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByTestId("voice-button")).toBeInTheDocument();
        expect(screen.getByTestId("watch-party-button")).toBeInTheDocument();
    });

    it("keeps the voice and watch party buttons out of a system room", () => {
        // given
        const room = makeRoom({ is_system: true });

        // when
        renderRoom({ room });

        // then
        expect(screen.queryByTestId("voice-button")).not.toBeInTheDocument();
        expect(screen.queryByTestId("watch-party-button")).not.toBeInTheDocument();
    });

    it("opens the watch party window for an active session", async () => {
        // given
        const activeSession = { session: { started_by: viewer.id } };

        // when
        renderRoom({ activeSession });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveTextContent("true");
    });

    it("knows the viewer did not start somebody else's watch party", async () => {
        // given
        const activeSession = { session: { started_by: "someone-else" } };

        // when
        renderRoom({ activeSession });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveTextContent("false");
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
        renderRoom({ inviteModalOpen, room: makeRoom({ viewer_role: "host" }) });

        // then
        expect(screen.getByTestId("invite-modal")).toBeInTheDocument();
    });

    it("shows the moderation dialog only once it is opened", () => {
        // given
        const moderationDialogOpen = true;

        // when
        renderRoom({ moderationDialogOpen, room: makeRoom({ viewer_role: "host" }) });

        // then
        expect(screen.getByTestId("moderation-dialog")).toBeInTheDocument();
    });

    it("shows the pinned messages panel only once it is opened", () => {
        // given
        const pinnedOpen = true;

        // when
        renderRoom({ pinnedOpen });

        // then
        expect(screen.getByTestId("pinned-panel")).toBeInTheDocument();
    });
});

describe("RoomPage notices", () => {
    it("says who is typing", () => {
        // given
        const typingNames = ["Beatrice", "Ange"];

        // when
        renderRoom({ typingNames });

        // then
        expect(screen.getByText("Beatrice and Ange are typing...")).toBeInTheDocument();
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
        const room = makeRoom({ id: "room-42" });

        // when
        renderRoom({ room });

        // then
        expect(screen.getByTestId("composer")).toHaveAttribute("data-room", "room-42");
        expect(screen.getByTestId("room-messages")).toBeInTheDocument();
    });
});
