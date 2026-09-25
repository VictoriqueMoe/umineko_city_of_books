import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createContext, type ReactNode, type RefObject } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeWatchPartySession } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { User, WatchPartyParticipant, WatchPartySession } from "../../../types/api";
import type { ActiveWatchPartySession } from "../../../hooks/useWatchParty";
import { WatchPartyModal } from "./WatchPartyModal";

const mocks = vi.hoisted(() => ({
    hyperbeam: vi.fn(),
    useSessionMedia: vi.fn(),
    forceMute: vi.fn(),
    useIsMobile: vi.fn(),
    useParticipants: vi.fn(),
}));

vi.mock("@hyperbeam/web", () => ({ default: mocks.hyperbeam }));

vi.mock("@livekit/components-react", () => ({
    RoomContext: createContext<unknown>(null),
    RoomAudioRenderer: () => <div data-testid="room-audio" />,
    useParticipants: mocks.useParticipants,
}));

vi.mock("../../../hooks/useIsMobile", () => ({ useIsMobile: mocks.useIsMobile }));

interface MobileViewProps {
    title: string;
    canEnd: boolean;
    copied: boolean;
    watcherCount: number;
    voiceCount: number;
    onCopyInvite: () => void;
    onHide: () => void;
    onLeave: () => void;
    onEnd: () => void;
    stageRef: RefObject<HTMLDivElement | null>;
    stage: ReactNode;
    chat: ReactNode;
    voice: ReactNode;
    people: ReactNode;
}

vi.mock("./WatchPartyMobileView", () => ({
    WatchPartyMobileView: (props: MobileViewProps) => (
        <div
            data-testid="mobile-view"
            data-watchers={String(props.watcherCount)}
            data-voices={String(props.voiceCount)}
        >
            <span>{props.title}</span>
            <button type="button" onClick={props.onCopyInvite}>
                {props.copied ? "Phone menu: Link copied" : "Phone menu: Copy invite"}
            </button>
            <button type="button" onClick={props.onHide}>
                Phone menu: Hide
            </button>
            <button type="button" onClick={props.onLeave}>
                Phone menu: Leave
            </button>
            {props.canEnd && (
                <button type="button" onClick={props.onEnd}>
                    Phone menu: End for everyone
                </button>
            )}
            <div data-testid="mobile-stage" ref={props.stageRef}>
                {props.stage}
            </div>
            <div data-testid="mobile-chat">{props.chat}</div>
            <div data-testid="mobile-voice">{props.voice}</div>
            <div data-testid="mobile-people">{props.people}</div>
        </div>
    ),
}));

vi.mock("../Voice/VoiceParticipants", () => ({
    VoiceParticipantList: ({
        canModerate,
        onForceMute,
    }: {
        canModerate: boolean;
        onForceMute: (identity: string, muted: boolean) => void;
    }) => (
        <div data-testid="voice-participants" data-can-moderate={String(canModerate)}>
            <button type="button" onClick={() => onForceMute("battler", true)}>
                force mute
            </button>
        </div>
    ),
}));

vi.mock("./ScreenShareView", () => ({
    ScreenShareView: ({ placeholder, onReload }: { placeholder: string; onReload?: () => void }) => (
        <div data-testid="screen-share-view">
            <span>{placeholder}</span>
            <button type="button" onClick={onReload}>
                reload share
            </button>
        </div>
    ),
}));

vi.mock("./useAudioPlaybackGuard", () => ({ useAudioPlaybackGuard: vi.fn() }));

vi.mock("../../../hooks/useSessionMedia", () => ({ useSessionMedia: mocks.useSessionMedia }));

vi.mock("../../../api/endpoints/watchParty", () => ({
    forceMuteWatchPartyVoiceParticipant: mocks.forceMute,
    endWatchParty: vi.fn(),
    getWatchPartyVoiceToken: vi.fn(),
    identifyWatchPartyParticipant: vi.fn(),
    joinWatchParty: vi.fn(),
    kickWatchPartyParticipant: vi.fn(),
    leaveWatchParty: vi.fn(),
    startWatchParty: vi.fn(),
    transferWatchPartyControl: vi.fn(),
}));

vi.mock("../RoomChatPanel/RoomChatPanel", () => ({
    RoomChatPanel: ({
        roomId,
        title,
        flush,
        hideHeader,
    }: {
        roomId?: string;
        title: string;
        flush?: boolean;
        hideHeader?: boolean;
    }) => (
        <div
            data-testid="room-chat-panel"
            data-room-id={roomId}
            data-flush={String(Boolean(flush))}
            data-hide-header={String(Boolean(hideHeader))}
        >
            {title}
        </div>
    ),
}));

interface NodeProcess {
    on(event: "unhandledRejection", handler: (reason: unknown) => void): void;
    off(event: "unhandledRejection", handler: (reason: unknown) => void): void;
}

const nodeProcess = (globalThis as unknown as { process: NodeProcess }).process;

const viewerId = "user-viewer";

function makeChatUser(overrides: Partial<User> = {}): User {
    return { id: viewerId, username: "beatrice", display_name: "Beatrice", ...overrides };
}

function makeParticipant(overrides: Partial<WatchPartyParticipant> = {}): WatchPartyParticipant {
    return { user: makeChatUser(), has_control: false, joined_at: "2026-08-01T10:00:00Z", ...overrides };
}

function makeSession(overrides: Partial<WatchPartySession> = {}): WatchPartySession {
    return makeWatchPartySession({
        started_by: viewerId,
        controller_id: viewerId,
        participants: [makeParticipant()],
        ...overrides,
    });
}

function makeActive(overrides: Partial<ActiveWatchPartySession> = {}): ActiveWatchPartySession {
    return {
        session: makeSession(),
        embedURL: "https://hb.test/embed",
        hasControl: false,
        ...overrides,
    };
}

const screenShareParty = makeActive({ session: makeSession({ type: "screenshare" }), embedURL: "" });

const twoWatchers = makeSession({
    participants: [
        makeParticipant(),
        makeParticipant({ user: makeChatUser({ id: "user-battler", display_name: "Battler" }) }),
    ],
});

const layouts = [
    { name: "desktop", mobile: false, menu: "" },
    { name: "phone", mobile: true, menu: "Phone menu: " },
];

const roles = [
    { role: "an ordinary watcher", isStarter: false, viewerIsStaff: false, trusted: false },
    { role: "the host", isStarter: true, viewerIsStaff: false, trusted: true },
    { role: "site staff who did not start it", isStarter: false, viewerIsStaff: true, trusted: true },
];

const roleCases = layouts.flatMap(layout => roles.map(role => ({ ...layout, ...role })));

interface MediaState {
    room?: unknown;
    status?: "idle" | "connecting" | "connected";
    inVoice?: boolean;
    isSharing?: boolean;
    shareError?: string | null;
}

function stubMedia(state: MediaState = {}) {
    const media = {
        room: state.room ?? null,
        status: state.status ?? "idle",
        inVoice: state.inVoice ?? false,
        isSharing: state.isSharing ?? false,
        shareError: state.shareError ?? null,
        joinVoice: vi.fn(() => Promise.resolve()),
        leaveVoice: vi.fn(() => Promise.resolve()),
        shareScreen: vi.fn(() => Promise.resolve()),
        reload: vi.fn(() => Promise.resolve()),
    };
    mocks.useSessionMedia.mockReturnValue(media);

    return media;
}

function stubFullscreen() {
    const requestFullscreen = vi.fn(() => Promise.resolve());
    Object.defineProperty(Element.prototype, "requestFullscreen", {
        configurable: true,
        writable: true,
        value: requestFullscreen,
    });

    return requestFullscreen;
}

function paneOf(mobile: boolean, testId: string) {
    if (!mobile) {
        return screen;
    }

    return within(screen.getByTestId(testId));
}

interface ModalOptions {
    isOpen?: boolean;
    mobile?: boolean;
    active?: ActiveWatchPartySession;
    isStarter?: boolean;
    viewerIsStaff?: boolean;
    voiceEnabled?: boolean;
    onLeave?: () => Promise<void>;
    onEnd?: () => Promise<void>;
}

function renderModal(options: ModalOptions = {}) {
    mocks.useIsMobile.mockReturnValue(options.mobile ?? false);

    const callbacks = {
        onClose: vi.fn(),
        onLeave: vi.fn(options.onLeave ?? (() => Promise.resolve())),
        onEnd: vi.fn(options.onEnd ?? (() => Promise.resolve())),
        onTransferControl: vi.fn(() => Promise.resolve()),
        onKick: vi.fn(() => Promise.resolve()),
        onIdentify: vi.fn(() => Promise.resolve()),
    };

    const modalWith = (active: ActiveWatchPartySession) => (
        <WatchPartyModal
            isOpen={options.isOpen ?? true}
            active={active}
            viewerUserId={viewerId}
            viewerRole={undefined}
            isStarter={options.isStarter ?? false}
            viewerIsStaff={options.viewerIsStaff ?? false}
            voiceEnabled={options.voiceEnabled ?? true}
            {...callbacks}
        />
    );

    const result = renderWithProviders(modalWith(options.active ?? makeActive()));

    return { ...result, ...callbacks, modalWith };
}

function makeHandle(userId = "hb-user-1") {
    return { destroy: vi.fn(), disableInput: true, userId };
}

beforeEach(() => {
    stubMedia();
    mocks.useParticipants.mockReturnValue([]);
    mocks.hyperbeam.mockResolvedValue(makeHandle());
    mocks.forceMute.mockResolvedValue(undefined);
    vi.spyOn(console, "error").mockImplementation(() => {});
});

describe("WatchPartyModal shell", () => {
    it("renders nothing at all while it is closed", () => {
        // given
        const options = { isOpen: false };

        // when
        renderModal(options);

        // then
        expect(screen.queryByText("Watch party")).not.toBeInTheDocument();
        expect(mocks.hyperbeam).not.toHaveBeenCalled();
    });

    it.each([
        { name: "a titled party on the desktop", mobile: false, title: "Chiru rewatch", shown: "Chiru rewatch" },
        { name: "an untitled party on the desktop", mobile: false, title: "", shown: "Untitled party" },
        { name: "a titled party on the phone", mobile: true, title: "Chiru rewatch", shown: "Chiru rewatch" },
        { name: "an untitled party on the phone", mobile: true, title: "", shown: "Untitled party" },
    ])("names $name", ({ mobile, title, shown }) => {
        // given
        const active = makeActive({ session: makeSession({ title }) });

        // when
        renderModal({ active, mobile });

        // then
        expect(screen.getByText(shown)).toBeInTheDocument();
    });

    it.each(layouts)("copies an invite that points straight at this party on the $name", async ({ mobile, menu }) => {
        // given
        const user = userEvent.setup();
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        renderModal({ mobile });

        // when
        await user.click(screen.getByRole("button", { name: `${menu}Copy invite` }));

        // then
        expect(writeText).toHaveBeenCalledWith("https://whentheycry.social/rooms/room-1?party=session-1");
        expect(await screen.findByRole("button", { name: `${menu}Link copied` })).toBeInTheDocument();
    });

    it("puts the invite control back two seconds after a copy", async () => {
        // given
        vi.useFakeTimers({ shouldAdvanceTime: true });
        const user = userEvent.setup();
        vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        renderModal();
        await user.click(screen.getByRole("button", { name: "Copy invite" }));
        await screen.findByRole("button", { name: "Link copied" });

        // when
        act(() => {
            vi.advanceTimersByTime(2000);
        });

        // then
        expect(screen.getByRole("button", { name: "Copy invite" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Link copied" })).not.toBeInTheDocument();
    });

    it.each(layouts)("hides the window on the $name without leaving the party", async ({ mobile, menu }) => {
        // given
        const user = userEvent.setup();
        const { onClose, onLeave } = renderModal({ mobile });

        // when
        await user.click(screen.getByRole("button", { name: `${menu}Hide` }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
        expect(onLeave).not.toHaveBeenCalled();
    });

    it.each(layouts)("leaves the party on the $name and then closes the window", async ({ mobile, menu }) => {
        // given
        const user = userEvent.setup();
        const { onClose, onLeave } = renderModal({ mobile });

        // when
        await user.click(screen.getByRole("button", { name: `${menu}Leave` }));

        // then
        expect(onLeave).toHaveBeenCalledOnce();
        await waitFor(() => {
            expect(onClose).toHaveBeenCalledOnce();
        });
    });

    it.each(layouts)("lets the host end the party for everyone on the $name", async ({ mobile, menu }) => {
        // given
        const user = userEvent.setup();
        const { onEnd, onClose } = renderModal({ mobile, isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: `${menu}End for everyone` }));

        // then
        expect(onEnd).toHaveBeenCalledOnce();
        await waitFor(() => {
            expect(onClose).toHaveBeenCalledOnce();
        });
    });

    it.each(roleCases)(
        "$role on the $name may end the party and moderate voice: $trusted",
        ({ mobile, menu, isStarter, viewerIsStaff, trusted }) => {
            // given
            stubMedia({ room: {} });

            // when
            renderModal({ mobile, isStarter, viewerIsStaff });

            // then
            const voicePane = paneOf(mobile, "mobile-voice");
            expect(screen.queryAllByRole("button", { name: `${menu}End for everyone` })).toHaveLength(trusted ? 1 : 0);
            expect(voicePane.getByTestId("voice-participants")).toHaveAttribute("data-can-moderate", String(trusted));
            expect(voicePane.getByRole("button", { name: "Join voice" })).toBeInTheDocument();
        },
    );

    it.each([
        { name: "leave", button: "Leave", options: { onLeave: () => Promise.reject(new Error("network is down")) } },
        {
            name: "end",
            button: "End for everyone",
            options: { isStarter: true, onEnd: () => Promise.reject(new Error("only the host may end it")) },
        },
    ])("swallows a refused $name instead of leaving the rejection unhandled", async ({ button, options }) => {
        // given
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        nodeProcess.on("unhandledRejection", record);
        const user = userEvent.setup();
        const { onClose } = renderModal(options);

        // when
        await user.click(screen.getByRole("button", { name: button }));
        await new Promise(resolve => setTimeout(resolve, 0));
        nodeProcess.off("unhandledRejection", record);

        // then
        expect(unhandled).toEqual([]);
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: button })).toBeEnabled();
    });
});

describe("WatchPartyModal virtual browser", () => {
    it.each([
        { name: "a passenger", hasControl: false, disableInput: true, badges: 0 },
        { name: "the controller", hasControl: true, disableInput: false, badges: 1 },
    ])(
        "mounts the virtual browser for $name, with input and badge matching their control, and reports their seat",
        async ({ hasControl, disableInput, badges }) => {
            // given
            mocks.hyperbeam.mockResolvedValue(makeHandle("hb-user-7"));
            const active = makeActive({ embedURL: "https://hb.test/embed-9", hasControl });

            // when
            const { onIdentify } = renderModal({ active });

            // then
            await waitFor(() => {
                expect(onIdentify).toHaveBeenCalledWith("hb-user-7");
            });
            expect(mocks.hyperbeam).toHaveBeenCalledOnce();
            expect(mocks.hyperbeam.mock.calls[0][1]).toBe("https://hb.test/embed-9");
            expect(mocks.hyperbeam.mock.calls[0][2]).toEqual({ delegateKeyboard: true, disableInput });
            expect(screen.queryAllByText("Controller")).toHaveLength(badges);
        },
    );

    it("unlocks the virtual browser when control is handed to the viewer", async () => {
        // given
        const handle = makeHandle();
        mocks.hyperbeam.mockResolvedValue(handle);
        const { onIdentify, rerender, modalWith } = renderModal({ active: makeActive({ hasControl: false }) });
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalled();
        });

        // when
        rerender(modalWith(makeActive({ hasControl: true })));

        // then
        expect(handle.disableInput).toBe(false);
    });

    it("explains a virtual browser that refused to connect", async () => {
        // given
        mocks.hyperbeam.mockRejectedValue(new Error("the VM has expired"));

        // when
        renderModal();

        // then
        expect(await screen.findByText("Virtual browser failed to connect")).toBeInTheDocument();
        expect(screen.getByText("the VM has expired")).toBeInTheDocument();
        expect(screen.getByText(/Try ending this party and starting a fresh one/)).toBeInTheDocument();
    });

    it("waits without mounting anything while the embed url is still missing", () => {
        // given
        const active = makeActive({ embedURL: "" });

        // when
        renderModal({ active });

        // then
        expect(screen.getByText("Loading virtual browser...")).toBeInTheDocument();
        expect(mocks.hyperbeam).not.toHaveBeenCalled();
    });
});

describe("WatchPartyModal screen share", () => {
    it("says it is still connecting until livekit is up", () => {
        // given
        stubMedia({ room: null });

        // when
        renderModal({ active: screenShareParty });

        // then
        expect(screen.getByText("Connecting...")).toBeInTheDocument();
        expect(screen.queryByTestId("screen-share-view")).not.toBeInTheDocument();
    });

    it.each([
        {
            name: "a watcher they are waiting on the host",
            isStarter: false,
            shown: "Waiting for the host to share their screen.",
        },
        { name: "the host they can start sharing", isStarter: true, shown: "Click Share screen to start sharing." },
    ])("tells $name, and never mounts a virtual browser", ({ isStarter, shown }) => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ active: screenShareParty, isStarter });

        // then
        expect(screen.getByText(shown)).toBeInTheDocument();
        expect(mocks.hyperbeam).not.toHaveBeenCalled();
    });

    it("rebuilds the stream when a watcher asks for a reload", async () => {
        // given
        const user = userEvent.setup();
        const media = stubMedia({ room: {} });
        renderModal({ active: screenShareParty });

        // when
        await user.click(screen.getByRole("button", { name: "reload share" }));

        // then
        expect(media.reload).toHaveBeenCalledOnce();
    });

    it("asks the browser for fullscreen on the media panel", async () => {
        // given
        const user = userEvent.setup();
        const requestFullscreen = stubFullscreen();
        stubMedia({ room: {} });
        renderModal({ active: screenShareParty });

        // when
        await user.click(screen.getByRole("button", { name: "Fullscreen" }));

        // then
        expect(requestFullscreen).toHaveBeenCalledOnce();
    });
});

describe("WatchPartyModal voice", () => {
    it("hides the voice strip entirely when the site has voice switched off", () => {
        // given
        const options = { voiceEnabled: false };

        // when
        renderModal(options);

        // then
        expect(screen.queryByRole("button", { name: "Join voice" })).not.toBeInTheDocument();
    });

    it.each(layouts)("lets a watcher join the voice channel on the $name", async ({ mobile }) => {
        // given
        const user = userEvent.setup();
        const media = stubMedia();
        renderModal({ mobile });

        // when
        await user.click(paneOf(mobile, "mobile-voice").getByRole("button", { name: "Join voice" }));

        // then
        expect(media.joinVoice).toHaveBeenCalledOnce();
    });

    it("holds the join control shut while the connection is being made", () => {
        // given
        stubMedia({ status: "connecting" });

        // when
        renderModal();

        // then
        expect(screen.getByRole("button", { name: "Join voice" })).toBeDisabled();
    });

    it("offers to leave voice once the viewer is in it", async () => {
        // given
        const user = userEvent.setup();
        const media = stubMedia({ inVoice: true });
        renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "Leave voice" }));

        // then
        expect(media.leaveVoice).toHaveBeenCalledOnce();
        expect(screen.queryByRole("button", { name: "Join voice" })).not.toBeInTheDocument();
    });

    it("shows the voice roster only once a room has been connected", () => {
        // given
        stubMedia({ room: null });

        // when
        renderModal();

        // then
        expect(screen.queryByTestId("voice-participants")).not.toBeInTheDocument();
    });

    it("forwards a forced mute to the party voice endpoint and says nothing once it has worked", async () => {
        // given
        const user = userEvent.setup();
        stubMedia({ room: {} });
        renderModal({ isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "force mute" }));

        // then
        await waitFor(() => {
            expect(mocks.forceMute).toHaveBeenCalledWith("room-1", "session-1", "battler", true);
        });
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it.each([
        {
            name: "with the reason it was given",
            reason: "that watcher outranks you",
            shown: "that watcher outranks you",
        },
        { name: "with a plain message when it carries none", reason: "", shown: "Could not change that microphone." },
    ])("tells the moderator when a forced mute was refused, $name", async ({ reason, shown }) => {
        // given
        const user = userEvent.setup();
        mocks.forceMute.mockRejectedValue(new Error(reason));
        stubMedia({ room: {} });
        renderModal({ isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "force mute" }));

        // then
        expect(await screen.findByRole("alert")).toHaveTextContent(shown);
    });
});

describe("WatchPartyModal sharing controls", () => {
    it.each([
        { name: "anybody but the host", active: screenShareParty, isStarter: false },
        { name: "the host of a virtual browser party", active: makeActive(), isStarter: true },
    ])("keeps the share controls away from $name", ({ active, isStarter }) => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ active, isStarter });

        // then
        expect(screen.queryByRole("button", { name: "Share screen" })).not.toBeInTheDocument();
    });

    it.each([
        { name: "the gaming preset by default", modes: [], preset: "gaming" },
        { name: "the screenshare preset once that mode is chosen", modes: ["Screenshare"], preset: "screenshare" },
    ])("shares with $name", async ({ modes, preset }) => {
        // given
        const user = userEvent.setup();
        const media = stubMedia({ room: {} });
        renderModal({ active: screenShareParty, isStarter: true });

        // when
        for (const mode of modes) {
            await user.click(screen.getByRole("button", { name: mode }));
        }

        await user.click(screen.getByRole("button", { name: "Share screen" }));

        // then
        expect(media.shareScreen).toHaveBeenCalledWith(true, preset);
    });

    it("offers to stop a share that is already running", async () => {
        // given
        const user = userEvent.setup();
        const media = stubMedia({ room: {}, isSharing: true });
        renderModal({ active: screenShareParty, isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "Stop sharing" }));

        // then
        expect(media.shareScreen).toHaveBeenCalledWith(false, "gaming");
        expect(screen.queryByRole("group", { name: "Stream mode" })).not.toBeInTheDocument();
    });

    it("keeps claiming nothing is shared while the hook says the share never took, and tells the host why", () => {
        // given
        stubMedia({ room: {}, isSharing: false, shareError: "Could not start audio source" });

        // when
        renderModal({ active: screenShareParty, isStarter: true });

        // then
        expect(screen.getByRole("alert")).toHaveTextContent("Could not start audio source");
        expect(screen.queryByRole("button", { name: "Stop sharing" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Share screen" })).toBeInTheDocument();
    });

    it("keeps quiet about a picker the host simply closed", async () => {
        // given
        const user = userEvent.setup();
        stubMedia({ room: {} });
        renderModal({ active: screenShareParty, isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "Share screen" }));

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("keeps the host's share controls and share errors reachable on the desktop with voice switched off", () => {
        // given
        stubMedia({ room: {}, shareError: "Could not start audio source" });

        // when
        renderModal({ active: screenShareParty, isStarter: true, voiceEnabled: false });

        // then
        expect(screen.getByRole("button", { name: "Share screen" })).toBeInTheDocument();
        expect(screen.getByRole("alert")).toHaveTextContent("Could not start audio source");
        expect(screen.queryByRole("button", { name: "Join voice" })).not.toBeInTheDocument();
    });
});

describe("WatchPartyModal panels", () => {
    it("hands the party media hook everything it needs about the session", () => {
        // given
        const active = makeActive({ session: makeSession({ type: "screenshare" }) });

        // when
        renderModal({ active, isStarter: true });

        // then
        expect(mocks.useSessionMedia).toHaveBeenCalledWith({
            roomId: "room-1",
            sessionId: "session-1",
            type: "screenshare",
            isStarter: true,
        });
    });

    it.each([
        { name: "alongside the media on the desktop", mobile: false, hideHeader: "false", flush: "false" },
        {
            name: "in a flush pane with no header of its own on the phone",
            mobile: true,
            hideHeader: "true",
            flush: "true",
        },
    ])("shows the party chat $name, scoped to the session's own room", ({ mobile, hideHeader, flush }) => {
        // given
        const active = makeActive({ session: makeSession({ id: "session-42" }) });

        // when
        renderModal({ active, mobile });

        // then
        const panel = paneOf(mobile, "mobile-chat").getByTestId("room-chat-panel");
        expect(panel).toHaveTextContent("Party chat");
        expect(panel).toHaveAttribute("data-room-id", "session-42");
        expect(panel).toHaveAttribute("data-hide-header", hideHeader);
        expect(panel).toHaveAttribute("data-flush", flush);
    });

    it.each(layouts)("lists the watchers of the party on the $name", ({ mobile }) => {
        // given
        const active = makeActive({ session: twoWatchers });

        // when
        renderModal({ active, mobile });

        // then
        const pane = paneOf(mobile, "mobile-people");
        expect(pane.getByText("2 watchers")).toBeInTheDocument();
        expect(pane.getByText("Battler")).toBeInTheDocument();
    });
});

describe("WatchPartyModal on a phone", () => {
    it("keeps the full sized header actions off the phone", () => {
        // given
        const options = { mobile: true, isStarter: true };

        // when
        renderModal(options);

        // then
        expect(screen.queryByRole("button", { name: "Copy invite" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Hide" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Leave" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "End for everyone" })).not.toBeInTheDocument();
    });

    it("keeps the host's share controls reachable even with voice switched off", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ mobile: true, active: screenShareParty, isStarter: true, voiceEnabled: false });

        // then
        const pane = within(screen.getByTestId("mobile-voice"));
        expect(pane.getByRole("button", { name: "Share screen" })).toBeInTheDocument();
        expect(pane.queryByRole("button", { name: "Join voice" })).not.toBeInTheDocument();
    });

    it.each([
        { name: "on", voiceEnabled: true, voices: "2", rosters: 1 },
        { name: "switched off", voiceEnabled: false, voices: "0", rosters: 0 },
    ])(
        "counts the watchers, and who is in voice, for the phone's tabs with voice $name",
        ({ voiceEnabled, voices, rosters }) => {
            // given
            stubMedia({ room: {} });
            mocks.useParticipants.mockReturnValue([{ identity: "beatrice" }, { identity: "battler" }]);

            // when
            renderModal({ mobile: true, active: makeActive({ session: twoWatchers }), voiceEnabled });

            // then
            const view = screen.getByTestId("mobile-view");
            expect(view).toHaveAttribute("data-watchers", "2");
            expect(view).toHaveAttribute("data-voices", voices);
            expect(within(screen.getByTestId("mobile-voice")).queryAllByTestId("voice-participants")).toHaveLength(
                rosters,
            );
        },
    );

    it("keeps the party audio playing from the stage while another pane is open", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ mobile: true });

        // then
        expect(within(screen.getByTestId("mobile-stage")).getByTestId("room-audio")).toBeInTheDocument();
    });

    it("fullscreens the whole stage from the control sitting on the picture", async () => {
        // given
        const user = userEvent.setup();
        const requestFullscreen = stubFullscreen();
        stubMedia({ room: {} });
        renderModal({ mobile: true, active: screenShareParty });

        // when
        const stage = within(screen.getByTestId("mobile-stage"));
        await user.click(stage.getByRole("button", { name: "Fullscreen" }));

        // then
        expect(stage.getByTestId("screen-share-view")).toBeInTheDocument();
        expect(requestFullscreen).toHaveBeenCalledOnce();
        expect(requestFullscreen.mock.contexts[0]).toBe(screen.getByTestId("mobile-stage"));
    });
});
