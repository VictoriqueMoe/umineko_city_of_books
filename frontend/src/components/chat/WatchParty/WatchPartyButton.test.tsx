import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { User, WatchPartyParticipant, WatchPartySession } from "../../../types/api";
import { WatchPartyButton } from "./WatchPartyButton";

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
    return {
        id: "session-1",
        room_id: "room-1",
        started_by: viewerId,
        controller_id: viewerId,
        title: "Chiru rewatch",
        type: "hyperbeam",
        status: "active",
        started_at: "2026-08-01T10:00:00Z",
        participants: [],
        ...overrides,
    };
}

interface ButtonOptions {
    enabled?: boolean;
    screenShareEnabled?: boolean;
    sessions?: WatchPartySession[];
    activeSessionId?: string | null;
    viewerUserId?: string | null;
    onStart?: (opts: { title?: string; type?: "hyperbeam" | "screenshare" }) => Promise<unknown>;
    onJoin?: (sessionId: string) => Promise<void>;
}

function renderButton(options: ButtonOptions = {}) {
    const onStart = vi.fn(options.onStart ?? (() => Promise.resolve(null)));
    const onJoin = vi.fn(options.onJoin ?? (() => Promise.resolve()));
    const onOpenExisting = vi.fn();

    const result = renderWithProviders(
        <WatchPartyButton
            enabled={options.enabled ?? true}
            screenShareEnabled={options.screenShareEnabled ?? true}
            sessions={options.sessions ?? []}
            activeSessionId={options.activeSessionId ?? null}
            viewerUserId={options.viewerUserId === undefined ? viewerId : options.viewerUserId}
            onStart={onStart}
            onJoin={onJoin}
            onOpenExisting={onOpenExisting}
        />,
    );

    return { ...result, onStart, onJoin, onOpenExisting };
}

async function openPicker(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole("button", { name: /Watch Part/i }));
}

describe("WatchPartyButton", () => {
    it("renders nothing when the room allows neither kind of party", () => {
        // given
        const options = { enabled: false, screenShareEnabled: false };

        // when
        const { container } = renderButton(options);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("invites the visitor to start the first party of the room", () => {
        // given
        const sessions: WatchPartySession[] = [];

        // when
        renderButton({ sessions });

        // then
        expect(screen.getByRole("button", { name: "+ Watch Party" })).toBeInTheDocument();
    });

    it("counts the parties that are already running", () => {
        // given
        const sessions = [makeSession({ id: "session-1" }), makeSession({ id: "session-2" })];

        // when
        renderButton({ sessions });

        // then
        expect(screen.getByRole("button", { name: "Watch Parties (2)" })).toBeInTheDocument();
    });

    it("says so plainly when the picker has no parties to offer", async () => {
        // given
        const user = userEvent.setup();
        renderButton();

        // when
        await openPicker(user);

        // then
        expect(screen.getByText("No active parties yet.")).toBeInTheDocument();
    });

    it("names an untitled party and counts a single watcher in the singular", async () => {
        // given
        const user = userEvent.setup();
        renderButton({ sessions: [makeSession({ title: "", participants: [makeParticipant()] })] });

        // when
        await openPicker(user);

        // then
        expect(screen.getByText("Untitled party")).toBeInTheDocument();
        expect(screen.getByText("1 participant")).toBeInTheDocument();
    });

    it("counts several watchers in the plural", async () => {
        // given
        const user = userEvent.setup();
        const participants = [makeParticipant(), makeParticipant({ user: makeChatUser({ id: "user-battler" }) })];
        renderButton({ sessions: [makeSession({ participants })] });

        // when
        await openPicker(user);

        // then
        expect(screen.getByText("2 participants")).toBeInTheDocument();
    });

    it("offers to open the party the viewer already has running", async () => {
        // given
        const user = userEvent.setup();
        const { onOpenExisting } = renderButton({
            sessions: [makeSession()],
            activeSessionId: "session-1",
        });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Open" }));

        // then
        expect(onOpenExisting).toHaveBeenCalledWith("session-1");
        expect(screen.queryByText("Watch parties")).not.toBeInTheDocument();
    });

    it("offers to resume a party the viewer is a member of but has hidden", async () => {
        // given
        const user = userEvent.setup();
        const { onJoin } = renderButton({ sessions: [makeSession({ participants: [makeParticipant()] })] });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Resume" }));

        // then
        expect(onJoin).toHaveBeenCalledWith("session-1");
        await waitFor(() => {
            expect(screen.queryByText("Watch parties")).not.toBeInTheDocument();
        });
    });

    it("offers to join a party the viewer has never been part of", async () => {
        // given
        const user = userEvent.setup();
        const participants = [makeParticipant({ user: makeChatUser({ id: "user-battler" }) })];
        const { onJoin } = renderButton({ sessions: [makeSession({ participants })] });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Join" }));

        // then
        expect(onJoin).toHaveBeenCalledWith("session-1");
    });

    it("offers a signed out visitor nothing but a plain join", async () => {
        // given
        const user = userEvent.setup();
        renderButton({ sessions: [makeSession({ participants: [makeParticipant()] })], viewerUserId: null });

        // when
        await openPicker(user);

        // then
        expect(screen.getByRole("button", { name: "Join" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Resume" })).not.toBeInTheDocument();
    });

    it("starts a virtual browser party with the title that was typed", async () => {
        // given
        const user = userEvent.setup();
        const { onStart } = renderButton();
        await openPicker(user);

        // when
        await user.type(screen.getByPlaceholderText("Title (optional)"), "  Chiru rewatch  ");
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then
        expect(onStart).toHaveBeenCalledWith({ title: "Chiru rewatch", type: "hyperbeam" });
    });

    it("starts a virtual browser party once the flags arrive, even though it mounted before they did", async () => {
        // given the button mounted before the watch party list resolved, so both flags were still false
        const user = userEvent.setup();
        const { onStart, rerender } = renderButton({ enabled: false, screenShareEnabled: false });

        // when the list arrives saying the virtual browser is the only option
        rerender(
            <WatchPartyButton
                enabled={true}
                screenShareEnabled={false}
                sessions={[]}
                activeSessionId={null}
                viewerUserId={viewerId}
                onStart={onStart}
                onJoin={vi.fn()}
                onOpenExisting={vi.fn()}
            />,
        );
        await openPicker(user);
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then it must not fall back to the screen share it was mounted with
        expect(onStart).toHaveBeenCalledWith({ title: undefined, type: "hyperbeam" });
    });

    it("starts a screen share party when that is the only option available", async () => {
        // given
        const user = userEvent.setup();
        const { onStart } = renderButton({ enabled: false, screenShareEnabled: true });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then
        expect(onStart).toHaveBeenCalledWith({ title: undefined, type: "screenshare" });
    });

    it("sends no title at all when the field was left blank", async () => {
        // given
        const user = userEvent.setup();
        const { onStart } = renderButton();
        await openPicker(user);

        // when
        await user.type(screen.getByPlaceholderText("Title (optional)"), "   ");
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then
        expect(onStart).toHaveBeenCalledWith({ title: undefined, type: "hyperbeam" });
    });

    it("closes the picker and forgets the draft title once the party has started", async () => {
        // given
        const user = userEvent.setup();
        renderButton();
        await openPicker(user);
        await user.type(screen.getByPlaceholderText("Title (optional)"), "Chiru rewatch");

        // when
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then
        await waitFor(() => {
            expect(screen.queryByText("Watch parties")).not.toBeInTheDocument();
        });
        await openPicker(user);
        expect(screen.getByPlaceholderText("Title (optional)")).toHaveValue("");
    });

    it("lets the host choose a screen share instead of a virtual browser", async () => {
        // given
        const user = userEvent.setup();
        const { onStart } = renderButton();
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Screen share" }));
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then
        expect(onStart).toHaveBeenCalledWith({ title: undefined, type: "screenshare" });
    });

    it("hides the type choice when the room only allows a virtual browser", async () => {
        // given
        const user = userEvent.setup();
        const { onStart } = renderButton({ enabled: true, screenShareEnabled: false });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then
        expect(screen.queryByRole("button", { name: "Virtual browser" })).not.toBeInTheDocument();
        expect(onStart).toHaveBeenCalledWith({ title: undefined, type: "hyperbeam" });
    });

    it("falls back to a screen share when the room has no virtual browser", async () => {
        // given
        const user = userEvent.setup();
        const { onStart } = renderButton({ enabled: false, screenShareEnabled: true });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then
        expect(onStart).toHaveBeenCalledWith({ title: undefined, type: "screenshare" });
    });

    it("shows that a party is being created while the request is in flight", async () => {
        // given
        let release: () => void = () => {};
        const user = userEvent.setup();
        renderButton({
            onStart: () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Start new" }));

        // then
        expect(screen.getByRole("button", { name: "Starting..." })).toBeDisabled();
        expect(screen.getByPlaceholderText("Title (optional)")).toBeDisabled();
        release();
        await waitFor(() => {
            expect(screen.queryByText("Watch parties")).not.toBeInTheDocument();
        });
    });

    it("swallows a refused start instead of leaving the rejection unhandled", async () => {
        // given
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        nodeProcess.on("unhandledRejection", record);
        const user = userEvent.setup();
        renderButton({ onStart: () => Promise.reject(new Error("too many parties already")) });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Start new" }));
        await new Promise(resolve => setTimeout(resolve, 0));
        nodeProcess.off("unhandledRejection", record);

        // then
        expect(unhandled).toEqual([]);
        expect(screen.getByRole("button", { name: "Start new" })).toBeEnabled();
    });

    it("swallows a refused join instead of leaving the rejection unhandled", async () => {
        // given
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        nodeProcess.on("unhandledRejection", record);
        const user = userEvent.setup();
        const participants = [makeParticipant({ user: makeChatUser({ id: "user-battler" }) })];
        renderButton({
            sessions: [makeSession({ participants })],
            onJoin: () => Promise.reject(new Error("you are banned from this room")),
        });
        await openPicker(user);

        // when
        await user.click(screen.getByRole("button", { name: "Join" }));
        await new Promise(resolve => setTimeout(resolve, 0));
        nodeProcess.off("unhandledRejection", record);

        // then
        expect(unhandled).toEqual([]);
        expect(screen.getByRole("button", { name: "Join" })).toBeEnabled();
    });

    it("closes the picker when the visitor clicks away from it", async () => {
        // given
        const user = userEvent.setup();
        renderButton();
        await openPicker(user);
        expect(screen.getByText("Watch parties")).toBeInTheDocument();

        // when
        await user.click(document.body);

        // then
        expect(screen.queryByText("Watch parties")).not.toBeInTheDocument();
    });

    it("keeps the picker open while the visitor works inside it", async () => {
        // given
        const user = userEvent.setup();
        renderButton();
        await openPicker(user);

        // when
        await user.click(screen.getByPlaceholderText("Title (optional)"));

        // then
        expect(screen.getByText("Watch parties")).toBeInTheDocument();
    });
});
