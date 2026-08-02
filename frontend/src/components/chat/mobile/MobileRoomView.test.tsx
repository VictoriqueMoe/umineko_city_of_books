import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { RoomController } from "../../../hooks/useRoomController";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatRoom, ChatRoomMember, User } from "../../../types/api";
import { MobileRoomView } from "./MobileRoomView";

const mocks = vi.hoisted(() => ({ forceMuteVoiceParticipant: vi.fn(() => Promise.resolve()) }));

vi.mock("../../../api/endpoints", async importOriginal => {
    const actual = await importOriginal<typeof import("../../../api/endpoints")>();

    return { ...actual, forceMuteVoiceParticipant: mocks.forceMuteVoiceParticipant };
});

vi.mock("../MessageList/RoomMessageList", () => ({
    RoomMessageList: () => <div data-testid="room-messages">messages</div>,
}));

vi.mock("../MessageSearchPanel/MessageSearchPanel", () => ({
    MessageSearchPanel: ({ isOpen }: { isOpen: boolean }) =>
        isOpen ? <div data-testid="search-panel">search</div> : null,
}));

vi.mock("../PinnedMessagesPanel/PinnedMessagesPanel", () => ({
    PinnedMessagesPanel: ({ isOpen, canUnpin }: { isOpen: boolean; canUnpin: boolean }) =>
        isOpen ? <div data-testid="pinned-panel" data-can-unpin={String(canUnpin)} /> : null,
}));

vi.mock("../EditRoomProfileDialog/EditRoomProfileDialog", () => ({
    EditRoomProfileDialog: ({ isOpen, onSaved }: { isOpen: boolean; onSaved: (member: ChatRoomMember) => void }) =>
        isOpen ? (
            <div data-testid="edit-profile">
                <button
                    type="button"
                    onClick={() =>
                        onSaved({
                            user: { id: "u1", username: "beatrice", display_name: "Beatrice" },
                            role: "member",
                            joined_at: "2026-07-01T00:00:00Z",
                            nickname: "Golden Witch",
                            member_avatar_url: "",
                            nickname_locked: false,
                        })
                    }
                >
                    save room profile
                </button>
            </div>
        ) : null,
}));

vi.mock("../RoomModerationDialog/RoomModerationDialog", () => ({
    RoomModerationDialog: ({ isOpen }: { isOpen: boolean }) =>
        isOpen ? <div data-testid="moderation-dialog" /> : null,
}));

vi.mock("../InviteMembersModal/InviteMembersModal", () => ({
    InviteMembersModal: ({
        isOpen,
        onInvited,
    }: {
        isOpen: boolean;
        onInvited: (result: { invited_count: number; skipped_count: number }) => void;
    }) =>
        isOpen ? (
            <div data-testid="invite-modal">
                <button type="button" onClick={() => onInvited({ invited_count: 1, skipped_count: 0 })}>
                    invite one
                </button>
                <button type="button" onClick={() => onInvited({ invited_count: 3, skipped_count: 0 })}>
                    invite three
                </button>
                <button type="button" onClick={() => onInvited({ invited_count: 0, skipped_count: 2 })}>
                    invite nobody
                </button>
            </div>
        ) : null,
}));

vi.mock("../WatchParty/WatchPartyButton", () => ({
    WatchPartyButton: ({ enabled }: { enabled: boolean }) => (
        <div data-testid="watch-party-button" data-enabled={String(enabled)} />
    ),
}));

vi.mock("../WatchParty/WatchPartyModal", () => ({
    WatchPartyModal: ({ isStarter }: { isStarter: boolean }) => (
        <div data-testid="watch-party-modal" data-is-starter={String(isStarter)} />
    ),
}));

vi.mock("../Voice/VoiceBar", () => ({
    VoiceBar: ({
        canModerate,
        onForceMute,
    }: {
        canModerate?: boolean;
        onForceMute?: (identity: string, muted: boolean) => void;
    }) => (
        <div data-testid="voice-bar" data-can-moderate={String(canModerate)}>
            <button type="button" onClick={() => onForceMute?.("battler", true)}>
                server mute battler
            </button>
        </div>
    ),
}));

vi.mock("../ChatComposer/ChatComposer", () => ({
    ChatComposer: ({
        roomId,
        timeoutUntil,
        mentionPool,
        extraActions,
    }: {
        roomId: string | null;
        timeoutUntil?: string;
        mentionPool?: User[];
        extraActions?: React.ReactNode;
    }) => (
        <div
            data-testid="composer"
            data-room-id={roomId ?? ""}
            data-timeout-until={timeoutUntil ?? ""}
            data-mention-pool={(mentionPool ?? []).map(u => u.username).join(",")}
        >
            {extraActions}
        </div>
    ),
}));

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return {
        user: { id: "u2", username: "battler", display_name: "Battler" },
        role: "member",
        joined_at: "2026-07-01T00:00:00Z",
        nickname: "",
        member_avatar_url: "",
        nickname_locked: false,
        ...overrides,
    };
}

const selfMember = makeMember({ user: { id: "u1", username: "beatrice", display_name: "Beatrice" } });

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "Rokkenjima",
        description: "",
        type: "group",
        is_public: true,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_role: "member",
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 2,
        hot_score: 0,
        members: [],
        created_at: "2026-07-01T00:00:00Z",
        ...overrides,
    };
}

function makeVoice(overrides: Record<string, unknown> = {}): RoomController["voice"] {
    return {
        status: "idle",
        room: null,
        participantIds: [],
        presenceCount: 0,
        join: vi.fn(),
        leave: vi.fn(),
        ...overrides,
    } as unknown as RoomController["voice"];
}

function makeWatchParty(overrides: Record<string, unknown> = {}): RoomController["watchParty"] {
    const base = {
        enabled: true,
        screenShareEnabled: false,
        loaded: true,
        sessions: [],
        activeSession: null,
        openSessionId: null,
        error: "",
        refresh: vi.fn(),
        start: vi.fn(),
        join: vi.fn(),
        leave: vi.fn(),
        end: vi.fn(),
        transferControl: vi.fn(),
        kick: vi.fn(),
        identify: vi.fn(),
        sendMessage: vi.fn(),
        openExisting: vi.fn(),
        close: vi.fn(),
        clearError: vi.fn(),
        ...overrides,
    };

    return base as unknown as RoomController["watchParty"];
}

function makeController(overrides: Partial<RoomController> = {}): RoomController {
    const base = {
        user: viewer,
        navigate: vi.fn(),
        room: makeRoom(),
        roomId: "room-1",
        members: [selfMember, makeMember()],
        memberGroups: [{ label: "Members", members: [selfMember, makeMember()] }],
        presenceMapMerged: {},
        currentMember: selfMember,
        mobileView: "chat",
        setMobileView: vi.fn(),
        scrollToBottom: vi.fn(),
        typingNames: [],
        voice: makeVoice(),
        voiceEnabled: true,
        watchParty: makeWatchParty(),
        invitedPartyMissing: false,
        replyingTo: null,
        setReplyingTo: vi.fn(),
        viewerTimeoutUntil: undefined,
        lightboxSrc: null,
        setLightboxSrc: vi.fn(),
        toast: "",
        setToast: vi.fn(),
        busy: "",
        sendWSMessage: vi.fn(),
        pinnedOpen: false,
        setPinnedOpen: vi.fn(),
        searchOpen: false,
        setSearchOpen: vi.fn(),
        pinnedRefreshKey: 0,
        editProfileOpen: false,
        setEditProfileOpen: vi.fn(),
        inviteModalOpen: false,
        setInviteModalOpen: vi.fn(),
        moderationDialogOpen: false,
        setModerationDialogOpen: vi.fn(),
        openMemberMenu: null,
        setOpenMemberMenu: vi.fn(),
        setMembers: vi.fn(),
        nicknameDialogTarget: null,
        setNicknameDialogTarget: vi.fn(),
        nicknameDialogValue: "",
        setNicknameDialogValue: vi.fn(),
        nicknameDialogError: "",
        nicknameDialogSaving: false,
        timeoutDialogTarget: null,
        setTimeoutDialogTarget: vi.fn(),
        timeoutDialogAmount: "10",
        setTimeoutDialogAmount: vi.fn(),
        timeoutDialogUnit: "seconds",
        setTimeoutDialogUnit: vi.fn(),
        timeoutDialogError: "",
        timeoutDialogSaving: false,
        openNicknameDialog: vi.fn(),
        openTimeoutDialog: vi.fn(),
        handleSentMessage: vi.fn(),
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
    };

    return { ...base, ...overrides } as unknown as RoomController;
}

function renderView(overrides: Partial<RoomController> = {}) {
    const controller = makeController(overrides);
    const result = renderWithProviders(<MobileRoomView controller={controller} />);

    return { ...result, controller };
}

function membersView(overrides: Partial<RoomController> = {}) {
    return renderView({ mobileView: "members", ...overrides });
}

const staffViewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice", role: "admin" });

beforeEach(() => {
    mocks.forceMuteVoiceParticipant.mockResolvedValue(undefined);
});

describe("MobileRoomView chat view", () => {
    it("renders nothing until the viewer is signed in", () => {
        // given
        const user = null;

        // when
        const { container } = renderView({ user });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("renders nothing until the room has loaded", () => {
        // given
        const room = undefined;

        // when
        const { container } = renderView({ room });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("names the room and says how many people and how open it is", () => {
        // given
        const room = makeRoom({ name: "Rokkenjima", member_count: 7, is_public: true });

        // when
        renderView({ room });

        // then
        expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        expect(screen.getByText(/7 members/)).toHaveTextContent("public");
    });

    it("says when a room is private", () => {
        // given
        const room = makeRoom({ is_public: false });

        // when
        renderView({ room });

        // then
        expect(screen.getByText(/members/)).toHaveTextContent("private");
    });

    it("counts the members it has when the server sent no count", () => {
        // given
        const room = makeRoom({
            member_count: undefined as unknown as number,
            members: [
                { id: "u1", username: "beatrice", display_name: "Beatrice" },
                { id: "u2", username: "battler", display_name: "Battler" },
                { id: "u3", username: "ronove", display_name: "Ronove" },
            ],
        });

        // when
        renderView({ room });

        // then
        expect(screen.getByText(/members/)).toHaveTextContent("3 members");
    });

    it("badges a staff room and a roleplay room", () => {
        // given
        const room = makeRoom({ is_system: true, is_rp: true });

        // when
        renderView({ room });

        // then
        expect(screen.getByText("Staff")).toBeInTheDocument();
        expect(screen.getByText("RP")).toBeInTheDocument();
    });

    it("goes back to the room directory", async () => {
        // given
        const navigate = vi.fn();
        const user = userEvent.setup();
        renderView({ navigate });

        // when
        await user.click(screen.getByLabelText("Back to rooms"));

        // then
        expect(navigate).toHaveBeenCalledWith("/rooms");
    });

    it("opens the search, the pins and the member list from the top bar", async () => {
        // given
        const setSearchOpen = vi.fn();
        const setPinnedOpen = vi.fn();
        const setMobileView = vi.fn();
        const user = userEvent.setup();
        renderView({ setSearchOpen, setPinnedOpen, setMobileView });

        // when
        await user.click(screen.getByLabelText("Search messages"));
        await user.click(screen.getByLabelText("Pinned messages"));
        await user.click(screen.getByLabelText("Members"));

        // then
        expect(setSearchOpen).toHaveBeenCalledWith(true);
        expect(setPinnedOpen).toHaveBeenCalledWith(true);
        expect(setMobileView).toHaveBeenCalledWith("members");
    });

    it("keeps the search panel out of the way until it is opened", () => {
        // given
        const searchOpen = false;

        // when
        renderView({ searchOpen });

        // then
        expect(screen.queryByTestId("search-panel")).not.toBeInTheDocument();
    });

    it("lets only a moderator unpin from the pinned panel", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        renderView({ room, pinnedOpen: true });

        // then
        expect(screen.getByTestId("pinned-panel")).toHaveAttribute("data-can-unpin", "true");
    });

    it("denies unpinning to an ordinary member", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        renderView({ room, pinnedOpen: true });

        // then
        expect(screen.getByTestId("pinned-panel")).toHaveAttribute("data-can-unpin", "false");
    });

    it("keeps the voice bar away while nobody has joined the call", () => {
        // given
        const voice = makeVoice({ status: "idle", room: null });

        // when
        renderView({ voice });

        // then
        expect(screen.queryByTestId("voice-bar")).not.toBeInTheDocument();
    });

    it("shows the voice bar once the viewer is connected to the call", async () => {
        // given
        const voice = makeVoice({ status: "connected", room: { name: "voice" } });

        // when
        renderView({ voice });

        // then
        expect(await screen.findByTestId("voice-bar")).toHaveAttribute("data-can-moderate", "false");
    });

    it("lets the host moderate the call", async () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        renderView({ room, voice: makeVoice({ status: "connected", room: { name: "voice" } }) });

        // then
        expect(await screen.findByTestId("voice-bar")).toHaveAttribute("data-can-moderate", "true");
    });

    it("sends a server mute to the room the call belongs to", async () => {
        // given
        const user = userEvent.setup();
        renderView({ voice: makeVoice({ status: "connected", room: { name: "voice" } }) });
        await screen.findByTestId("voice-bar");

        // when
        await user.click(screen.getByRole("button", { name: "server mute battler" }));

        // then
        expect(mocks.forceMuteVoiceParticipant).toHaveBeenCalledWith("room-1", "battler", true);
    });

    it("gives the composer the room, the mention pool and the viewer's timeout", () => {
        // given
        const viewerTimeoutUntil = "2026-08-02T12:00:00Z";

        // when
        renderView({ viewerTimeoutUntil });

        // then
        const composer = screen.getByTestId("composer");
        expect(composer).toHaveAttribute("data-room-id", "room-1");
        expect(composer).toHaveAttribute("data-timeout-until", "2026-08-02T12:00:00Z");
        expect(composer).toHaveAttribute("data-mention-pool", "beatrice,battler");
    });

    it("offers the voice and watch party controls in an ordinary room", () => {
        // given
        const room = makeRoom({ is_system: false });

        // when
        renderView({ room });

        // then
        expect(screen.getByTitle("Join voice")).toBeInTheDocument();
        expect(screen.getByTestId("watch-party-button")).toBeInTheDocument();
    });

    it("takes the voice and watch party controls away in a staff room", () => {
        // given
        const room = makeRoom({ is_system: true });

        // when
        renderView({ room });

        // then
        expect(screen.queryByTitle("Join voice")).not.toBeInTheDocument();
        expect(screen.queryByTestId("watch-party-button")).not.toBeInTheDocument();
    });

    it("shows the message list and who is typing", () => {
        // given
        const typingNames = ["Battler"];

        // when
        renderView({ typingNames });

        // then
        expect(screen.getByTestId("room-messages")).toBeInTheDocument();
        expect(screen.getByText("Battler is typing...")).toBeInTheDocument();
    });

    it("tells the viewer when the watch party they were invited to has ended", () => {
        // given
        const invitedPartyMissing = true;

        // when
        renderView({ invitedPartyMissing });

        // then
        expect(screen.getByText("That watch party has ended.")).toBeInTheDocument();
    });

    it("shows a toast the controller raised", () => {
        // given
        const toast = "1 member invited";

        // when
        renderView({ toast });

        // then
        expect(screen.getByText("1 member invited")).toBeInTheDocument();
    });

    it("opens the watch party for the person who started it", async () => {
        // given
        const watchParty = makeWatchParty({ activeSession: { session: { started_by: "u1" } } });

        // when
        renderView({ watchParty });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-is-starter", "true");
    });

    it("opens the watch party as a guest for everybody else", async () => {
        // given
        const watchParty = makeWatchParty({ activeSession: { session: { started_by: "u9" } } });

        // when
        renderView({ watchParty });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-is-starter", "false");
    });

    it("reports a single invitation in the singular", async () => {
        // given
        const setToast = vi.fn();
        const user = userEvent.setup();
        renderView({ setToast, inviteModalOpen: true });

        // when
        await user.click(screen.getByRole("button", { name: "invite one" }));

        // then
        expect(setToast).toHaveBeenCalledWith("1 member invited");
    });

    it("reports several invitations in the plural", async () => {
        // given
        const setToast = vi.fn();
        const user = userEvent.setup();
        renderView({ setToast, inviteModalOpen: true });

        // when
        await user.click(screen.getByRole("button", { name: "invite three" }));

        // then
        expect(setToast).toHaveBeenCalledWith("3 members invited");
    });

    it("explains when nobody could be invited", async () => {
        // given
        const setToast = vi.fn();
        const user = userEvent.setup();
        renderView({ setToast, inviteModalOpen: true });

        // when
        await user.click(screen.getByRole("button", { name: "invite nobody" }));

        // then
        expect(setToast).toHaveBeenCalledWith("No one invited (all were already members or blocked)");
    });

    it("swaps in the saved room profile without disturbing the other members", async () => {
        // given
        const setMembers = vi.fn();
        const user = userEvent.setup();
        renderView({ setMembers, editProfileOpen: true });

        // when
        await user.click(screen.getByRole("button", { name: "save room profile" }));

        // then
        const updater = setMembers.mock.calls[0][0] as (prev: ChatRoomMember[]) => ChatRoomMember[];
        const next = updater([selfMember, makeMember()]);
        expect(next[0].nickname).toBe("Golden Witch");
        expect(next[1].nickname).toBe("");
    });

    it("shows the lightbox for the image the viewer opened", () => {
        // given
        const lightboxSrc = "https://cdn.example/photo.png";

        // when
        renderView({ lightboxSrc });

        // then
        expect(screen.getByRole("dialog")).toBeInTheDocument();
    });
});

describe("MobileRoomView members view", () => {
    it("heads the list with how many people are in the room", () => {
        // given
        const members = [
            selfMember,
            makeMember(),
            makeMember({ user: { id: "u3", username: "ronove", display_name: "Ronove" } }),
        ];

        // when
        membersView({ members, memberGroups: [{ label: "Online", members }] });

        // then
        expect(screen.getByText("Members")).toBeInTheDocument();
        expect(screen.getByText("3 members")).toBeInTheDocument();
    });

    it("goes back to the chat from the member list", async () => {
        // given
        const setMobileView = vi.fn();
        const user = userEvent.setup();
        membersView({ setMobileView });

        // when
        await user.click(screen.getByLabelText("Back to chat"));

        // then
        expect(setMobileView).toHaveBeenCalledWith("chat");
    });

    it("groups the members under the labels the controller gave", () => {
        // given
        const memberGroups = [
            { label: "In Voice", members: [makeMember()] },
            { label: "Offline", members: [selfMember] },
        ];

        // when
        membersView({ memberGroups });

        // then
        expect(screen.getByText("In Voice")).toBeInTheDocument();
        expect(screen.getByText("Offline")).toBeInTheDocument();
    });

    it("prefers a member's room nickname over their profile name", () => {
        // given
        const memberGroups = [{ label: "Members", members: [makeMember({ nickname: "Endless Sorcerer" })] }];

        // when
        membersView({ memberGroups });

        // then
        expect(screen.getByText("Endless Sorcerer")).toBeInTheDocument();
        expect(screen.queryByText("Battler")).not.toBeInTheDocument();
    });

    it("badges the host of the room", () => {
        // given
        const memberGroups = [{ label: "Members", members: [makeMember({ role: "host" })] }];

        // when
        membersView({ memberGroups });

        // then
        expect(screen.getByText("Host")).toBeInTheDocument();
    });

    it("offers the invite control to the host of an ordinary room only", () => {
        // given
        const room = makeRoom({ viewer_role: "host", is_system: false });

        // when
        membersView({ room });

        // then
        expect(screen.getByLabelText("Invite members")).toBeInTheDocument();
    });

    it("withholds the invite control from an ordinary member", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        membersView({ room });

        // then
        expect(screen.queryByLabelText("Invite members")).not.toBeInTheDocument();
    });

    it("withholds the invite control in a staff room", () => {
        // given
        const room = makeRoom({ viewer_role: "host", is_system: true });

        // when
        membersView({ room });

        // then
        expect(screen.queryByLabelText("Invite members")).not.toBeInTheDocument();
    });

    it("puts the room profile control on the viewer's own row alone", () => {
        // given
        const memberGroups = [{ label: "Members", members: [selfMember, makeMember()] }];

        // when
        membersView({ memberGroups });

        // then
        expect(screen.getAllByLabelText("Edit profile in this room")).toHaveLength(1);
    });

    it("opens the room profile editor from the viewer's own row", async () => {
        // given
        const setEditProfileOpen = vi.fn();
        const user = userEvent.setup();
        membersView({ setEditProfileOpen, memberGroups: [{ label: "Members", members: [selfMember] }] });

        // when
        await user.click(screen.getByLabelText("Edit profile in this room"));

        // then
        expect(setEditProfileOpen).toHaveBeenCalledWith(true);
    });

    it("hides the moderator actions from a member who cannot act on anyone", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        membersView({ room, memberGroups: [{ label: "Members", members: [makeMember()] }] });

        // then
        expect(screen.queryByLabelText("Moderator actions")).not.toBeInTheDocument();
    });

    it("offers the moderator actions to the host", async () => {
        // given
        const setOpenMemberMenu = vi.fn();
        const room = makeRoom({ viewer_role: "host" });
        const user = userEvent.setup();
        membersView({ room, setOpenMemberMenu, memberGroups: [{ label: "Members", members: [makeMember()] }] });

        // when
        await user.click(screen.getByLabelText("Moderator actions"));

        // then
        expect(setOpenMemberMenu).toHaveBeenCalledTimes(1);
    });

    it("offers a host kicking and banning but not renaming", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        membersView({
            room,
            openMemberMenu: "u2",
            memberGroups: [{ label: "Members", members: [makeMember()] }],
        });

        // then
        expect(screen.getByRole("button", { name: "Kick member" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Ban from room" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Change nickname" })).not.toBeInTheDocument();
    });

    it("offers site staff the nickname controls as well", () => {
        // given
        const user = staffViewer;

        // when
        membersView({
            user,
            openMemberMenu: "u2",
            memberGroups: [{ label: "Members", members: [makeMember({ nickname_locked: true })] }],
        });

        // then
        expect(screen.getByRole("button", { name: "Change nickname" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Reset/unlock nickname" })).toBeInTheDocument();
    });

    it("hides the unlock control while the nickname is not locked", () => {
        // given
        const user = staffViewer;

        // when
        membersView({
            user,
            openMemberMenu: "u2",
            memberGroups: [{ label: "Members", members: [makeMember({ nickname_locked: false })] }],
        });

        // then
        expect(screen.queryByRole("button", { name: "Reset/unlock nickname" })).not.toBeInTheDocument();
    });

    it("kicks a member and closes the menu behind it", async () => {
        // given
        const handleKick = vi.fn();
        const setOpenMemberMenu = vi.fn();
        const room = makeRoom({ viewer_role: "host" });
        const user = userEvent.setup();
        membersView({
            room,
            handleKick,
            setOpenMemberMenu,
            openMemberMenu: "u2",
            memberGroups: [{ label: "Members", members: [makeMember()] }],
        });

        // when
        await user.click(screen.getByRole("button", { name: "Kick member" }));

        // then
        expect(setOpenMemberMenu).toHaveBeenCalledWith(null);
        expect(handleKick).toHaveBeenCalledWith("u2");
    });

    it("bans a member from the room", async () => {
        // given
        const handleBan = vi.fn();
        const room = makeRoom({ viewer_role: "host" });
        const user = userEvent.setup();
        membersView({
            room,
            handleBan,
            openMemberMenu: "u2",
            memberGroups: [{ label: "Members", members: [makeMember()] }],
        });

        // when
        await user.click(screen.getByRole("button", { name: "Ban from room" }));

        // then
        expect(handleBan).toHaveBeenCalledWith("u2");
    });

    it("opens the timeout dialog for the member the moderator chose", async () => {
        // given
        const openTimeoutDialog = vi.fn();
        const target = makeMember();
        const room = makeRoom({ viewer_role: "host" });
        const user = userEvent.setup();
        membersView({
            room,
            openTimeoutDialog,
            openMemberMenu: "u2",
            memberGroups: [{ label: "Members", members: [target] }],
        });

        // when
        await user.click(screen.getByRole("button", { name: "Set timeout" }));

        // then
        expect(openTimeoutDialog).toHaveBeenCalledWith(target);
    });

    it("lifts an existing timeout", async () => {
        // given
        const handleClearTimeout = vi.fn();
        const room = makeRoom({ viewer_role: "host" });
        const user = userEvent.setup();
        membersView({
            room,
            handleClearTimeout,
            openMemberMenu: "u2",
            memberGroups: [{ label: "Members", members: [makeMember({ timeout_until: "2026-09-01T00:00:00Z" })] }],
        });

        // when
        await user.click(screen.getByRole("button", { name: "Remove timeout" }));

        // then
        expect(handleClearTimeout).toHaveBeenCalledWith("u2");
    });

    it("labels the mute control by whether the viewer already muted the room", () => {
        // given
        const room = makeRoom({ viewer_muted: true });

        // when
        membersView({ room });

        // then
        expect(screen.getByRole("button", { name: "Unmute" })).toBeInTheDocument();
    });

    it("mutes the room through the controller", async () => {
        // given
        const handleToggleMute = vi.fn();
        const user = userEvent.setup();
        membersView({ handleToggleMute });

        // when
        await user.click(screen.getByRole("button", { name: "Mute" }));

        // then
        expect(handleToggleMute).toHaveBeenCalledTimes(1);
    });

    it("shows the mute control as busy while the request is in flight", () => {
        // given
        const busy = "mute";

        // when
        membersView({ busy });

        // then
        expect(screen.getByRole("button", { name: "..." })).toBeDisabled();
    });

    it("gives a moderator the moderation and delete controls", () => {
        // given
        const room = makeRoom({ viewer_role: "host", is_system: false });

        // when
        membersView({ room });

        // then
        expect(screen.getByRole("button", { name: "Moderation" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("withholds the moderation and delete controls in a staff room", () => {
        // given
        const room = makeRoom({ viewer_role: "host", is_system: true });

        // when
        membersView({ room });

        // then
        expect(screen.queryByRole("button", { name: "Moderation" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("offers an ordinary member the way out of the room", async () => {
        // given
        const handleLeave = vi.fn();
        const room = makeRoom({ viewer_role: "member", is_system: false });
        const user = userEvent.setup();
        membersView({ room, handleLeave });

        // when
        await user.click(screen.getByRole("button", { name: "Leave" }));

        // then
        expect(handleLeave).toHaveBeenCalledTimes(1);
    });

    it("keeps the host from leaving their own room", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        membersView({ room });

        // then
        expect(screen.queryByRole("button", { name: "Leave" })).not.toBeInTheDocument();
    });

    it("asks the moderator to name the new nickname before saving it", async () => {
        // given
        const handleModSetNickname = vi.fn();
        const setNicknameDialogValue = vi.fn();
        const user = userEvent.setup();
        membersView({
            handleModSetNickname,
            setNicknameDialogValue,
            nicknameDialogTarget: makeMember(),
            nicknameDialogValue: "Endless",
        });

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(screen.getByText("Change nickname for Battler")).toBeInTheDocument();
        expect(handleModSetNickname).toHaveBeenCalledTimes(1);
    });

    it("reports why a nickname could not be saved", () => {
        // given
        const nicknameDialogError = "That nickname is taken";

        // when
        membersView({ nicknameDialogError, nicknameDialogTarget: makeMember() });

        // then
        expect(screen.getByText("That nickname is taken")).toBeInTheDocument();
    });

    it("locks the nickname dialog while it is saving", () => {
        // given
        const nicknameDialogSaving = true;

        // when
        membersView({ nicknameDialogSaving, nicknameDialogTarget: makeMember() });

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    });

    it("sets a timeout with the amount and unit the moderator chose", async () => {
        // given
        const handleSetTimeout = vi.fn();
        const setTimeoutDialogUnit = vi.fn();
        const user = userEvent.setup();
        membersView({
            handleSetTimeout,
            setTimeoutDialogUnit,
            timeoutDialogTarget: makeMember(),
            timeoutDialogAmount: "5",
            timeoutDialogUnit: "hours",
        });

        // when
        await user.selectOptions(screen.getByRole("combobox"), "weeks");
        await user.click(screen.getByRole("button", { name: "Set timeout" }));

        // then
        expect(screen.getByText("Set timeout for Battler")).toBeInTheDocument();
        expect(setTimeoutDialogUnit).toHaveBeenCalledWith("weeks");
        expect(handleSetTimeout).toHaveBeenCalledTimes(1);
    });

    it("closes the timeout dialog without setting anything", async () => {
        // given
        const setTimeoutDialogTarget = vi.fn();
        const user = userEvent.setup();
        membersView({
            setTimeoutDialogTarget,
            timeoutDialogTarget: makeMember(),
        });

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(setTimeoutDialogTarget).toHaveBeenCalledWith(null);
    });
});
