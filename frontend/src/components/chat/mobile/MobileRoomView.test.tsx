import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
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

type ControllerPatch = {
    [K in Exclude<keyof RoomController, "anchor" | "capabilities">]?: Partial<RoomController[K]>;
};

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

const staffViewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice", role: "admin" });

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

function renderView(patch: ControllerPatch = {}, view: "chat" | "members" = "chat") {
    const defaults = makeRoomController();
    const members = patch.members?.list ?? [selfMember, makeMember()];

    const controller = makeRoomController({
        room: { ...defaults.room, data: makeRoom(), ...patch.room },
        session: { ...defaults.session, viewer, ...patch.session },
        members: {
            ...defaults.members,
            list: members,
            groups: [{ label: "Members", members }],
            current: selfMember,
            ...patch.members,
        },
        moderation: {
            ...defaults.moderation,
            formatTimeoutUntil: value => `until ${value ?? "never"}`,
            ...patch.moderation,
        },
        prefs: { ...defaults.prefs, mobileView: view, ...patch.prefs },
        voice: { ...defaults.voice, ...patch.voice },
        watchParty: { ...defaults.watchParty, ...patch.watchParty },
        panels: { ...defaults.panels, ...patch.panels },
        toast: { ...defaults.toast, ...patch.toast },
    });

    return renderWithProviders(<MobileRoomView controller={controller} />);
}

describe("MobileRoomView chat view", () => {
    const unreadyCases: { name: string; patch: ControllerPatch }[] = [
        { name: "renders nothing until the viewer is signed in", patch: { session: { viewer: null } } },
        { name: "renders nothing until the room has loaded", patch: { room: { data: null } } },
    ];

    it.each(unreadyCases)("$name", ({ patch }) => {
        // given the missing viewer or room, from the table row

        // when
        const { container } = renderView(patch);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    const topBarCases: { name: string; room: Partial<ChatRoom>; meta: string; badges: string[] }[] = [
        {
            name: "names the room and says how many people and how open it is",
            room: { member_count: 7, is_public: true },
            meta: "7 members · public",
            badges: [],
        },
        {
            name: "says when a room is private",
            room: { is_public: false },
            meta: "2 members · private",
            badges: [],
        },
        {
            name: "counts the members it has when the server sent no count",
            room: {
                member_count: undefined as unknown as number,
                members: [
                    { id: "u1", username: "beatrice", display_name: "Beatrice" },
                    { id: "u2", username: "battler", display_name: "Battler" },
                    { id: "u3", username: "ronove", display_name: "Ronove" },
                ],
            },
            meta: "3 members · public",
            badges: [],
        },
        {
            name: "badges a staff room and a roleplay room",
            room: { is_system: true, is_rp: true },
            meta: "2 members · public",
            badges: ["Staff", "RP"],
        },
    ];

    it.each(topBarCases)("$name", ({ room, meta, badges }) => {
        // given
        const data = makeRoom(room);

        // when
        renderView({ room: { data } });

        // then
        expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        expect(screen.getByText(/members/)).toHaveTextContent(meta);
        expect(screen.queryAllByText(/^(Staff|RP)$/).map(badge => badge.textContent)).toEqual(badges);
    });

    it("goes back to the room directory and opens the search, the pins and the member list from the top bar", async () => {
        // given
        const backToRooms = vi.fn();
        const openPanel = vi.fn();
        const setMobileView = vi.fn();
        const user = userEvent.setup();
        renderView({ room: { backToRooms }, panels: { openPanel }, prefs: { setMobileView } });

        // when
        await user.click(screen.getByLabelText("Back to rooms"));
        await user.click(screen.getByLabelText("Search messages"));
        await user.click(screen.getByLabelText("Pinned messages"));
        await user.click(screen.getByLabelText("Members"));

        // then
        expect(backToRooms).toHaveBeenCalledOnce();
        expect(openPanel).toHaveBeenCalledWith("search");
        expect(openPanel).toHaveBeenCalledWith("pins");
        expect(setMobileView).toHaveBeenCalledWith("members");
    });

    it("keeps the info panels and the voice bar away until they are opened or joined", () => {
        // given
        const voice = { status: "idle" as const, room: null };

        // when
        renderView({ voice, panels: { panelTab: null } });

        // then
        expect(screen.queryByTestId(/-panel$/)).not.toBeInTheDocument();
        expect(screen.queryByTestId("voice-bar")).not.toBeInTheDocument();
    });

    const moderatorPowerCases: { name: string; viewerRole: string; granted: string }[] = [
        {
            name: "lets only the host unpin from the pinned panel and moderate the call",
            viewerRole: "host",
            granted: "true",
        },
        {
            name: "denies unpinning and call moderation to an ordinary member connected to the call",
            viewerRole: "member",
            granted: "false",
        },
    ];

    it.each(moderatorPowerCases)("$name", async ({ viewerRole, granted }) => {
        // given
        const data = makeRoom({ viewer_role: viewerRole });

        // when
        renderView({ room: { data }, panels: { panelTab: "pins" }, voice: { status: "connected", room: new Room() } });

        // then
        expect(screen.getByTestId("pins-panel")).toHaveAttribute("data-can-unpin", granted);
        expect(await screen.findByTestId("voice-bar")).toHaveAttribute("data-can-moderate", granted);
    });

    it("sends a server mute to the room the call belongs to and tells the moderator when it did not take", async () => {
        // given
        mocks.forceMuteVoiceParticipant.mockRejectedValueOnce(new Error("LiveKit said no"));
        const show = vi.fn();
        const user = userEvent.setup();
        renderView({ toast: { show }, voice: { status: "connected", room: new Room() } });
        await screen.findByTestId("voice-bar");

        // when
        await user.click(screen.getByRole("button", { name: "server mute battler" }));

        // then
        expect(mocks.forceMuteVoiceParticipant).toHaveBeenCalledWith("room-1", "battler", true);
        await waitFor(() => {
            expect(show).toHaveBeenCalledWith("LiveKit said no");
        });
    });

    it("gives the composer the room, the mention pool and the viewer's timeout", () => {
        // given
        const viewerTimeoutUntil = "2026-08-02T12:00:00Z";

        // when
        renderView({ room: { viewerTimeoutUntil } });

        // then
        const composer = screen.getByTestId("composer");
        expect(composer).toHaveAttribute("data-room-id", "room-1");
        expect(composer).toHaveAttribute("data-timeout-until", "2026-08-02T12:00:00Z");
        expect(composer).toHaveAttribute("data-mention-pool", "beatrice,battler");
    });

    const callControlCases: { name: string; isSystem: boolean; offered: boolean }[] = [
        { name: "offers the voice and watch party controls in an ordinary room", isSystem: false, offered: true },
        { name: "takes the voice and watch party controls away in a staff room", isSystem: true, offered: false },
    ];

    it.each(callControlCases)("$name", ({ isSystem, offered }) => {
        // given
        const data = makeRoom({ is_system: isSystem });

        // when
        renderView({ room: { data } });

        // then
        expect(screen.queryByTitle("Join voice") !== null).toBe(offered);
        expect(screen.queryByTestId("watch-party-button") !== null).toBe(offered);
    });

    it("shows the message list and who is typing", () => {
        // given
        const typingNames = ["Battler"];

        // when
        renderView({ session: { typingNames } });

        // then
        expect(screen.getByTestId("room-messages")).toBeInTheDocument();
        expect(screen.getByText(/is typing/)).toHaveTextContent("Battler is typing...");
    });

    it("tells the viewer when the watch party they were invited to has ended", () => {
        // given
        const invitedPartyMissing = true;

        // when
        renderView({ watchParty: { invitedPartyMissing } });

        // then
        expect(screen.getByText("That watch party has ended.")).toBeInTheDocument();
    });

    it("shows a toast the controller raised", () => {
        // given
        const message = "1 member invited";

        // when
        renderView({ toast: { message } });

        // then
        expect(screen.getByText("1 member invited")).toBeInTheDocument();
    });

    const watchPartyCases: {
        name: string;
        startedBy: string;
        voiceEnabled: boolean;
        starter: string;
        voice: string;
    }[] = [
        {
            name: "opens the watch party for the person who started it, with voice when the site allows voice and screen share is on",
            startedBy: "u1",
            voiceEnabled: true,
            starter: "true",
            voice: "true",
        },
        {
            name: "opens the watch party for the person who started it, obeying the site voice setting",
            startedBy: "u1",
            voiceEnabled: false,
            starter: "true",
            voice: "false",
        },
        {
            name: "opens the watch party as a guest for everybody else",
            startedBy: "u9",
            voiceEnabled: true,
            starter: "false",
            voice: "true",
        },
    ];

    it.each(watchPartyCases)("$name", async ({ startedBy, voiceEnabled, starter, voice }) => {
        // given
        const activeSession = {
            session: makeWatchPartySession({ started_by: startedBy }),
            embedURL: "",
            hasControl: false,
        };

        // when
        renderView({ voice: { enabled: voiceEnabled }, watchParty: { activeSession, screenShareEnabled: true } });

        // then
        const modal = await screen.findByTestId("watch-party-modal");
        expect(modal).toHaveAttribute("data-is-starter", starter);
        expect(modal).toHaveAttribute("data-voice", voice);
    });

    const inviteCases: { name: string; button: string; toast: string }[] = [
        { name: "reports a single invitation in the singular", button: "invite one", toast: "1 member invited" },
        { name: "reports several invitations in the plural", button: "invite three", toast: "3 members invited" },
        {
            name: "explains when nobody could be invited",
            button: "invite nobody",
            toast: "No one invited (all were already members or blocked)",
        },
    ];

    it.each(inviteCases)("$name", async ({ button, toast }) => {
        // given
        const show = vi.fn();
        const user = userEvent.setup();
        renderView({ toast: { show }, panels: { inviteModalOpen: true } });

        // when
        await user.click(screen.getByRole("button", { name: button }));

        // then
        expect(show).toHaveBeenCalledWith(toast);
    });

    it("swaps in the saved room profile without disturbing the other members", async () => {
        // given
        const set = vi.fn();
        const user = userEvent.setup();
        renderView({ members: { set }, panels: { editProfileOpen: true } });

        // when
        await user.click(screen.getByRole("button", { name: "save room profile" }));

        // then
        const updater = set.mock.calls[0][0] as (prev: ChatRoomMember[]) => ChatRoomMember[];
        const next = updater([selfMember, makeMember()]);
        expect(next[0].nickname).toBe("Golden Witch");
        expect(next[1].nickname).toBe("");
    });

    it("shows the lightbox for the image the viewer opened", () => {
        // given
        const lightboxSrc = "https://cdn.example/photo.png";

        // when
        renderView({ panels: { lightboxSrc } });

        // then
        expect(screen.getByRole("dialog")).toBeInTheDocument();
    });
});

describe("MobileRoomView members view", () => {
    it("heads the list with how many people are in the room and goes back to the chat", async () => {
        // given
        const members = [
            selfMember,
            makeMember(),
            makeMember({ user: { id: "u3", username: "ronove", display_name: "Ronove" } }),
        ];
        const setMobileView = vi.fn();
        const user = userEvent.setup();
        renderView(
            { members: { list: members, groups: [{ label: "Online", members }] }, prefs: { setMobileView } },
            "members",
        );

        // when
        await user.click(screen.getByLabelText("Back to chat"));

        // then
        expect(screen.getByText("Members")).toBeInTheDocument();
        expect(screen.getByText("3 members")).toBeInTheDocument();
        expect(setMobileView).toHaveBeenCalledWith("chat");
    });

    it("groups the members under the labels the controller gave, without online or offline headings", () => {
        // given
        const groups = [
            { label: "In Voice", members: [selfMember] },
            { label: "Everyone", members: [makeMember()] },
        ];

        // when
        renderView({ members: { groups } }, "members");

        // then
        expect(screen.getByText("In Voice")).toBeInTheDocument();
        expect(screen.getByText("Everyone")).toBeInTheDocument();
        expect(screen.queryByText("Online")).not.toBeInTheDocument();
        expect(screen.queryByText("Offline")).not.toBeInTheDocument();
    });

    it("says whether a member is watching the room right now and marks one who is timed out", () => {
        // given
        const groups = [{ label: "Members", members: [makeMember({ timeout_until: "2026-09-01T00:00:00Z" })] }];

        // when
        renderView({ members: { groups, presence: { u2: "active" } } }, "members");

        // then
        expect(screen.getByLabelText("Active in this room")).toBeInTheDocument();
        expect(screen.getByLabelText("Timed out until until 2026-09-01T00:00:00Z")).toBeInTheDocument();
    });

    const viewerControlCases: {
        name: string;
        account: UserProfile;
        room: Partial<ChatRoom>;
        invite: boolean;
        memberActions: boolean;
        roomAdmin: boolean;
        leave: boolean;
    }[] = [
        {
            name: "gives the host of an ordinary room every moderator control but no way out",
            account: viewer,
            room: { viewer_role: "host", is_system: false },
            invite: true,
            memberActions: true,
            roomAdmin: true,
            leave: false,
        },
        {
            name: "gives an ordinary member the way out and no moderator controls",
            account: viewer,
            room: { viewer_role: "member", is_system: false },
            invite: false,
            memberActions: false,
            roomAdmin: false,
            leave: true,
        },
        {
            name: "withholds the moderator controls and the way out from the host of a staff room",
            account: viewer,
            room: { viewer_role: "host", is_system: true },
            invite: false,
            memberActions: false,
            roomAdmin: false,
            leave: false,
        },
        {
            name: "gives site staff who do not host the room every moderator control and the way out",
            account: staffViewer,
            room: { viewer_role: "member", is_system: false },
            invite: true,
            memberActions: true,
            roomAdmin: true,
            leave: true,
        },
    ];

    it.each(viewerControlCases)("$name", ({ account, room, invite, memberActions, roomAdmin, leave }) => {
        // given
        const data = makeRoom(room);

        // when
        renderView({ session: { viewer: account }, room: { data } }, "members");

        // then
        expect(screen.queryByLabelText("Invite members") !== null).toBe(invite);
        expect(screen.queryByLabelText("Moderator actions") !== null).toBe(memberActions);
        expect(screen.queryByRole("button", { name: "Moderation" }) !== null).toBe(roomAdmin);
        expect(screen.queryByRole("button", { name: "Delete" }) !== null).toBe(roomAdmin);
        expect(screen.queryByRole("button", { name: "Leave" }) !== null).toBe(leave);
    });

    it("puts the room profile control on the viewer's own row alone and opens the editor from it", async () => {
        // given
        const setEditProfileOpen = vi.fn();
        const user = userEvent.setup();
        renderView({ panels: { setEditProfileOpen } }, "members");

        // when
        await user.click(screen.getByLabelText("Edit profile in this room"));

        // then
        expect(screen.getAllByLabelText("Edit profile in this room")).toHaveLength(1);
        expect(setEditProfileOpen).toHaveBeenCalledWith(true);
    });

    const hostRoster: ControllerPatch = {
        room: { data: makeRoom({ viewer_role: "host" }) },
        members: { groups: [{ label: "Members", members: [makeMember()] }] },
    };

    it("offers the moderator actions to the host", async () => {
        // given
        const setOpenMemberMenu = vi.fn();
        const user = userEvent.setup();
        renderView({ ...hostRoster, moderation: { setOpenMemberMenu } }, "members");

        // when
        await user.click(screen.getByLabelText("Moderator actions"));

        // then
        expect(setOpenMemberMenu).toHaveBeenCalledTimes(1);
    });

    it("kicks a member and closes the menu behind it", async () => {
        // given
        const handleKick = vi.fn();
        const setOpenMemberMenu = vi.fn();
        const user = userEvent.setup();
        renderView({ ...hostRoster, moderation: { openMemberMenu: "u2", handleKick, setOpenMemberMenu } }, "members");

        // when
        await user.click(screen.getByRole("button", { name: "Kick member" }));

        // then
        expect(setOpenMemberMenu).toHaveBeenCalledWith(null);
        expect(handleKick).toHaveBeenCalledWith("u2");
    });

    it("offers a host kicking, banning and timeouts from the open menu but not renaming", async () => {
        // given
        const handleBan = vi.fn();
        const openTimeoutDialog = vi.fn();
        const handleClearTimeout = vi.fn();
        const target = makeMember({ timeout_until: "2026-09-01T00:00:00Z" });
        const user = userEvent.setup();
        renderView(
            {
                ...hostRoster,
                members: { groups: [{ label: "Members", members: [target] }] },
                moderation: { openMemberMenu: "u2", handleBan, openTimeoutDialog, handleClearTimeout },
            },
            "members",
        );

        // when
        await user.click(screen.getByRole("button", { name: "Ban from room" }));
        await user.click(screen.getByRole("button", { name: "Set timeout" }));
        await user.click(screen.getByRole("button", { name: "Remove timeout" }));

        // then
        expect(screen.getByRole("button", { name: "Kick member" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Ban from room" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Change nickname" })).not.toBeInTheDocument();
        expect(handleBan).toHaveBeenCalledWith("u2");
        expect(openTimeoutDialog).toHaveBeenCalledWith(target);
        expect(handleClearTimeout).toHaveBeenCalledWith("u2");
    });

    it("offers site staff the nickname controls as well", () => {
        // given
        const groups = [{ label: "Members", members: [makeMember({ nickname_locked: true })] }];

        // when
        renderView(
            { session: { viewer: staffViewer }, members: { groups }, moderation: { openMemberMenu: "u2" } },
            "members",
        );

        // then
        expect(screen.getByRole("button", { name: "Change nickname" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Reset/unlock nickname" })).toBeInTheDocument();
    });

    const muteLabelCases: { name: string; muted: boolean; busy: string | null; label: string; disabled: boolean }[] = [
        {
            name: "labels the mute control by whether the viewer already muted the room",
            muted: true,
            busy: null,
            label: "Unmute",
            disabled: false,
        },
        {
            name: "shows the mute control as busy while the request is in flight",
            muted: false,
            busy: "mute",
            label: "...",
            disabled: true,
        },
    ];

    it.each(muteLabelCases)("$name", ({ muted, busy, label, disabled }) => {
        // given
        const data = makeRoom({ viewer_muted: muted });

        // when
        renderView({ room: { data }, moderation: { busy } }, "members");

        // then
        expect(screen.getByRole("button", { name: label })).toHaveProperty("disabled", disabled);
    });

    it("mutes and leaves the room through the controller", async () => {
        // given
        const toggleMute = vi.fn();
        const leave = vi.fn();
        const user = userEvent.setup();
        renderView({ room: { toggleMute, leave } }, "members");

        // when
        await user.click(screen.getByRole("button", { name: "Mute" }));
        await user.click(screen.getByRole("button", { name: "Leave" }));

        // then
        expect(toggleMute).toHaveBeenCalledTimes(1);
        expect(leave).toHaveBeenCalledTimes(1);
    });

    it("asks the moderator to name the new nickname before saving it and reports why it could not be saved", async () => {
        // given
        const handleModSetNickname = vi.fn();
        const user = userEvent.setup();
        renderView(
            {
                moderation: {
                    handleModSetNickname,
                    nicknameDialogTarget: makeMember(),
                    nicknameDialogValue: "Endless",
                    nicknameDialogError: "That nickname is taken",
                },
            },
            "members",
        );

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(screen.getByText(/Change nickname for/)).toHaveTextContent("Change nickname for Battler");
        expect(screen.getByPlaceholderText("Nickname (leave blank to clear)")).toHaveValue("Endless");
        expect(screen.getByText("That nickname is taken")).toBeInTheDocument();
        expect(handleModSetNickname).toHaveBeenCalledTimes(1);
    });

    it("locks the nickname dialog while it is saving", () => {
        // given
        const nicknameDialogSaving = true;

        // when
        renderView({ moderation: { nicknameDialogSaving, nicknameDialogTarget: makeMember() } }, "members");

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    });

    it("sets a timeout with the amount and unit the moderator chose, and closes the dialog on cancel", async () => {
        // given
        const handleSetTimeout = vi.fn();
        const setTimeoutDialogUnit = vi.fn();
        const setTimeoutDialogTarget = vi.fn();
        const user = userEvent.setup();
        renderView(
            {
                moderation: {
                    handleSetTimeout,
                    setTimeoutDialogUnit,
                    setTimeoutDialogTarget,
                    timeoutDialogTarget: makeMember(),
                    timeoutDialogAmount: "5",
                    timeoutDialogUnit: "hours",
                },
            },
            "members",
        );

        // when
        await user.selectOptions(screen.getByRole("combobox"), "weeks");
        await user.click(screen.getByRole("button", { name: "Set timeout" }));
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.getByText(/Set timeout for/)).toHaveTextContent("Set timeout for Battler");
        expect(screen.getByRole("spinbutton")).toHaveValue(5);
        expect(setTimeoutDialogUnit).toHaveBeenCalledWith("weeks");
        expect(handleSetTimeout).toHaveBeenCalledTimes(1);
        expect(setTimeoutDialogTarget).toHaveBeenCalledWith(null);
    });
});
