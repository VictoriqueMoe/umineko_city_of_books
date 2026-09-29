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
    WatchPartyButton: () => <button type="button">Watch party</button>,
}));

vi.mock("../../components/chat/WatchParty/WatchPartyModal", () => ({
    WatchPartyModal: (props: { isStarter: boolean; voiceEnabled: boolean }) => (
        <div
            data-testid="watch-party-modal"
            data-starter={String(props.isStarter)}
            data-voice={String(props.voiceEnabled)}
        />
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
    VoiceButton: () => <button type="button">Voice</button>,
}));

vi.mock("../../components/Lightbox/Lightbox", () => ({
    Lightbox: (props: { src: string }) => <div data-testid="lightbox">{props.src}</div>,
}));

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });
const staff = makeUser({ id: "viewer-1", role: "moderator" });
const TIMED_OUT_UNTIL = "2026-02-01T13:00:00Z";
const ROOM_CONTROLS = ["Moderation", "Delete Room", "Leave Room", "+ Invite", "Voice", "Watch party"];
const MEMBER_CONTROLS = [
    "Moderator actions",
    "Change nickname",
    "Reset/unlock nickname",
    "Kick member",
    "Ban from room",
    "Set timeout",
];
const OVERLAYS = ["search-panel", "pins-panel", "invite-modal", "moderation-dialog", "lightbox"];

function makeMemberUser(overrides: Partial<User> = {}): User {
    return makePublicUser({ id: "member-1", ...overrides });
}

function makeMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return makeRoomMember({ user: makeMemberUser(), ...overrides });
}

function offeredButtons(labels: string[]): string[] {
    return labels.filter(label => screen.queryByRole("button", { name: label }) !== null);
}

interface RoomSetup {
    viewer?: UserProfile | null;
    room?: ChatRoom | null;
    members?: ChatRoomMember[];
    toast?: string;
    roomState?: Partial<RoomController["room"]>;
    session?: Partial<RoomController["session"]>;
    roster?: Partial<RoomController["members"]>;
    moderation?: Partial<RoomController["moderation"]>;
    prefs?: Partial<RoomController["prefs"]>;
    voice?: Partial<RoomController["voice"]>;
    watchParty?: Partial<RoomController["watchParty"]>;
    panels?: Partial<RoomController["panels"]>;
}

function renderRoom(setup: RoomSetup = {}) {
    const roomViewer = setup.viewer === undefined ? viewer : setup.viewer;
    const room = setup.room === undefined ? makeChatRoom() : setup.room;
    const members = setup.members ?? [];
    const base = makeRoomController();

    const controller = makeRoomController({
        capabilities: roomCapabilities(room, roomViewer),
        room: {
            ...base.room,
            data: room,
            join: vi.fn(),
            toggleMute: vi.fn(),
            backToRooms: vi.fn(),
            ...setup.roomState,
        },
        session: { ...base.session, viewer: roomViewer, ...setup.session },
        members: { ...base.members, list: members, groups: [{ label: "Members", members }], ...setup.roster },
        moderation: {
            ...base.moderation,
            setOpenMemberMenu: vi.fn(),
            formatTimeoutUntil: (value?: string) => `until ${value ?? "never"}`,
            openTimeoutDialog: vi.fn(),
            handleModSetNickname: vi.fn(),
            handleSetTimeout: vi.fn(),
            handleClearTimeout: vi.fn(),
            handleKick: vi.fn(),
            handleBan: vi.fn(),
            ...setup.moderation,
        },
        prefs: { ...base.prefs, toggleSidebar: vi.fn(), toggleDescExpanded: vi.fn(), ...setup.prefs },
        voice: { ...base.voice, ...setup.voice },
        watchParty: { ...base.watchParty, ...setup.watchParty },
        panels: { ...base.panels, openPanel: vi.fn(), setEditProfileOpen: vi.fn(), ...setup.panels },
        toast: { message: setup.toast ?? null, show: vi.fn() },
    });
    mocks.useRoomController.mockReturnValue(controller);

    const { container } = renderWithProviders(<RoomPage />, { user: roomViewer });

    return { container, controller };
}

const hostRoom = makeChatRoom({ viewer_role: "host" });
const menuTarget = makeMember();
const hostMenu: RoomSetup = { room: hostRoom, members: [menuTarget], moderation: { openMemberMenu: "member-1" } };

beforeEach(() => {
    mocks.useIsMobile.mockReturnValue(false);
    mocks.forceMute.mockReset();
});

describe("RoomPage gates", () => {
    it("shows nothing at all to a signed out visitor", () => {
        // given
        const signedOut = null;

        // when
        const { container } = renderRoom({ viewer: signedOut });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("waits while the room is being loaded", () => {
        // given
        const roomState = { loading: true };

        // when
        renderRoom({ roomState });

        // then
        expect(screen.getByText("Loading room...")).toBeInTheDocument();
    });

    it("offers a way in when the viewer is not a member and tries to join when asked", async () => {
        // given
        const pointer = userEvent.setup();
        const { controller } = renderRoom({ room: null });

        // when
        await pointer.click(screen.getByRole("button", { name: "Try to Join" }));

        // then
        expect(screen.getByText("You're not a member of this room.")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Back to Rooms" })).toBeInTheDocument();
        expect(controller.room.join).toHaveBeenCalledTimes(1);
    });

    it("says it is joining while the request is in flight", () => {
        // given
        const roomState = { joining: true };

        // when
        renderRoom({ room: null, roomState });

        // then
        expect(screen.getByRole("button", { name: "Joining..." })).toBeDisabled();
    });

    it("hides the join button when there is no room in the address at all", () => {
        // given
        const roomState = { id: undefined };

        // when
        renderRoom({ room: null, roomState });

        // then
        expect(screen.queryByRole("button", { name: "Try to Join" })).not.toBeInTheDocument();
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
    const headerCases: { name: string; room: Partial<ChatRoom>; meta: string; badges: string[] }[] = [
        { name: "names the room and describes who can see it", room: {}, meta: "7 members · public", badges: [] },
        {
            name: "marks a private room as private",
            room: { is_public: false },
            meta: "7 members · private",
            badges: [],
        },
        {
            name: "badges a staff room and a roleplay room",
            room: { is_system: true, is_rp: true },
            meta: "7 members · public",
            badges: ["Staff", "RP"],
        },
    ];

    it.each(headerCases)("$name", ({ room, meta, badges }) => {
        // given the room, from the table row

        // when
        renderRoom({ room: makeChatRoom({ name: "Rokkenjima", member_count: 7, is_public: true, ...room }) });

        // then
        expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        expect(screen.getByText(meta)).toBeInTheDocument();
        expect(["Staff", "RP"].filter(badge => screen.queryByText(badge) !== null)).toEqual(badges);
    });

    const infoCases: { name: string; room: Partial<ChatRoom>; toggles: string[]; shows: string[] }[] = [
        {
            name: "leaves the info panel out when there is nothing to say",
            room: { description: "", tags: [] },
            toggles: [],
            shows: [],
        },
        {
            name: "offers the info panel when the room has a description",
            room: { description: "Where the witches take tea" },
            toggles: ["Show info ▼"],
            shows: ["Where the witches take tea"],
        },
        {
            name: "lists the room's tags",
            room: { tags: ["horror", "spoilers"] },
            toggles: ["Show info ▼"],
            shows: ["#horror", "#spoilers"],
        },
    ];

    it.each(infoCases)("$name", ({ room, toggles, shows }) => {
        // given the room's description and tags, from the table row

        // when
        renderRoom({ room: makeChatRoom(room) });

        // then
        expect(screen.queryAllByRole("button", { name: /Show info/ }).map(button => button.textContent)).toEqual(
            toggles,
        );
        for (const text of shows) {
            expect(screen.getByText(text)).toBeInTheDocument();
        }
    });
});

describe("RoomPage members", () => {
    const voiceMembers = [makeMember()];
    const rosterCases: { name: string; setup: RoomSetup; shows: () => HTMLElement; hides?: string[] }[] = [
        {
            name: "counts the members in the sidebar",
            setup: {
                members: [makeMember(), makeMember({ user: makeMemberUser({ id: "member-2", username: "ange" }) })],
            },
            shows: () => screen.getByText("2"),
        },
        {
            name: "groups the members under an online heading when they are around",
            setup: { members: [makeMember()], roster: { onlineWeight: () => 0 } },
            shows: () => screen.getByText("Online"),
        },
        {
            name: "groups the members under an offline heading when they are away",
            setup: { members: [makeMember()], roster: { onlineWeight: () => 1 } },
            shows: () => screen.getByText("Offline"),
        },
        {
            name: "skips the status heading inside the voice group",
            setup: { members: voiceMembers, roster: { groups: [{ label: "In Voice", members: voiceMembers }] } },
            shows: () => screen.getByText("In Voice"),
            hides: ["Offline"],
        },
        {
            name: "says whether a member is watching the room right now",
            setup: { members: [makeMember()], roster: { presence: { "member-1": "active" } } },
            shows: () => screen.getByLabelText("Active in this room"),
        },
        {
            name: "marks a member who is timed out",
            setup: { members: [makeMember({ timeout_until: TIMED_OUT_UNTIL })] },
            shows: () => screen.getByLabelText(`Timed out until until ${TIMED_OUT_UNTIL}`),
        },
    ];

    it.each(rosterCases)("$name", ({ setup, shows, hides = [] }) => {
        // given the roster, from the table row

        // when
        renderRoom(setup);

        // then
        expect(shows()).toBeInTheDocument();
        for (const text of hides) {
            expect(screen.queryByText(text)).not.toBeInTheDocument();
        }
    });
});

describe("RoomPage permissions", () => {
    const permissionCases: {
        name: string;
        viewer: UserProfile;
        room: Partial<ChatRoom>;
        roomControls: string[];
        memberControls: string[];
    }[] = [
        {
            name: "gives the host of an ordinary room its controls and kick, ban and timeout but no nickname change",
            viewer,
            room: { viewer_role: "host" },
            roomControls: ["Moderation", "Delete Room", "+ Invite", "Voice", "Watch party"],
            memberControls: ["Moderator actions", "Kick member", "Ban from room", "Set timeout"],
        },
        {
            name: "offers an ordinary member the door and the call buttons but nothing to moderate",
            viewer,
            room: { viewer_role: "member" },
            roomControls: ["Leave Room", "Voice", "Watch party"],
            memberControls: [],
        },
        {
            name: "keeps every control away from the host of a system room",
            viewer,
            room: { viewer_role: "host", is_system: true },
            roomControls: [],
            memberControls: [],
        },
        {
            name: "gives site staff who do not host the room every control, nickname ones included",
            viewer: staff,
            room: { viewer_role: "member" },
            roomControls: ROOM_CONTROLS,
            memberControls: MEMBER_CONTROLS,
        },
        {
            name: "keeps every control away from site staff in a system room",
            viewer: staff,
            room: { viewer_role: "member", is_system: true },
            roomControls: [],
            memberControls: [],
        },
    ];

    it.each(permissionCases)("$name", ({ viewer: roomViewer, room, roomControls, memberControls }) => {
        // given the viewer and the room, from the table row, with the menu open on a member whose nickname is locked
        const members = [makeMember({ nickname_locked: true })];

        // when
        renderRoom({
            viewer: roomViewer,
            room: makeChatRoom(room),
            members,
            moderation: { openMemberMenu: "member-1" },
        });

        // then
        expect(offeredButtons(ROOM_CONTROLS)).toEqual(roomControls);
        expect(offeredButtons(MEMBER_CONTROLS)).toEqual(memberControls);
    });

    it("says it is deleting while the room is being removed", () => {
        // given
        const moderation = { busy: "delete" };

        // when
        renderRoom({ room: hostRoom, moderation });

        // then
        expect(screen.getByRole("button", { name: "Deleting..." })).toBeDisabled();
    });
});

describe("RoomPage controls", () => {
    const controlCases: {
        name: string;
        setup: RoomSetup;
        button: string;
        handler: (controller: RoomController) => unknown;
        args?: unknown[];
    }[] = [
        {
            name: "opens the message search when asked",
            setup: {},
            button: "Search messages",
            handler: controller => controller.panels.openPanel,
            args: ["search"],
        },
        {
            name: "opens the pinned messages when asked",
            setup: {},
            button: "Pinned messages",
            handler: controller => controller.panels.openPanel,
            args: ["pins"],
        },
        {
            name: "collapses the info panel when it is already expanded",
            setup: { room: makeChatRoom({ description: "Tea" }), prefs: { descExpanded: true } },
            button: "Hide info ▲",
            handler: controller => controller.prefs.toggleDescExpanded,
        },
        {
            name: "lets the viewer edit their own profile in the room",
            setup: { members: [makeMember({ user: makeMemberUser({ id: viewer.id, username: "battler" }) })] },
            button: "Edit profile in this room",
            handler: controller => controller.panels.setEditProfileOpen,
            args: [true],
        },
        {
            name: "opens the moderator menu when it is clicked",
            setup: { room: hostRoom, members: [makeMember()] },
            button: "Moderator actions",
            handler: controller => controller.moderation.setOpenMemberMenu,
            args: [expect.any(Function)],
        },
        {
            name: "kicks a member when the host chooses to",
            setup: hostMenu,
            button: "Kick member",
            handler: controller => controller.moderation.handleKick,
            args: ["member-1"],
        },
        {
            name: "bans a member when the host chooses to",
            setup: hostMenu,
            button: "Ban from room",
            handler: controller => controller.moderation.handleBan,
            args: ["member-1"],
        },
        {
            name: "opens the timeout dialog for a member",
            setup: hostMenu,
            button: "Set timeout",
            handler: controller => controller.moderation.openTimeoutDialog,
            args: [menuTarget],
        },
        {
            name: "clears an existing timeout",
            setup: { ...hostMenu, members: [makeMember({ timeout_until: TIMED_OUT_UNTIL })] },
            button: "Remove timeout",
            handler: controller => controller.moderation.handleClearTimeout,
            args: ["member-1"],
        },
        {
            name: "offers to mute the room's notifications",
            setup: {},
            button: "Mute notifications",
            handler: controller => controller.room.toggleMute,
        },
        {
            name: "offers to unmute a room that is already muted",
            setup: { room: makeChatRoom({ viewer_muted: true }) },
            button: "Unmute notifications",
            handler: controller => controller.room.toggleMute,
        },
        {
            name: "collapses the member sidebar when asked",
            setup: {},
            button: "Hide members",
            handler: controller => controller.prefs.toggleSidebar,
        },
        {
            name: "offers a rail to bring the sidebar back once it is collapsed",
            setup: { prefs: { sidebarCollapsed: true } },
            button: "Show members",
            handler: controller => controller.prefs.toggleSidebar,
        },
        {
            name: "goes back to the rooms list from the sidebar",
            setup: {},
            button: "Back to rooms",
            handler: controller => controller.room.backToRooms,
            args: [],
        },
    ];

    it.each(controlCases)("$name", async ({ setup, button, handler, args }) => {
        // given the page and the control, from the table row
        const pointer = userEvent.setup();
        const { controller } = renderRoom(setup);

        // when
        await pointer.click(screen.getByRole("button", { name: button }));

        // then
        if (args === undefined) {
            expect(handler(controller)).toHaveBeenCalledOnce();
        } else {
            expect(handler(controller)).toHaveBeenCalledExactlyOnceWith(...args);
        }
    });
});

describe("RoomPage voice and watch party", () => {
    it("keeps the voice bar away until the call is connected", () => {
        // given
        const voice = { status: "connecting" as const, room: new Room() };

        // when
        renderRoom({ voice });

        // then
        expect(screen.queryByRole("button", { name: "voice bar" })).not.toBeInTheDocument();
    });

    it("lets a moderator force mute a voice participant in this room and says when it did not take", async () => {
        // given
        mocks.forceMute.mockImplementation(
            (
                _roomId: string | null | undefined,
                _variables: { userId: string; muted: boolean },
                options?: { onError?: (err: unknown) => void },
            ) => options?.onError?.(new Error("LiveKit said no")),
        );
        const pointer = userEvent.setup();
        const { controller } = renderRoom({ room: hostRoom, voice: { status: "connected", room: new Room() } });
        const bar = await screen.findByRole("button", { name: "voice bar" });

        // when
        await pointer.click(bar);

        // then
        expect(bar).toHaveAttribute("data-moderator", "true");
        expect(mocks.forceMute).toHaveBeenCalledWith("room-1", { userId: "u9", muted: true }, expect.anything());
        await waitFor(() => {
            expect(controller.toast.show).toHaveBeenCalledWith("LiveKit said no");
        });
    });

    const watchPartyCases = [
        {
            name: "opens the watch party window for an active session, with voice when the site allows it",
            enabled: true,
            startedBy: viewer.id,
            starter: "true",
        },
        {
            name: "obeys the site voice setting inside a watch party",
            enabled: false,
            startedBy: viewer.id,
            starter: "true",
        },
        {
            name: "knows the viewer did not start somebody else's watch party",
            enabled: true,
            startedBy: "someone-else",
            starter: "false",
        },
    ];

    it.each(watchPartyCases)("$name", async ({ enabled, startedBy, starter }) => {
        // given the site voice setting and who started the party, from the table row
        const activeSession = {
            session: makeWatchPartySession({ started_by: startedBy }),
            embedURL: "",
            hasControl: false,
        };

        // when
        renderRoom({ watchParty: { activeSession }, voice: { enabled } });

        // then
        const modal = await screen.findByTestId("watch-party-modal");
        expect(modal).toHaveAttribute("data-starter", starter);
        expect(modal).toHaveAttribute("data-voice", String(enabled));
    });
});

describe("RoomPage member dialogs", () => {
    it("names the member whose nickname is being changed and saves the new nickname when asked", async () => {
        // given
        const pointer = userEvent.setup();
        const { controller } = renderRoom({ moderation: { nicknameDialogTarget: makeMember() } });

        // when
        await pointer.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(screen.getByRole("heading", { name: "Change nickname for Beatrice" })).toBeInTheDocument();
        expect(controller.moderation.handleModSetNickname).toHaveBeenCalledTimes(1);
    });

    it("names the member being timed out, offers the units and sets the timeout when asked", async () => {
        // given
        const pointer = userEvent.setup();
        const { controller } = renderRoom({ moderation: { timeoutDialogTarget: makeMember() } });

        // when
        await pointer.click(screen.getByRole("button", { name: "Set timeout" }));

        // then
        expect(screen.getByRole("heading", { name: "Set timeout for Beatrice" })).toBeInTheDocument();
        expect(screen.getByRole("option", { name: "centuries" })).toBeInTheDocument();
        expect(controller.moderation.handleSetTimeout).toHaveBeenCalledTimes(1);
    });
});

describe("RoomPage overlays", () => {
    const overlayCases: { name: string; panels: Partial<RoomController["panels"]>; open: string[] }[] = [
        { name: "keeps every panel, dialog and the lightbox shut until one is opened", panels: {}, open: [] },
        { name: "shows the search panel once it is open", panels: { panelTab: "search" }, open: ["search-panel"] },
        {
            name: "shows the pinned messages panel only once it is opened",
            panels: { panelTab: "pins" },
            open: ["pins-panel"],
        },
        {
            name: "shows the invite modal only once it is opened",
            panels: { inviteModalOpen: true },
            open: ["invite-modal"],
        },
        {
            name: "shows the moderation dialog only once it is opened",
            panels: { moderationDialogOpen: true },
            open: ["moderation-dialog"],
        },
    ];

    it.each(overlayCases)("$name", ({ panels, open }) => {
        // given the open panels, from the table row

        // when
        renderRoom({ panels });

        // then
        expect(OVERLAYS.filter(testId => screen.queryByTestId(testId) !== null)).toEqual(open);
        expect(screen.queryByRole("heading")).not.toBeInTheDocument();
    });

    it("shows the image the viewer opened", () => {
        // given
        const panels = { lightboxSrc: "/media/witch.png" };

        // when
        renderRoom({ panels });

        // then
        expect(screen.getByTestId("lightbox")).toHaveTextContent("/media/witch.png");
    });
});

describe("RoomPage notices", () => {
    const noticeCases: { name: string; setup: RoomSetup; text: string }[] = [
        { name: "shows a passing toast", setup: { toast: "1 member invited" }, text: "1 member invited" },
        {
            name: "passes on why the join failed",
            setup: { room: null, toast: "You are banned from this room." },
            text: "You are banned from this room.",
        },
        {
            name: "says so when the invited watch party has already ended",
            setup: { watchParty: { invitedPartyMissing: true } },
            text: "That watch party has ended.",
        },
        {
            name: "reports why a nickname could not be saved",
            setup: { moderation: { nicknameDialogTarget: makeMember(), nicknameDialogError: "That name is taken." } },
            text: "That name is taken.",
        },
    ];

    it.each(noticeCases)("$name", ({ setup, text }) => {
        // given the notice, from the table row

        // when
        renderRoom(setup);

        // then
        expect(screen.getByText(text)).toBeInTheDocument();
    });

    it("says who is typing", () => {
        // given
        const session = { typingNames: ["Beatrice", "Ange"] };

        // when
        renderRoom({ session });

        // then
        expect(screen.getByText(/are typing/)).toHaveTextContent("Beatrice and Ange are typing...");
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
