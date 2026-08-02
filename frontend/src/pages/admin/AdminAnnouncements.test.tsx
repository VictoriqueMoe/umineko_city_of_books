import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { Announcement } from "../../types/api";
import { AdminAnnouncements } from "./AdminAnnouncements";

const mocks = vi.hoisted(() => ({
    useAdminAnnouncements: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    pin: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({ useAdminAnnouncements: mocks.useAdminAnnouncements }));

vi.mock("../../api/mutations/admin", () => ({
    useCreateAnnouncement: () => ({ mutateAsync: mocks.create, isPending: false }),
    useUpdateAnnouncement: () => ({ mutateAsync: mocks.update, isPending: false }),
    useDeleteAnnouncement: () => ({ mutateAsync: mocks.remove, isPending: false }),
    usePinAnnouncement: () => ({ mutateAsync: mocks.pin, isPending: false }),
}));

function makeAnnouncement(overrides: Partial<Announcement> = {}): Announcement {
    return {
        id: "ann-1",
        title: "The witch returns",
        body: "# Golden\n\nA new game begins.",
        author: { id: "staff-1", username: "virgilia", display_name: "Virgilia" },
        pinned: false,
        created_at: "2026-01-02T00:00:00Z",
        updated_at: "2026-01-02T00:00:00Z",
        ...overrides,
    };
}

function stubAnnouncements(announcements: Announcement[], loading = false) {
    mocks.useAdminAnnouncements.mockReturnValue({ announcements, loading, refresh: vi.fn() });
}

beforeEach(() => {
    mocks.create.mockResolvedValue(undefined);
    mocks.update.mockResolvedValue(undefined);
    mocks.remove.mockResolvedValue(undefined);
    mocks.pin.mockResolvedValue(undefined);
});

describe("AdminAnnouncements", () => {
    it("waits while the announcements are being fetched", () => {
        // given
        stubAnnouncements([], true);

        // when
        renderWithProviders(<AdminAnnouncements />);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("says so when nothing has been announced yet", () => {
        // given
        stubAnnouncements([]);

        // when
        renderWithProviders(<AdminAnnouncements />);

        // then
        expect(screen.getByText("No announcements yet.")).toBeInTheDocument();
    });

    it("lists an announcement with its author and marks the pinned one", () => {
        // given
        stubAnnouncements([makeAnnouncement({ pinned: true })]);

        // when
        renderWithProviders(<AdminAnnouncements />);

        // then
        expect(screen.getByText("The witch returns")).toBeInTheDocument();
        expect(screen.getByText("Pinned")).toBeInTheDocument();
        expect(screen.getByText("Virgilia")).toBeInTheDocument();
    });

    it("opens an empty editor when a new announcement is started", async () => {
        // given
        stubAnnouncements([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);

        // when
        await user.click(screen.getByRole("button", { name: "Create Announcement" }));

        // then
        expect(screen.getByRole("heading", { name: "Create Announcement" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Announcement title...")).toHaveValue("");
        expect(screen.getByRole("button", { name: "Publish" })).toBeDisabled();
    });

    it("refuses to publish until both a title and a body are written", async () => {
        // given
        stubAnnouncements([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);
        await user.click(screen.getByRole("button", { name: "Create Announcement" }));

        // when
        await user.type(screen.getByPlaceholderText("Announcement title..."), "The witch returns");

        // then
        expect(screen.getByRole("button", { name: "Publish" })).toBeDisabled();
    });

    it("publishes the trimmed title and body and then closes the editor", async () => {
        // given
        stubAnnouncements([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);
        await user.click(screen.getByRole("button", { name: "Create Announcement" }));
        await user.type(screen.getByPlaceholderText("Announcement title..."), "  The witch returns  ");
        await user.type(screen.getByPlaceholderText("Write your announcement in Markdown..."), "  A new game.  ");

        // when
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith({ title: "The witch returns", body: "A new game." });
        expect(await screen.findByRole("button", { name: "Create Announcement" })).toBeInTheDocument();
    });

    it("keeps the editor open when publishing fails", async () => {
        // given
        stubAnnouncements([]);
        mocks.create.mockRejectedValue(new Error("the golden land refuses"));
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);
        await user.click(screen.getByRole("button", { name: "Create Announcement" }));
        await user.type(screen.getByPlaceholderText("Announcement title..."), "The witch returns");
        await user.type(screen.getByPlaceholderText("Write your announcement in Markdown..."), "A new game.");

        // when
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(screen.getByRole("heading", { name: "Create Announcement" })).toBeInTheDocument();
    });

    it("prefills the editor from the announcement being edited", async () => {
        // given
        stubAnnouncements([makeAnnouncement()]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // then
        expect(screen.getByRole("heading", { name: "Edit Announcement" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Announcement title...")).toHaveValue("The witch returns");
        expect(screen.getByRole("button", { name: "Save Changes" })).toBeInTheDocument();
    });

    it("saves an edit against the announcement it came from", async () => {
        // given
        stubAnnouncements([makeAnnouncement({ id: "ann-9" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);
        await user.click(screen.getByRole("button", { name: "Edit" }));
        await user.clear(screen.getByPlaceholderText("Announcement title..."));
        await user.type(screen.getByPlaceholderText("Announcement title..."), "The witch departs");

        // when
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "ann-9",
            title: "The witch departs",
            body: "# Golden\n\nA new game begins.",
        });
    });

    it("abandons an edit without saving anything", async () => {
        // given
        stubAnnouncements([makeAnnouncement()]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(mocks.update).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Create Announcement" })).toBeInTheDocument();
    });

    it("renders the written markdown when the preview tab is chosen", async () => {
        // given
        stubAnnouncements([makeAnnouncement()]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // when
        await user.click(screen.getByRole("button", { name: "Preview" }));

        // then
        expect(screen.getByRole("heading", { name: "Golden", level: 1 })).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Write your announcement in Markdown...")).not.toBeInTheDocument();
    });

    it("asks before deleting an announcement", async () => {
        // given
        stubAnnouncements([makeAnnouncement()]);
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this announcement?");
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("deletes the announcement once confirmed", async () => {
        // given
        stubAnnouncements([makeAnnouncement({ id: "ann-4" })]);
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(mocks.remove).toHaveBeenCalledWith("ann-4");
    });

    it("reports why an announcement could not be deleted", async () => {
        // given
        stubAnnouncements([makeAnnouncement({ id: "ann-4" })]);
        mocks.remove.mockRejectedValue(new Error("the golden land refuses"));
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(await screen.findByText("the golden land refuses")).toBeInTheDocument();
    });

    it("pins an unpinned announcement", async () => {
        // given
        stubAnnouncements([makeAnnouncement({ id: "ann-4", pinned: false })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);

        // when
        await user.click(screen.getByRole("button", { name: "Pin" }));

        // then
        expect(mocks.pin).toHaveBeenCalledWith({ id: "ann-4", pinned: true });
    });

    it("unpins an announcement that is already pinned", async () => {
        // given
        stubAnnouncements([makeAnnouncement({ id: "ann-4", pinned: true })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);

        // when
        await user.click(screen.getByRole("button", { name: "Unpin" }));

        // then
        expect(mocks.pin).toHaveBeenCalledWith({ id: "ann-4", pinned: false });
    });

    it("reports why an announcement could not be pinned", async () => {
        // given
        stubAnnouncements([makeAnnouncement({ id: "ann-4", pinned: false })]);
        mocks.pin.mockRejectedValue(new Error("the golden land refuses"));
        const user = userEvent.setup();
        renderWithProviders(<AdminAnnouncements />);

        // when
        await user.click(screen.getByRole("button", { name: "Pin" }));

        // then
        expect(await screen.findByText("the golden land refuses")).toBeInTheDocument();
    });
});
