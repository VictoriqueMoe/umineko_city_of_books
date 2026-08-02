import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { User, WatchPartyParticipant } from "../../../types/api";
import type { SiteRole } from "../../../utils/permissions";
import { WatchPartyParticipants } from "./WatchPartyParticipants";

interface NodeProcess {
    on(event: "unhandledRejection", handler: (reason: unknown) => void): void;
    off(event: "unhandledRejection", handler: (reason: unknown) => void): void;
}

const nodeProcess = (globalThis as unknown as { process: NodeProcess }).process;

const viewerId = "user-viewer";
const ownerId = "user-owner";

function makeChatUser(overrides: Partial<User> = {}): User {
    return { id: viewerId, username: "beatrice", display_name: "Beatrice", ...overrides };
}

function makeParticipant(overrides: Partial<WatchPartyParticipant> = {}): WatchPartyParticipant {
    return { user: makeChatUser(), has_control: false, joined_at: "2026-08-01T10:00:00Z", ...overrides };
}

interface StripOptions {
    participants?: WatchPartyParticipant[];
    viewerUserId?: string;
    viewerRole?: SiteRole | undefined;
    viewerHasControl?: boolean;
    ownerUserId?: string;
    onTransferControl?: (userId: string) => Promise<void>;
}

function renderStrip(options: StripOptions = {}) {
    const onTransferControl = vi.fn(options.onTransferControl ?? (() => Promise.resolve()));
    const onKick = vi.fn(() => Promise.resolve());

    const result = renderWithProviders(
        <WatchPartyParticipants
            participants={options.participants ?? [makeParticipant()]}
            viewerUserId={options.viewerUserId ?? viewerId}
            viewerRole={options.viewerRole}
            viewerHasControl={options.viewerHasControl ?? false}
            ownerUserId={options.ownerUserId ?? ownerId}
            onTransferControl={onTransferControl}
            onKick={onKick}
        />,
    );

    return { ...result, onTransferControl, onKick };
}

const battler = makeChatUser({ id: "user-battler", username: "battler", display_name: "Battler" });
const owner = makeChatUser({ id: ownerId, username: "kinzo", display_name: "Kinzo" });

describe("WatchPartyParticipants roster", () => {
    it("counts a lone watcher in the singular", () => {
        // given
        const participants = [makeParticipant()];

        // when
        renderStrip({ participants });

        // then
        expect(screen.getByText("1 watcher")).toBeInTheDocument();
    });

    it("counts several watchers in the plural", () => {
        // given
        const participants = [makeParticipant(), makeParticipant({ user: battler })];

        // when
        renderStrip({ participants });

        // then
        expect(screen.getByText("2 watchers")).toBeInTheDocument();
    });

    it("marks who owns the party and who is driving it", () => {
        // given
        const participants = [makeParticipant({ user: owner }), makeParticipant({ user: battler, has_control: true })];

        // when
        renderStrip({ participants });

        // then
        expect(screen.getByText("owner")).toBeInTheDocument();
        expect(screen.getByText("control")).toBeInTheDocument();
    });
});

describe("WatchPartyParticipants control handover", () => {
    it("lets the owner take control back from a plain watcher", async () => {
        // given
        const user = userEvent.setup();
        const participants = [
            makeParticipant({ user: makeChatUser({ id: ownerId, display_name: "Kinzo" }) }),
            makeParticipant({ user: battler, has_control: true }),
        ];
        const { onTransferControl } = renderStrip({ participants, viewerUserId: ownerId });

        // when
        await user.click(screen.getByRole("button", { name: "Reclaim control" }));

        // then
        expect(onTransferControl).toHaveBeenCalledWith(ownerId);
    });

    it("offers the same reclaim from the row of whoever currently drives", async () => {
        // given
        const user = userEvent.setup();
        const participants = [
            makeParticipant({ user: makeChatUser({ id: ownerId, display_name: "Kinzo" }) }),
            makeParticipant({ user: battler, has_control: true }),
        ];
        const { onTransferControl } = renderStrip({ participants, viewerUserId: ownerId });

        // when
        await user.click(screen.getByRole("button", { name: "Reclaim" }));

        // then
        expect(onTransferControl).toHaveBeenCalledWith(ownerId);
    });

    it("refuses a plain watcher any way to take control from the owner", () => {
        // given
        const participants = [makeParticipant(), makeParticipant({ user: owner, has_control: true })];

        // when
        renderStrip({ participants });

        // then
        expect(screen.queryByRole("button", { name: "Reclaim control" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Reclaim" })).not.toBeInTheDocument();
    });

    it("lets a site moderator wrest control away from the owner", async () => {
        // given
        const user = userEvent.setup();
        const participants = [
            makeParticipant({ user: makeChatUser({ role: "moderator" }) }),
            makeParticipant({ user: owner, has_control: true }),
        ];
        const { onTransferControl } = renderStrip({ participants, viewerRole: "moderator" });

        // when
        await user.click(screen.getByRole("button", { name: "Reclaim" }));

        // then
        expect(onTransferControl).toHaveBeenCalledWith(viewerId);
    });

    it("lets the controller pass the seat on to another watcher", async () => {
        // given
        const user = userEvent.setup();
        const participants = [makeParticipant({ has_control: true }), makeParticipant({ user: battler })];
        const { onTransferControl } = renderStrip({ participants, viewerHasControl: true });

        // when
        await user.click(screen.getByRole("button", { name: "Pass control" }));

        // then
        expect(onTransferControl).toHaveBeenCalledWith("user-battler");
    });

    it("offers a handover to everybody while nobody holds control at all", async () => {
        // given
        const user = userEvent.setup();
        const participants = [makeParticipant(), makeParticipant({ user: battler })];
        const { onTransferControl } = renderStrip({ participants });

        // when
        await user.click(screen.getByRole("button", { name: "Pass control" }));

        // then
        expect(screen.getByRole("button", { name: "Reclaim control" })).toBeInTheDocument();
        expect(onTransferControl).toHaveBeenCalledWith("user-battler");
    });

    it("keeps a plain watcher from handing the seat around behind the owner's back", () => {
        // given
        const participants = [
            makeParticipant(),
            makeParticipant({ user: owner, has_control: true }),
            makeParticipant({ user: battler }),
        ];

        // when
        renderStrip({ participants });

        // then
        expect(screen.queryByRole("button", { name: "Pass control" })).not.toBeInTheDocument();
    });

    it("swallows a refused handover instead of leaving the rejection unhandled", async () => {
        // given
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        nodeProcess.on("unhandledRejection", record);
        const user = userEvent.setup();
        const participants = [makeParticipant(), makeParticipant({ user: battler })];
        renderStrip({
            participants,
            onTransferControl: () => Promise.reject(new Error("you do not outrank the controller")),
        });

        // when
        await user.click(screen.getByRole("button", { name: "Pass control" }));
        await new Promise(resolve => setTimeout(resolve, 0));
        nodeProcess.off("unhandledRejection", record);

        // then
        expect(unhandled).toEqual([]);
        expect(screen.getByRole("button", { name: "Pass control" })).toBeEnabled();
    });

    it("disables the row while its handover is still running", async () => {
        // given
        let release: () => void = () => {};
        const user = userEvent.setup();
        const participants = [makeParticipant(), makeParticipant({ user: battler })];
        renderStrip({
            participants,
            onTransferControl: () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        });

        // when
        await user.click(screen.getByRole("button", { name: "Pass control" }));

        // then
        expect(screen.getByRole("button", { name: "Pass control" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Reclaim control" })).toBeEnabled();
        release();
        await waitFor(() => {
            expect(screen.getByRole("button", { name: "Pass control" })).toBeEnabled();
        });
    });
});

describe("WatchPartyParticipants removals", () => {
    it("never offers the viewer a way to remove themselves", () => {
        // given
        const participants = [makeParticipant({ user: makeChatUser({ role: "admin" }) })];

        // when
        renderStrip({ participants, viewerRole: "admin" });

        // then
        expect(screen.queryByRole("button", { name: "Kick" })).not.toBeInTheDocument();
    });

    it("refuses a plain watcher any way to remove anybody", () => {
        // given
        const participants = [makeParticipant(), makeParticipant({ user: battler })];

        // when
        renderStrip({ participants });

        // then
        expect(screen.queryByRole("button", { name: "Kick" })).not.toBeInTheDocument();
    });

    it("lets a site moderator remove a plain watcher", async () => {
        // given
        const user = userEvent.setup();
        const participants = [
            makeParticipant({ user: makeChatUser({ role: "moderator" }) }),
            makeParticipant({ user: battler }),
        ];
        const { onKick } = renderStrip({ participants, viewerRole: "moderator" });

        // when
        await user.click(screen.getByRole("button", { name: "Kick" }));

        // then
        expect(onKick).toHaveBeenCalledWith("user-battler");
    });

    it("keeps a moderator from removing another moderator of the same standing", () => {
        // given
        const participants = [
            makeParticipant({ user: makeChatUser({ role: "moderator" }) }),
            makeParticipant({ user: { ...battler, role: "moderator" } }),
        ];

        // when
        renderStrip({ participants, viewerRole: "moderator" });

        // then
        expect(screen.queryByRole("button", { name: "Kick" })).not.toBeInTheDocument();
    });

    it("lets an admin remove a moderator", async () => {
        // given
        const user = userEvent.setup();
        const participants = [
            makeParticipant({ user: makeChatUser({ role: "admin" }) }),
            makeParticipant({ user: { ...battler, role: "moderator" } }),
        ];
        const { onKick } = renderStrip({ participants, viewerRole: "admin" });

        // when
        await user.click(screen.getByRole("button", { name: "Kick" }));

        // then
        expect(onKick).toHaveBeenCalledWith("user-battler");
    });

    it("lets a site moderator remove even the owner of the party", async () => {
        // given
        const user = userEvent.setup();
        const participants = [
            makeParticipant({ user: makeChatUser({ role: "moderator" }) }),
            makeParticipant({ user: owner }),
        ];
        const { onKick } = renderStrip({ participants, viewerRole: "moderator", ownerUserId: ownerId });

        // when
        await user.click(screen.getByRole("button", { name: "Kick" }));

        // then
        expect(onKick).toHaveBeenCalledWith(ownerId);
    });

    it("keeps the owner from removing a site moderator who outranks them", () => {
        // given
        const participants = [
            makeParticipant({ user: makeChatUser({ id: ownerId, display_name: "Kinzo" }) }),
            makeParticipant({ user: { ...battler, role: "moderator" } }),
        ];

        // when
        renderStrip({ participants, viewerUserId: ownerId });

        // then
        expect(screen.queryByRole("button", { name: "Kick" })).not.toBeInTheDocument();
    });

    it("lets the owner remove a plain watcher on the strength of owning the party", async () => {
        // given
        const user = userEvent.setup();
        const participants = [
            makeParticipant({ user: makeChatUser({ id: ownerId, display_name: "Kinzo" }) }),
            makeParticipant({ user: battler }),
        ];
        const { onKick } = renderStrip({ participants, viewerUserId: ownerId });

        // when
        await user.click(screen.getByRole("button", { name: "Kick" }));

        // then
        expect(onKick).toHaveBeenCalledWith("user-battler");
    });
});
