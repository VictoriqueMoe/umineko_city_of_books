import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Room } from "livekit-client";
import type { RoomController } from "../../../hooks/useRoomController";
import { makeRoomController } from "../../../hooks/useRoomController.fixture";
import { makeChatRoom, makeRoomMember, makeUser, makeWatchPartySession } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatRoom, ChatRoomMember, User, UserProfile } from "../../../types/api";
import { MobileRoomView } from "./MobileRoomView";

const mocks = vi.hoisted(() => ({
    forceMuteVoiceParticipant: vi.fn<
        (roomId: string | null | undefined, userId: string, muted: boolean) => Promise<void>
    >(() => Promise.resolve()),
}));

vi.mock("../../../hooks/mutations/chat", async importOriginal => {
    const actual = await importOriginal<typeof import("../../../hooks/mutations/chat")>();

    return {
        ...actual,
        useForceMuteVoiceParticipant: (roomId: string | null | undefined) => ({
            mutate: (
                variables: { userId: string; muted: boolean },
                callbacks?: { onError?: (err: unknown) => void },
            ) => {
                mocks
                    .forceMuteVoiceParticipant(roomId, variables.userId, variables.muted)
                    .catch((err: unknown) => callbacks?.onError?.(err));
            },
        }),
    };
});

vi.mock("../MessageList/MessageList", () => ({
    MessageList: () => <div data-testid="room-messages">messages</div>,
}));

vi.mock("../RoomInfoPanel/RoomInfoPanel", () => ({
    RoomInfoPanel: ({ tab, canUnpin }: { tab: string | null; canUnpin: boolean }) =>
        tab ? <div data-testid={`${tab}-panel`} data-can-unpin={String(canUnpin)} /> : null,
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
    WatchPartyModal: ({ isStarter, voiceEnabled }: { isStarter: boolean; voiceEnabled: boolean }) => (
        <div data-testid="watch-party-modal" data-is-starter={String(isStarter)} data-voice={String(voiceEnabled)} />
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
    return makeRoomMember({
        user: { id: "u2", username: "battler", display_name: "Battler" },
        joined_at: "2026-07-01T00:00:00Z",
        ...overrides,
    });
}

const selfMember = makeMember({ user: { id: "u1", username: "beatrice", display_name: "Beatrice" } });

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({
        name: "Rokkenjima",
        viewer_role: "member",
        created_at: "2026-07-01T00:00:00Z",
        ...overrides,
    });
}

const controllerDefaults = makeRoomController();

function makeVoice(overrides: Partial<RoomController["voice"]> = {}): RoomController["voice"] {
    return { ...controllerDefaults.voice, ...overrides };
}

function makeWatchParty(overrides: Partial<RoomController["watchParty"]> = {}): RoomController["watchParty"] {
    return { ...controllerDefaults.watchParty, screenShareEnabled: false, ...overrides };
}

function makeActiveSession(startedBy: string): RoomController["watchParty"]["activeSession"] {
    return { session: makeWatchPartySession({ started_by: startedBy }), embedURL: "", hasControl: false };
}

interface MemberGroupStub {
    label: string;
    members: ChatRoomMember[];
}

type RoomModerationStub = RoomController["moderation"];

interface ControllerOverrides {
    user?: UserProfile | null;
    room?: ChatRoom | null;
    backToRooms?: RoomController["room"]["backToRooms"];
    members?: ChatRoomMember[];
    memberGroups?: MemberGroupStub[];
    presenceMapMerged?: RoomController["members"]["presence"];
    currentMember?: ChatRoomMember | null;
    mobileView?: "members" | "chat";
    setMobileView?: RoomController["prefs"]["setMobileView"];
    typingNames?: string[];
    voice?: RoomController["voice"];
    voiceEnabled?: boolean;
    watchParty?: RoomController["watchParty"];
    invitedPartyMissing?: boolean;
    viewerTimeoutUntil?: string;
    lightboxSrc?: string | null;
    setLightboxSrc?: RoomController["panels"]["setLightboxSrc"];
    toast?: string;
    setToast?: RoomController["toast"]["show"];
    busy?: string;
    panelTab?: RoomController["panels"]["panelTab"];
    openPanel?: RoomController["panels"]["openPanel"];
    editProfileOpen?: boolean;
    setEditProfileOpen?: RoomController["panels"]["setEditProfileOpen"];
    inviteModalOpen?: boolean;
    setInviteModalOpen?: RoomController["panels"]["setInviteModalOpen"];
    moderationDialogOpen?: boolean;
    setModerationDialogOpen?: RoomController["panels"]["setModerationDialogOpen"];
    openMemberMenu?: string | null;
    setOpenMemberMenu?: RoomModerationStub["setOpenMemberMenu"];
    setMembers?: RoomController["members"]["set"];
    nicknameDialogTarget?: ChatRoomMember | null;
    setNicknameDialogTarget?: RoomModerationStub["setNicknameDialogTarget"];
    nicknameDialogValue?: string;
    setNicknameDialogValue?: RoomModerationStub["setNicknameDialogValue"];
    nicknameDialogError?: string;
    nicknameDialogSaving?: boolean;
    timeoutDialogTarget?: ChatRoomMember | null;
    setTimeoutDialogTarget?: RoomModerationStub["setTimeoutDialogTarget"];
    timeoutDialogAmount?: string;
    setTimeoutDialogAmount?: RoomModerationStub["setTimeoutDialogAmount"];
    timeoutDialogUnit?: string;
    setTimeoutDialogUnit?: RoomModerationStub["setTimeoutDialogUnit"];
    timeoutDialogError?: string;
    timeoutDialogSaving?: boolean;
    openNicknameDialog?: RoomModerationStub["openNicknameDialog"];
    openTimeoutDialog?: RoomModerationStub["openTimeoutDialog"];
    handleSentMessage?: RoomController["session"]["onSent"];
    handleModSetNickname?: RoomModerationStub["handleModSetNickname"];
    handleModUnlockNickname?: RoomModerationStub["handleModUnlockNickname"];
    handleSetTimeout?: RoomModerationStub["handleSetTimeout"];
    handleClearTimeout?: RoomModerationStub["handleClearTimeout"];
    handleKick?: RoomModerationStub["handleKick"];
    handleBan?: RoomModerationStub["handleBan"];
    handleToggleMute?: RoomController["room"]["toggleMute"];
    handleLeave?: RoomController["room"]["leave"];
    handleDelete?: RoomController["room"]["remove"];
    handleJumpToMessage?: RoomController["anchor"]["jumpTo"];
    handleEditLast?: RoomController["session"]["editLast"];
}

function makeController(overrides: ControllerOverrides = {}): RoomController {
    const members = overrides.members ?? [selfMember, makeMember()];

    return makeRoomController({
        room: {
            ...controllerDefaults.room,
            data: "room" in overrides ? (overrides.room ?? null) : makeRoom(),
            viewerTimeoutUntil: overrides.viewerTimeoutUntil,
            toggleMute: overrides.handleToggleMute ?? vi.fn(),
            leave: overrides.handleLeave ?? vi.fn(),
            remove: overrides.handleDelete ?? vi.fn(),
            backToRooms: overrides.backToRooms ?? vi.fn(),
        },
        session: {
            ...controllerDefaults.session,
            viewer: "user" in overrides ? (overrides.user ?? null) : viewer,
            typingNames: overrides.typingNames ?? [],
            onSent: overrides.handleSentMessage ?? vi.fn(),
            editLast: overrides.handleEditLast ?? vi.fn(),
        },
        members: {
            ...controllerDefaults.members,
            list: members,
            groups: overrides.memberGroups ?? [{ label: "Members", members }],
            presence: overrides.presenceMapMerged ?? {},
            current: overrides.currentMember ?? selfMember,
            set: overrides.setMembers ?? vi.fn(),
        },
        moderation: {
            ...controllerDefaults.moderation,
            busy: overrides.busy ?? "",
            openMemberMenu: overrides.openMemberMenu ?? null,
            setOpenMemberMenu: overrides.setOpenMemberMenu ?? vi.fn(),
            nicknameDialogTarget: overrides.nicknameDialogTarget ?? null,
            setNicknameDialogTarget: overrides.setNicknameDialogTarget ?? vi.fn(),
            nicknameDialogValue: overrides.nicknameDialogValue ?? "",
            setNicknameDialogValue: overrides.setNicknameDialogValue ?? vi.fn(),
            nicknameDialogError: overrides.nicknameDialogError ?? "",
            nicknameDialogSaving: overrides.nicknameDialogSaving ?? false,
            timeoutDialogTarget: overrides.timeoutDialogTarget ?? null,
            setTimeoutDialogTarget: overrides.setTimeoutDialogTarget ?? vi.fn(),
            timeoutDialogAmount: overrides.timeoutDialogAmount ?? "10",
            setTimeoutDialogAmount: overrides.setTimeoutDialogAmount ?? vi.fn(),
            timeoutDialogUnit: overrides.timeoutDialogUnit ?? "seconds",
            setTimeoutDialogUnit: overrides.setTimeoutDialogUnit ?? vi.fn(),
            timeoutDialogError: overrides.timeoutDialogError ?? "",
            timeoutDialogSaving: overrides.timeoutDialogSaving ?? false,
            formatTimeoutUntil: (value?: string) => `until ${value ?? "never"}`,
            openNicknameDialog: overrides.openNicknameDialog ?? vi.fn(),
            openTimeoutDialog: overrides.openTimeoutDialog ?? vi.fn(),
            handleModSetNickname: overrides.handleModSetNickname ?? vi.fn(),
            handleModUnlockNickname: overrides.handleModUnlockNickname ?? vi.fn(),
            handleSetTimeout: overrides.handleSetTimeout ?? vi.fn(),
            handleClearTimeout: overrides.handleClearTimeout ?? vi.fn(),
            handleKick: overrides.handleKick ?? vi.fn(),
            handleBan: overrides.handleBan ?? vi.fn(),
        },
        prefs: {
            ...controllerDefaults.prefs,
            mobileView: overrides.mobileView ?? "chat",
            setMobileView: overrides.setMobileView ?? vi.fn(),
        },
        anchor: {
            ...controllerDefaults.anchor,
            jumpTo: overrides.handleJumpToMessage ?? vi.fn(),
        },
        voice: { ...(overrides.voice ?? makeVoice()), enabled: overrides.voiceEnabled ?? true },
        watchParty: {
            ...(overrides.watchParty ?? makeWatchParty()),
            invitedPartyMissing: overrides.invitedPartyMissing ?? false,
        },
        panels: {
            ...controllerDefaults.panels,
            panelTab: overrides.panelTab ?? null,
            openPanel: overrides.openPanel ?? vi.fn(),
            lightboxSrc: overrides.lightboxSrc ?? null,
            setLightboxSrc: overrides.setLightboxSrc ?? vi.fn(),
            editProfileOpen: overrides.editProfileOpen ?? false,
            setEditProfileOpen: overrides.setEditProfileOpen ?? vi.fn(),
            inviteModalOpen: overrides.inviteModalOpen ?? false,
            setInviteModalOpen: overrides.setInviteModalOpen ?? vi.fn(),
            moderationDialogOpen: overrides.moderationDialogOpen ?? false,
            setModerationDialogOpen: overrides.setModerationDialogOpen ?? vi.fn(),
        },
        toast: {
            message: overrides.toast ?? "",
            show: overrides.setToast ?? vi.fn(),
        },
    });
}

function renderView(overrides: ControllerOverrides = {}) {
    const controller = makeController(overrides);
    const result = renderWithProviders(<MobileRoomView controller={controller} />);

    return { ...result, controller };
}

function membersView(overrides: ControllerOverrides = {}) {
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
        const backToRooms = vi.fn();
        const user = userEvent.setup();
        renderView({ backToRooms });

        // when
        await user.click(screen.getByLabelText("Back to rooms"));

        // then
        expect(backToRooms).toHaveBeenCalledOnce();
    });

    it("opens the search, the pins and the member list from the top bar", async () => {
        // given
        const openPanel = vi.fn();
        const setMobileView = vi.fn();
        const user = userEvent.setup();
        renderView({ openPanel, setMobileView });

        // when
        await user.click(screen.getByLabelText("Search messages"));
        await user.click(screen.getByLabelText("Pinned messages"));
        await user.click(screen.getByLabelText("Members"));

        // then
        expect(openPanel).toHaveBeenCalledWith("search");
        expect(openPanel).toHaveBeenCalledWith("pins");
        expect(setMobileView).toHaveBeenCalledWith("members");
    });

    it("keeps the search panel out of the way until it is opened", () => {
        // given
        const panelTab = null;

        // when
        renderView({ panelTab });

        // then
        expect(screen.queryByTestId("search-panel")).not.toBeInTheDocument();
    });

    it("lets only a moderator unpin from the pinned panel", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        renderView({ room, panelTab: "pins" });

        // then
        expect(screen.getByTestId("pins-panel")).toHaveAttribute("data-can-unpin", "true");
    });

    it("denies unpinning to an ordinary member", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        renderView({ room, panelTab: "pins" });

        // then
        expect(screen.getByTestId("pins-panel")).toHaveAttribute("data-can-unpin", "false");
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
        const voice = makeVoice({ status: "connected", room: new Room() });

        // when
        renderView({ voice });

        // then
        expect(await screen.findByTestId("voice-bar")).toHaveAttribute("data-can-moderate", "false");
    });

    it("lets the host moderate the call", async () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        renderView({ room, voice: makeVoice({ status: "connected", room: new Room() }) });

        // then
        expect(await screen.findByTestId("voice-bar")).toHaveAttribute("data-can-moderate", "true");
    });

    it("sends a server mute to the room the call belongs to", async () => {
        // given
        const user = userEvent.setup();
        renderView({ voice: makeVoice({ status: "connected", room: new Room() }) });
        await screen.findByTestId("voice-bar");

        // when
        await user.click(screen.getByRole("button", { name: "server mute battler" }));

        // then
        expect(mocks.forceMuteVoiceParticipant).toHaveBeenCalledWith("room-1", "battler", true);
    });

    it("tells the moderator when a server mute did not take", async () => {
        // given
        mocks.forceMuteVoiceParticipant.mockRejectedValue(new Error("LiveKit said no"));
        const setToast = vi.fn();
        const user = userEvent.setup();
        renderView({ setToast, voice: makeVoice({ status: "connected", room: new Room() }) });
        await screen.findByTestId("voice-bar");

        // when
        await user.click(screen.getByRole("button", { name: "server mute battler" }));

        // then
        await waitFor(() => {
            expect(setToast).toHaveBeenCalledWith("LiveKit said no");
        });
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
        expect(screen.getByText(/is typing/)).toHaveTextContent("Battler is typing...");
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
        const watchParty = makeWatchParty({ activeSession: makeActiveSession("u1") });

        // when
        renderView({ watchParty });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-is-starter", "true");
    });

    it("opens the watch party as a guest for everybody else", async () => {
        // given
        const watchParty = makeWatchParty({ activeSession: makeActiveSession("u9") });

        // when
        renderView({ watchParty });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-is-starter", "false");
    });

    it("offers watch party voice when the site allows voice and screen share is on", async () => {
        // given
        const watchParty = makeWatchParty({
            activeSession: makeActiveSession("u1"),
            screenShareEnabled: true,
        });

        // when
        renderView({ watchParty, voiceEnabled: true });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-voice", "true");
    });

    it("obeys the site voice setting inside a watch party", async () => {
        // given
        const watchParty = makeWatchParty({
            activeSession: makeActiveSession("u1"),
            screenShareEnabled: true,
        });

        // when
        renderView({ watchParty, voiceEnabled: false });

        // then
        expect(await screen.findByTestId("watch-party-modal")).toHaveAttribute("data-voice", "false");
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

    it("marks a member who is timed out", () => {
        // given
        const memberGroups = [{ label: "Members", members: [makeMember({ timeout_until: "2026-09-01T00:00:00Z" })] }];

        // when
        membersView({ memberGroups });

        // then
        expect(screen.getByLabelText("Timed out until until 2026-09-01T00:00:00Z")).toBeInTheDocument();
    });

    it("says whether a member is watching the room right now", () => {
        // given
        const presenceMapMerged: RoomController["members"]["presence"] = { u2: "active" };

        // when
        membersView({ presenceMapMerged, memberGroups: [{ label: "Members", members: [makeMember()] }] });

        // then
        expect(screen.getByLabelText("Active in this room")).toBeInTheDocument();
    });

    it("leaves the online and offline headings off the phone roster", () => {
        // given
        const memberGroups = [{ label: "Everyone", members: [makeMember()] }];

        // when
        membersView({ memberGroups });

        // then
        expect(screen.getByText("Everyone")).toBeInTheDocument();
        expect(screen.queryByText("Online")).not.toBeInTheDocument();
        expect(screen.queryByText("Offline")).not.toBeInTheDocument();
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

    it("offers the invite control to site staff who do not host the room", () => {
        // given
        const room = makeRoom({ viewer_role: "member", is_system: false });

        // when
        membersView({ room, user: staffViewer });

        // then
        expect(screen.getByLabelText("Invite members")).toBeInTheDocument();
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
        expect(screen.getByText(/Change nickname for/)).toHaveTextContent("Change nickname for Battler");
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
        expect(screen.getByText(/Set timeout for/)).toHaveTextContent("Set timeout for Battler");
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
