import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BannedGiphyEntry } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { AdminBannedGifs } from "./AdminBannedGifs";

const mocks = vi.hoisted(() => ({
    useBannedGifs: vi.fn(),
    add: vi.fn(),
    remove: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({ useBannedGifs: mocks.useBannedGifs }));

vi.mock("../../api/mutations/admin", () => ({
    useAddBannedGif: () => ({ mutateAsync: mocks.add, isPending: false }),
    useRemoveBannedGif: () => ({ mutateAsync: mocks.remove, isPending: false }),
}));

function makeEntry(overrides: Partial<BannedGiphyEntry> = {}): BannedGiphyEntry {
    return {
        kind: "gif",
        value: "abc123",
        reason: "",
        created_at: "2026-01-02T00:00:00Z",
        ...overrides,
    };
}

function stubEntries(entries: BannedGiphyEntry[], loading = false) {
    mocks.useBannedGifs.mockReturnValue({ entries, loading, refresh: vi.fn() });
}

function urlInput(): HTMLElement {
    return screen.getByPlaceholderText("https://giphy.com/gifs/... or https://giphy.com/channel/...");
}

beforeEach(() => {
    mocks.add.mockResolvedValue(undefined);
    mocks.remove.mockResolvedValue(undefined);
});

describe("AdminBannedGifs", () => {
    it("waits while the banlist is being fetched", () => {
        // given
        stubEntries([], true);

        // when
        renderWithProviders(<AdminBannedGifs />);

        // then
        expect(screen.getByText("Loading banlist...")).toBeInTheDocument();
    });

    it("says so when the banlist is empty", () => {
        // given
        stubEntries([]);

        // when
        renderWithProviders(<AdminBannedGifs />);

        // then
        expect(screen.getByText("Nothing banned yet.")).toBeInTheDocument();
    });

    it("calls a single banned image a GIF", () => {
        // given
        stubEntries([makeEntry({ kind: "gif", value: "abc123", reason: "gore" })]);

        // when
        renderWithProviders(<AdminBannedGifs />);

        // then
        expect(screen.getByText("GIF")).toBeInTheDocument();
        expect(screen.getByText("abc123")).toBeInTheDocument();
        expect(screen.getByText("gore")).toBeInTheDocument();
    });

    it("calls a banned uploader a channel", () => {
        // given
        stubEntries([makeEntry({ kind: "user", value: "Larperine" })]);

        // when
        renderWithProviders(<AdminBannedGifs />);

        // then
        expect(screen.getByText("Channel")).toBeInTheDocument();
        expect(screen.getByText("Larperine")).toBeInTheDocument();
    });

    it("dashes out the reason when none was given", () => {
        // given
        stubEntries([makeEntry({ reason: "" })]);

        // when
        renderWithProviders(<AdminBannedGifs />);

        // then
        expect(screen.getByText("—")).toBeInTheDocument();
    });

    it("refuses to add anything without a URL or id", () => {
        // given
        stubEntries([]);

        // when
        renderWithProviders(<AdminBannedGifs />);

        // then
        expect(screen.getByRole("button", { name: "Add to banlist" })).toBeDisabled();
    });

    it("bans the trimmed URL along with the reason", async () => {
        // given
        stubEntries([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedGifs />);
        await user.type(urlInput(), "  https://giphy.com/gifs/thing-abc123  ");
        await user.type(screen.getByPlaceholderText("Why is this being banned?"), " gore ");

        // when
        await user.click(screen.getByRole("button", { name: "Add to banlist" }));

        // then
        expect(mocks.add).toHaveBeenCalledWith({ input: "https://giphy.com/gifs/thing-abc123", reason: "gore" });
    });

    it("clears the form once the ban is accepted", async () => {
        // given
        stubEntries([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedGifs />);
        await user.type(urlInput(), "https://giphy.com/gifs/thing-abc123");

        // when
        await user.click(screen.getByRole("button", { name: "Add to banlist" }));

        // then
        expect(urlInput()).toHaveValue("");
    });

    it("reports why something could not be banned", async () => {
        // given
        stubEntries([]);
        mocks.add.mockRejectedValue(new Error("that is not a Giphy link"));
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedGifs />);
        await user.type(urlInput(), "https://example.com/nope");

        // when
        await user.click(screen.getByRole("button", { name: "Add to banlist" }));

        // then
        expect(await screen.findByText("that is not a Giphy link")).toBeInTheDocument();
    });

    it("asks before lifting a ban", async () => {
        // given
        stubEntries([makeEntry()]);
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedGifs />);

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(confirm).toHaveBeenCalledWith('Remove gif "abc123" from the banlist?');
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("lifts the ban on both the kind and the value once confirmed", async () => {
        // given
        stubEntries([makeEntry({ kind: "user", value: "Larperine" })]);
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedGifs />);

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(mocks.remove).toHaveBeenCalledWith({ kind: "user", value: "Larperine" });
    });
});
