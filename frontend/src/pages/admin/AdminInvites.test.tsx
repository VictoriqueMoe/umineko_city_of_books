import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { InviteItem } from "../../types/api";
import { renderWithProviders } from "../../test-utils/render";
import { AdminInvites } from "./AdminInvites";

const mocks = vi.hoisted(() => ({
    useInvites: vi.fn(),
    create: vi.fn(),
    remove: vi.fn(),
}));

vi.mock("../../hooks/queries/admin", () => ({ useInvites: mocks.useInvites }));

vi.mock("../../hooks/mutations/admin", () => ({
    useCreateInvite: () => ({ mutateAsync: mocks.create, isPending: false }),
    useDeleteInvite: () => ({ mutateAsync: mocks.remove, isPending: false }),
}));

function makeInvite(overrides: Partial<InviteItem> = {}): InviteItem {
    return {
        code: "kakera-77",
        created_by: "staff-1",
        created_at: "2026-01-02T00:00:00Z",
        ...overrides,
    };
}

function stubInvites(invites: InviteItem[], loading = false) {
    mocks.useInvites.mockReturnValue({ invites, loading, refresh: vi.fn() });
}

beforeEach(() => {
    mocks.create.mockResolvedValue(undefined);
    mocks.remove.mockResolvedValue(undefined);
});

describe("AdminInvites", () => {
    it("waits while the invite codes are being fetched", () => {
        // given
        stubInvites([], true);

        // when
        renderWithProviders(<AdminInvites />);

        // then
        expect(screen.getByText("Loading invites...")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Create Invite" })).not.toBeInTheDocument();
    });

    it("says so when no invite has been created yet", () => {
        // given
        stubInvites([]);

        // when
        renderWithProviders(<AdminInvites />);

        // then
        expect(screen.getByText("No invites created yet.")).toBeInTheDocument();
    });

    it("lists an unused code as available and offers to delete it", () => {
        // given
        stubInvites([makeInvite()]);

        // when
        renderWithProviders(<AdminInvites />);

        // then
        expect(screen.getByText("kakera-77")).toBeInTheDocument();
        expect(screen.getByText("Available")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("marks a redeemed code as used and withholds the delete action", () => {
        // given
        stubInvites([makeInvite({ used_by: "user-9", used_at: "2026-01-03T00:00:00Z" })]);

        // when
        renderWithProviders(<AdminInvites />);

        // then
        expect(screen.getByText("Used")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("mints a fresh invite code on request", async () => {
        // given
        stubInvites([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminInvites />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Invite" }));

        // then
        expect(mocks.create).toHaveBeenCalledOnce();
    });

    it("reports why a new invite could not be minted", async () => {
        // given
        stubInvites([]);
        mocks.create.mockRejectedValue(new Error("the golden land is full"));
        const user = userEvent.setup();
        renderWithProviders(<AdminInvites />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Invite" }));

        // then
        expect(await screen.findByText("the golden land is full")).toBeInTheDocument();
    });

    it("asks before deleting an invite and deletes nothing when the ask is refused", async () => {
        // given
        stubInvites([makeInvite()]);
        const user = userEvent.setup();
        renderWithProviders(<AdminInvites />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));
        const dialog = await screen.findByRole("dialog", { name: "Delete Invite" });
        expect(within(dialog).getByText("Are you sure you want to delete this invite?")).toBeInTheDocument();
        await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("deletes the invite by its code once confirmed", async () => {
        // given
        stubInvites([makeInvite({ code: "kakera-99" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminInvites />);
        await user.click(screen.getByRole("button", { name: "Delete" }));
        const dialog = await screen.findByRole("dialog", { name: "Delete Invite" });

        // when
        await user.click(within(dialog).getByRole("button", { name: "Delete" }));

        // then
        expect(mocks.remove).toHaveBeenCalledWith("kakera-99");
    });

    it("reports why an invite could not be deleted", async () => {
        // given
        stubInvites([makeInvite()]);
        mocks.remove.mockRejectedValue(new Error("that code is already spent"));
        const user = userEvent.setup();
        renderWithProviders(<AdminInvites />);
        await user.click(screen.getByRole("button", { name: "Delete" }));
        const dialog = await screen.findByRole("dialog", { name: "Delete Invite" });

        // when
        await user.click(within(dialog).getByRole("button", { name: "Delete" }));

        // then
        expect(await screen.findByText("that code is already spent")).toBeInTheDocument();
    });
});
