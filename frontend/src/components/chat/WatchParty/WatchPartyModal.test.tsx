import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createContext, type ReactNode, type RefObject } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeWatchPartySession } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { SiteRole, User, WatchPartyParticipant, WatchPartySession } from "../../../types/api";
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
    hasControl: boolean;
    canEnd: boolean;
    busy: boolean;
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
            data-title={props.title}
            data-watchers={String(props.watcherCount)}
            data-voices={String(props.voiceCount)}
            data-can-end={String(props.canEnd)}
            data-control={String(props.hasControl)}
            data-copied={String(props.copied)}
        >
            <button type="button" onClick={props.onCopyInvite}>
                overflow copy invite
            </button>
            <button type="button" onClick={props.onHide}>
                overflow hide
            </button>
            <button type="button" onClick={props.onLeave}>
                overflow leave
            </button>
            <button type="button" onClick={props.onEnd}>
                overflow end
            </button>
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

interface ModalOptions {
    isOpen?: boolean;
    active?: ActiveWatchPartySession;
    viewerRole?: SiteRole | undefined;
    isStarter?: boolean;
    viewerIsStaff?: boolean;
    voiceEnabled?: boolean;
    onLeave?: () => Promise<void>;
    onEnd?: () => Promise<void>;
}

function renderModal(options: ModalOptions = {}) {
    const onClose = vi.fn();
    const onLeave = vi.fn(options.onLeave ?? (() => Promise.resolve()));
    const onEnd = vi.fn(options.onEnd ?? (() => Promise.resolve()));
    const onTransferControl = vi.fn(() => Promise.resolve());
    const onKick = vi.fn(() => Promise.resolve());
    const onIdentify = vi.fn(() => Promise.resolve());

    const result = renderWithProviders(
        <WatchPartyModal
            isOpen={options.isOpen ?? true}
            onClose={onClose}
            active={options.active ?? makeActive()}
            viewerUserId={viewerId}
            viewerRole={options.viewerRole}
            isStarter={options.isStarter ?? false}
            viewerIsStaff={options.viewerIsStaff ?? false}
            voiceEnabled={options.voiceEnabled ?? true}
            onLeave={onLeave}
            onEnd={onEnd}
            onTransferControl={onTransferControl}
            onKick={onKick}
            onIdentify={onIdentify}
        />,
    );

    return { ...result, onClose, onLeave, onEnd, onTransferControl, onKick, onIdentify };
}

function makeHandle(userId = "hb-user-1") {
    return { destroy: vi.fn(), disableInput: true, userId };
}

beforeEach(() => {
    stubMedia();
    mocks.useIsMobile.mockReturnValue(false);
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

    it("names the party in its header", () => {
        // given
        const active = makeActive({ session: makeSession({ title: "Chiru rewatch" }) });

        // when
        renderModal({ active });

        // then
        expect(screen.getByText("Chiru rewatch")).toBeInTheDocument();
    });

    it("falls back to a plain name for a party that was never titled", () => {
        // given
        const active = makeActive({ session: makeSession({ title: "" }) });

        // when
        renderModal({ active });

        // then
        expect(screen.getByText("Untitled party")).toBeInTheDocument();
    });

    it("badges the viewer who is driving the virtual browser", () => {
        // given
        const active = makeActive({ hasControl: true });

        // when
        renderModal({ active });

        // then
        expect(screen.getByText("Controller")).toBeInTheDocument();
    });

    it("leaves the badge off a viewer who is only watching", () => {
        // given
        const active = makeActive({ hasControl: false });

        // when
        renderModal({ active });

        // then
        expect(screen.queryByText("Controller")).not.toBeInTheDocument();
    });

    it("copies an invite that points straight at this party", async () => {
        // given
        const user = userEvent.setup();
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "Copy invite" }));

        // then
        expect(writeText).toHaveBeenCalledWith("https://whentheycry.social/rooms/room-1?party=session-1");
        expect(await screen.findByRole("button", { name: "Link copied" })).toBeInTheDocument();
    });

    it("hides the window without leaving the party", async () => {
        // given
        const user = userEvent.setup();
        const { onClose, onLeave } = renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "Hide" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
        expect(onLeave).not.toHaveBeenCalled();
    });

    it("leaves the party and then closes the window", async () => {
        // given
        const user = userEvent.setup();
        const { onClose, onLeave } = renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "Leave" }));

        // then
        expect(onLeave).toHaveBeenCalledOnce();
        await waitFor(() => {
            expect(onClose).toHaveBeenCalledOnce();
        });
    });

    it("keeps the end control away from an ordinary watcher", () => {
        // given
        const options = { isStarter: false, viewerIsStaff: false };

        // when
        renderModal(options);

        // then
        expect(screen.queryByRole("button", { name: "End for everyone" })).not.toBeInTheDocument();
    });

    it("lets the host end the party for everyone", async () => {
        // given
        const user = userEvent.setup();
        const { onEnd, onClose } = renderModal({ isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "End for everyone" }));

        // then
        expect(onEnd).toHaveBeenCalledOnce();
        await waitFor(() => {
            expect(onClose).toHaveBeenCalledOnce();
        });
    });

    it("swallows a refused leave instead of leaving the rejection unhandled", async () => {
        // given
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        nodeProcess.on("unhandledRejection", record);
        const user = userEvent.setup();
        const { onClose } = renderModal({ onLeave: () => Promise.reject(new Error("network is down")) });

        // when
        await user.click(screen.getByRole("button", { name: "Leave" }));
        await new Promise(resolve => setTimeout(resolve, 0));
        nodeProcess.off("unhandledRejection", record);

        // then
        expect(unhandled).toEqual([]);
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Leave" })).toBeEnabled();
    });

    it("swallows a refused end instead of leaving the rejection unhandled", async () => {
        // given
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        nodeProcess.on("unhandledRejection", record);
        const user = userEvent.setup();
        const { onClose } = renderModal({
            isStarter: true,
            onEnd: () => Promise.reject(new Error("only the host may end it")),
        });

        // when
        await user.click(screen.getByRole("button", { name: "End for everyone" }));
        await new Promise(resolve => setTimeout(resolve, 0));
        nodeProcess.off("unhandledRejection", record);

        // then
        expect(unhandled).toEqual([]);
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "End for everyone" })).toBeEnabled();
    });

    it("lets site staff end a party they did not start", () => {
        // given
        const options = { isStarter: false, viewerIsStaff: true };

        // when
        renderModal(options);

        // then
        expect(screen.getByRole("button", { name: "End for everyone" })).toBeInTheDocument();
    });
});

describe("WatchPartyModal virtual browser", () => {
    it("mounts the virtual browser at the embed url with input locked for a passenger", async () => {
        // given
        const active = makeActive({ embedURL: "https://hb.test/embed-9", hasControl: false });

        // when
        renderModal({ active });

        // then
        await waitFor(() => {
            expect(mocks.hyperbeam).toHaveBeenCalledOnce();
        });
        expect(mocks.hyperbeam.mock.calls[0][1]).toBe("https://hb.test/embed-9");
        expect(mocks.hyperbeam.mock.calls[0][2]).toEqual({ delegateKeyboard: true, disableInput: true });
    });

    it("unlocks input straight away for the viewer who holds control", async () => {
        // given
        const active = makeActive({ hasControl: true });

        // when
        renderModal({ active });

        // then
        await waitFor(() => {
            expect(mocks.hyperbeam).toHaveBeenCalledOnce();
        });
        expect(mocks.hyperbeam.mock.calls[0][2]).toEqual({ delegateKeyboard: true, disableInput: false });
    });

    it("tells the server which seat the virtual browser handed the viewer", async () => {
        // given
        mocks.hyperbeam.mockResolvedValue(makeHandle("hb-user-7"));

        // when
        const { onIdentify } = renderModal();

        // then
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalledWith("hb-user-7");
        });
    });

    it("identifies nobody when the virtual browser hands back no seat", async () => {
        // given
        mocks.hyperbeam.mockResolvedValue(makeHandle(""));

        // when
        const { onIdentify } = renderModal();

        // then
        await waitFor(() => {
            expect(mocks.hyperbeam).toHaveBeenCalledOnce();
        });
        expect(onIdentify).not.toHaveBeenCalled();
    });

    it("unlocks the virtual browser when control is handed to the viewer", async () => {
        // given
        const handle = makeHandle();
        mocks.hyperbeam.mockResolvedValue(handle);
        const { onIdentify, rerender } = renderModal({ active: makeActive({ hasControl: false }) });
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalled();
        });

        // when
        rerender(
            <WatchPartyModal
                isOpen
                onClose={() => {}}
                active={makeActive({ hasControl: true })}
                viewerUserId={viewerId}
                viewerRole={undefined}
                isStarter={false}
                viewerIsStaff={false}
                voiceEnabled
                onLeave={() => Promise.resolve()}
                onEnd={() => Promise.resolve()}
                onTransferControl={() => Promise.resolve()}
                onKick={() => Promise.resolve()}
                onIdentify={() => Promise.resolve()}
            />,
        );

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

    it("tears the virtual browser down when the window goes away", async () => {
        // given
        const handle = makeHandle();
        mocks.hyperbeam.mockResolvedValue(handle);
        const { unmount, onIdentify } = renderModal();
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalled();
        });

        // when
        unmount();

        // then
        expect(handle.destroy).toHaveBeenCalledOnce();
    });
});

describe("WatchPartyModal screen share", () => {
    const screenShareActive = () => makeActive({ session: makeSession({ type: "screenshare" }), embedURL: "" });

    it("says it is still connecting until livekit is up", () => {
        // given
        stubMedia({ room: null });

        // when
        renderModal({ active: screenShareActive() });

        // then
        expect(screen.getByText("Connecting...")).toBeInTheDocument();
        expect(screen.queryByTestId("screen-share-view")).not.toBeInTheDocument();
    });

    it("never mounts a virtual browser for a screen share party", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ active: screenShareActive() });

        // then
        expect(mocks.hyperbeam).not.toHaveBeenCalled();
    });

    it("tells a watcher they are waiting on the host to share", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ active: screenShareActive(), isStarter: false });

        // then
        expect(screen.getByText("Waiting for the host to share their screen.")).toBeInTheDocument();
    });

    it("prompts the host to start sharing", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ active: screenShareActive(), isStarter: true });

        // then
        expect(screen.getByText("Click Share screen to start sharing.")).toBeInTheDocument();
    });

    it("rebuilds the stream when a watcher asks for a reload", async () => {
        // given
        const user = userEvent.setup();
        const media = stubMedia({ room: {} });
        renderModal({ active: screenShareActive() });

        // when
        await user.click(screen.getByRole("button", { name: "reload share" }));

        // then
        expect(media.reload).toHaveBeenCalledOnce();
    });

    it("asks the browser for fullscreen on the media panel", async () => {
        // given
        const user = userEvent.setup();
        const requestFullscreen = vi.fn(() => Promise.resolve());
        Object.defineProperty(Element.prototype, "requestFullscreen", {
            configurable: true,
            writable: true,
            value: requestFullscreen,
        });
        stubMedia({ room: {} });
        renderModal({ active: screenShareActive() });

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

    it("lets a watcher join the voice channel", async () => {
        // given
        const user = userEvent.setup();
        const media = stubMedia();
        renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "Join voice" }));

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

    it("hands moderation of the voice roster to the host", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ isStarter: true });

        // then
        expect(screen.getByTestId("voice-participants")).toHaveAttribute("data-can-moderate", "true");
    });

    it("withholds voice moderation from an ordinary watcher", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ isStarter: false, viewerIsStaff: false });

        // then
        expect(screen.getByTestId("voice-participants")).toHaveAttribute("data-can-moderate", "false");
    });

    it("forwards a forced mute to the party voice endpoint", async () => {
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
    });

    it("tells the moderator when a forced mute was refused", async () => {
        // given
        const user = userEvent.setup();
        mocks.forceMute.mockRejectedValue(new Error("that watcher outranks you"));
        stubMedia({ room: {} });
        renderModal({ isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "force mute" }));

        // then
        expect(await screen.findByRole("alert")).toHaveTextContent("that watcher outranks you");
    });

    it("falls back to a plain message when the refused mute carries none", async () => {
        // given
        const user = userEvent.setup();
        mocks.forceMute.mockRejectedValue(new Error(""));
        stubMedia({ room: {} });
        renderModal({ isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "force mute" }));

        // then
        expect(await screen.findByRole("alert")).toHaveTextContent("Could not change that microphone.");
    });

    it("says nothing about the microphone while every forced mute has worked", async () => {
        // given
        const user = userEvent.setup();
        stubMedia({ room: {} });
        renderModal({ isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "force mute" }));

        // then
        await waitFor(() => {
            expect(mocks.forceMute).toHaveBeenCalled();
        });
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });
});

describe("WatchPartyModal sharing controls", () => {
    const screenShareActive = () => makeActive({ session: makeSession({ type: "screenshare" }), embedURL: "" });

    it("keeps the share controls away from anybody but the host", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ active: screenShareActive(), isStarter: false });

        // then
        expect(screen.queryByRole("button", { name: "Share screen" })).not.toBeInTheDocument();
    });

    it("keeps the share controls off a virtual browser party", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ isStarter: true });

        // then
        expect(screen.queryByRole("button", { name: "Share screen" })).not.toBeInTheDocument();
    });

    it("shares with the gaming preset by default", async () => {
        // given
        const user = userEvent.setup();
        const media = stubMedia({ room: {} });
        renderModal({ active: screenShareActive(), isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "Share screen" }));

        // then
        expect(media.shareScreen).toHaveBeenCalledWith(true, "gaming");
    });

    it("shares with the screenshare preset once that mode is chosen", async () => {
        // given
        const user = userEvent.setup();
        const media = stubMedia({ room: {} });
        renderModal({ active: screenShareActive(), isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "Screenshare" }));
        await user.click(screen.getByRole("button", { name: "Share screen" }));

        // then
        expect(media.shareScreen).toHaveBeenCalledWith(true, "screenshare");
    });

    it("offers to stop a share that is already running", async () => {
        // given
        const user = userEvent.setup();
        const media = stubMedia({ room: {}, isSharing: true });
        renderModal({ active: screenShareActive(), isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "Stop sharing" }));

        // then
        expect(media.shareScreen).toHaveBeenCalledWith(false, "gaming");
        expect(screen.queryByRole("group", { name: "Stream mode" })).not.toBeInTheDocument();
    });

    it("tells the host why a share never started", () => {
        // given
        stubMedia({ room: {}, shareError: "Could not start audio source" });

        // when
        renderModal({ active: screenShareActive(), isStarter: true });

        // then
        expect(screen.getByRole("alert")).toHaveTextContent("Could not start audio source");
        expect(screen.getByRole("button", { name: "Share screen" })).toBeInTheDocument();
    });

    it("says nothing at all while no share has failed", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ active: screenShareActive(), isStarter: true });

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("keeps quiet about a picker the host simply closed", async () => {
        // given
        const user = userEvent.setup();
        stubMedia({ room: {} });
        renderModal({ active: screenShareActive(), isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "Share screen" }));

        // then
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("keeps claiming nothing is shared while the hook says the share never took", () => {
        // given
        stubMedia({ room: {}, isSharing: false, shareError: "Could not start audio source" });

        // when
        renderModal({ active: screenShareActive(), isStarter: true });

        // then
        expect(screen.queryByRole("button", { name: "Stop sharing" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Share screen" })).toBeInTheDocument();
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

    it("shows the party chat alongside the media, scoped to the session's own room", () => {
        // given
        const active = makeActive({ session: makeSession({ id: "session-42" }) });

        // when
        renderModal({ active });

        // then
        const panel = screen.getByTestId("room-chat-panel");
        expect(panel).toHaveTextContent("Party chat");
        expect(panel).toHaveAttribute("data-room-id", "session-42");
        expect(panel).toHaveAttribute("data-hide-header", "false");
    });

    it("lists the watchers of the party underneath", () => {
        // given
        const active = makeActive({
            session: makeSession({
                participants: [
                    makeParticipant(),
                    makeParticipant({ user: makeChatUser({ id: "user-battler", display_name: "Battler" }) }),
                ],
            }),
        });

        // when
        renderModal({ active });

        // then
        expect(screen.getByText("2 watchers")).toBeInTheDocument();
        expect(screen.getByText("Battler")).toBeInTheDocument();
    });
});

describe("WatchPartyModal on a phone", () => {
    beforeEach(() => {
        mocks.useIsMobile.mockReturnValue(true);
    });

    it("names the party and counts its watchers for the phone layout", () => {
        // given
        const active = makeActive({
            session: makeSession({
                title: "Chiru rewatch",
                participants: [
                    makeParticipant(),
                    makeParticipant({ user: makeChatUser({ id: "user-battler", display_name: "Battler" }) }),
                ],
            }),
        });

        // when
        renderModal({ active });

        // then
        const view = screen.getByTestId("mobile-view");
        expect(view).toHaveAttribute("data-title", "Chiru rewatch");
        expect(view).toHaveAttribute("data-watchers", "2");
    });

    it("falls back to a plain name for an untitled party on a phone", () => {
        // given
        const active = makeActive({ session: makeSession({ title: "" }) });

        // when
        renderModal({ active });

        // then
        expect(screen.getByTestId("mobile-view")).toHaveAttribute("data-title", "Untitled party");
    });

    it("keeps the full sized header actions off the phone", () => {
        // given
        const options = { isStarter: true };

        // when
        renderModal(options);

        // then
        expect(screen.queryByRole("button", { name: "Copy invite" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Hide" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Leave" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "End for everyone" })).not.toBeInTheDocument();
    });

    it("copies the invite from the phone's overflow menu", async () => {
        // given
        const user = userEvent.setup();
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "overflow copy invite" }));

        // then
        expect(writeText).toHaveBeenCalledWith("https://whentheycry.social/rooms/room-1?party=session-1");
        await waitFor(() => {
            expect(screen.getByTestId("mobile-view")).toHaveAttribute("data-copied", "true");
        });
    });

    it("hides the window from the phone's overflow menu without leaving", async () => {
        // given
        const user = userEvent.setup();
        const { onClose, onLeave } = renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "overflow hide" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
        expect(onLeave).not.toHaveBeenCalled();
    });

    it("leaves the party from the phone's overflow menu", async () => {
        // given
        const user = userEvent.setup();
        const { onClose, onLeave } = renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "overflow leave" }));

        // then
        expect(onLeave).toHaveBeenCalledOnce();
        await waitFor(() => {
            expect(onClose).toHaveBeenCalledOnce();
        });
    });

    it("ends the party for everyone from the phone's overflow menu", async () => {
        // given
        const user = userEvent.setup();
        const { onEnd } = renderModal({ isStarter: true });

        // when
        await user.click(screen.getByRole("button", { name: "overflow end" }));

        // then
        expect(onEnd).toHaveBeenCalledOnce();
        expect(screen.getByTestId("mobile-view")).toHaveAttribute("data-can-end", "true");
    });

    it("tells the phone layout that an ordinary watcher may not end the party", () => {
        // given
        const options = { isStarter: false, viewerIsStaff: false };

        // when
        renderModal(options);

        // then
        expect(screen.getByTestId("mobile-view")).toHaveAttribute("data-can-end", "false");
    });

    it("gives the phone a chat pane with no header of its own", () => {
        // given
        const active = makeActive({ session: makeSession({ id: "session-42" }) });

        // when
        renderModal({ active });

        // then
        const panel = within(screen.getByTestId("mobile-chat")).getByTestId("room-chat-panel");
        expect(panel).toHaveAttribute("data-room-id", "session-42");
        expect(panel).toHaveAttribute("data-hide-header", "true");
        expect(panel).toHaveAttribute("data-flush", "true");
    });

    it("puts the voice roster and its join control in the voice pane", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal({ isStarter: true });

        // then
        const pane = within(screen.getByTestId("mobile-voice"));
        expect(pane.getByRole("button", { name: "Join voice" })).toBeInTheDocument();
        expect(pane.getByTestId("voice-participants")).toHaveAttribute("data-can-moderate", "true");
    });

    it("keeps the host's share controls reachable even with voice switched off", () => {
        // given
        stubMedia({ room: {} });
        const active = makeActive({ session: makeSession({ type: "screenshare" }), embedURL: "" });

        // when
        renderModal({ active, isStarter: true, voiceEnabled: false });

        // then
        const pane = within(screen.getByTestId("mobile-voice"));
        expect(pane.getByRole("button", { name: "Share screen" })).toBeInTheDocument();
        expect(pane.queryByRole("button", { name: "Join voice" })).not.toBeInTheDocument();
    });

    it("counts the watchers who are in voice for the voice tab", () => {
        // given
        stubMedia({ room: {} });
        mocks.useParticipants.mockReturnValue([{ identity: "beatrice" }, { identity: "battler" }]);

        // when
        renderModal();

        // then
        expect(screen.getByTestId("mobile-view")).toHaveAttribute("data-voices", "2");
    });

    it("counts nobody in voice while the site has voice switched off", () => {
        // given
        stubMedia({ room: {} });
        mocks.useParticipants.mockReturnValue([{ identity: "beatrice" }]);

        // when
        renderModal({ voiceEnabled: false });

        // then
        expect(screen.getByTestId("mobile-view")).toHaveAttribute("data-voices", "0");
        expect(within(screen.getByTestId("mobile-voice")).queryByTestId("voice-participants")).not.toBeInTheDocument();
    });

    it("keeps the party audio playing from the stage while another pane is open", () => {
        // given
        stubMedia({ room: {} });

        // when
        renderModal();

        // then
        expect(within(screen.getByTestId("mobile-stage")).getByTestId("room-audio")).toBeInTheDocument();
    });

    it("lists the watchers in the people pane", () => {
        // given
        const active = makeActive({
            session: makeSession({
                participants: [
                    makeParticipant(),
                    makeParticipant({ user: makeChatUser({ id: "user-battler", display_name: "Battler" }) }),
                ],
            }),
        });

        // when
        renderModal({ active });

        // then
        const pane = within(screen.getByTestId("mobile-people"));
        expect(pane.getByText("2 watchers")).toBeInTheDocument();
        expect(pane.getByText("Battler")).toBeInTheDocument();
    });

    it("fullscreens the whole stage from the control sitting on the picture", async () => {
        // given
        const user = userEvent.setup();
        const requestFullscreen = vi.fn(() => Promise.resolve());
        Object.defineProperty(Element.prototype, "requestFullscreen", {
            configurable: true,
            writable: true,
            value: requestFullscreen,
        });
        stubMedia({ room: {} });
        const active = makeActive({ session: makeSession({ type: "screenshare" }), embedURL: "" });
        renderModal({ active });

        // when
        const stage = within(screen.getByTestId("mobile-stage"));
        await user.click(stage.getByRole("button", { name: "Fullscreen" }));

        // then
        expect(stage.getByTestId("screen-share-view")).toBeInTheDocument();
        expect(requestFullscreen).toHaveBeenCalledOnce();
        expect(requestFullscreen.mock.contexts[0]).toBe(screen.getByTestId("mobile-stage"));
    });
});
