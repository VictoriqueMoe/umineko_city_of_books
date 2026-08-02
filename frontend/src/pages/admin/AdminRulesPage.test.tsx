import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { SiteSettings } from "../../types/api";
import { AdminRulesPage } from "./AdminRulesPage";

const mocks = vi.hoisted(() => ({
    useAdminSettings: vi.fn(),
    update: vi.fn(),
    isPending: false,
}));

vi.mock("../../api/queries/admin", () => ({ useAdminSettings: mocks.useAdminSettings }));

vi.mock("../../api/mutations/admin", () => ({
    useUpdateAdminSettings: () => ({ mutateAsync: mocks.update, isPending: mocks.isPending }),
}));

function stubSettings(settings: SiteSettings | null, loading = false) {
    mocks.useAdminSettings.mockReturnValue({ settings, loading, refresh: vi.fn() });
}

function editor(): HTMLTextAreaElement {
    return screen.getByPlaceholderText("Write the rules in Markdown...") as HTMLTextAreaElement;
}

beforeEach(() => {
    mocks.isPending = false;
    mocks.update.mockResolvedValue(undefined);
});

describe("AdminRulesPage", () => {
    it("waits while the saved rules page is being fetched", () => {
        // given
        stubSettings(null, true);

        // when
        renderWithProviders(<AdminRulesPage />);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("opens the editor on the rules that are already published", () => {
        // given
        stubSettings({ rules_page: "# House rules\n\nBe kind." });

        // when
        renderWithProviders(<AdminRulesPage />);

        // then
        expect(editor()).toHaveValue("# House rules\n\nBe kind.");
    });

    it("starts empty when no rules page has ever been written", () => {
        // given
        stubSettings({});

        // when
        renderWithProviders(<AdminRulesPage />);

        // then
        expect(editor()).toHaveValue("");
    });

    it("saves the edited rules alongside every other setting", async () => {
        // given
        stubSettings({ site_name: "When They Cry", rules_page: "old" });
        const user = userEvent.setup();
        renderWithProviders(<AdminRulesPage />);
        await user.clear(editor());
        await user.type(editor(), "new rules");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ site_name: "When They Cry", rules_page: "new rules" });
        expect(await screen.findByText("Saved")).toBeInTheDocument();
    });

    it("refuses to save while the settings themselves failed to load", async () => {
        // given
        stubSettings(null);
        const user = userEvent.setup();
        renderWithProviders(<AdminRulesPage />);

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("reports why the rules page could not be saved", async () => {
        // given
        stubSettings({ rules_page: "old" });
        mocks.update.mockRejectedValue(new Error("the settings are sealed"));
        const user = userEvent.setup();
        renderWithProviders(<AdminRulesPage />);

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(await screen.findByText("the settings are sealed")).toBeInTheDocument();
    });

    it("drops the feedback as soon as the rules are edited again", async () => {
        // given
        stubSettings({ rules_page: "old" });
        const user = userEvent.setup();
        renderWithProviders(<AdminRulesPage />);
        await user.click(screen.getByRole("button", { name: "Save" }));
        expect(await screen.findByText("Saved")).toBeInTheDocument();

        // when
        await user.type(editor(), "!");

        // then
        expect(screen.queryByText("Saved")).not.toBeInTheDocument();
    });

    it("renders the written markdown when the preview tab is chosen", async () => {
        // given
        stubSettings({ rules_page: "# House rules" });
        const user = userEvent.setup();
        renderWithProviders(<AdminRulesPage />);

        // when
        await user.click(screen.getByRole("button", { name: "Preview" }));

        // then
        expect(screen.getByRole("heading", { name: "House rules", level: 1 })).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Write the rules in Markdown...")).not.toBeInTheDocument();
    });

    it("goes back to the editor from the preview", async () => {
        // given
        stubSettings({ rules_page: "# House rules" });
        const user = userEvent.setup();
        renderWithProviders(<AdminRulesPage />);
        await user.click(screen.getByRole("button", { name: "Preview" }));

        // when
        await user.click(screen.getByRole("button", { name: "Write" }));

        // then
        expect(editor()).toBeInTheDocument();
    });

    it("locks the save control while a save is in flight", () => {
        // given
        stubSettings({ rules_page: "old" });
        mocks.isPending = true;

        // when
        renderWithProviders(<AdminRulesPage />);

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
    });
});
