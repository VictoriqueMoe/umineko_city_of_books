import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { SiteSettings } from "../../types/api";
import { AdminContentRules } from "./AdminContentRules";

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

function ruleBoxes(): HTMLTextAreaElement[] {
    return screen.getAllByPlaceholderText("Enter rules for this section...") as HTMLTextAreaElement[];
}

beforeEach(() => {
    mocks.isPending = false;
    mocks.update.mockResolvedValue(undefined);
});

describe("AdminContentRules", () => {
    it("waits while the saved rules are being fetched", () => {
        // given
        stubSettings(null, true);

        // when
        renderWithProviders(<AdminContentRules />);

        // then
        expect(screen.getByText("Loading rules...")).toBeInTheDocument();
    });

    it("gives every rules section its own box", () => {
        // given
        stubSettings({});

        // when
        renderWithProviders(<AdminContentRules />);

        // then
        expect(screen.getByRole("heading", { name: "Welcome (Landing)" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Theories (Higurashi)" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Game Board (Rose Guns Days)" })).toBeInTheDocument();
        expect(ruleBoxes()).toHaveLength(22);
    });

    it("prefills each box from the saved rules", () => {
        // given
        stubSettings({ rules_landing: "Be kind to the furniture." });

        // when
        renderWithProviders(<AdminContentRules />);

        // then
        expect(ruleBoxes()[0]).toHaveValue("Be kind to the furniture.");
        expect(ruleBoxes()[1]).toHaveValue("");
    });

    it("saves the edited rules on top of everything already stored", async () => {
        // given
        stubSettings({ site_name: "When They Cry", rules_landing: "Old rules" });
        const user = userEvent.setup();
        renderWithProviders(<AdminContentRules />);
        await user.clear(ruleBoxes()[0]);
        await user.type(ruleBoxes()[0], "New rules");

        // when
        await user.click(screen.getByRole("button", { name: "Save Rules" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ site_name: "When They Cry", rules_landing: "New rules" });
    });

    it("confirms the save", async () => {
        // given
        stubSettings({});
        const user = userEvent.setup();
        renderWithProviders(<AdminContentRules />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Rules" }));

        // then
        expect(await screen.findByText("Rules saved successfully")).toBeInTheDocument();
    });

    it("reports why the rules could not be saved", async () => {
        // given
        stubSettings({});
        mocks.update.mockRejectedValue(new Error("the settings are sealed"));
        const user = userEvent.setup();
        renderWithProviders(<AdminContentRules />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Rules" }));

        // then
        expect(await screen.findByText("the settings are sealed")).toBeInTheDocument();
    });

    it("drops the confirmation as soon as the rules are edited again", async () => {
        // given
        stubSettings({});
        const user = userEvent.setup();
        renderWithProviders(<AdminContentRules />);
        await user.click(screen.getByRole("button", { name: "Save Rules" }));
        expect(await screen.findByText("Rules saved successfully")).toBeInTheDocument();

        // when
        await user.type(ruleBoxes()[0], "a");

        // then
        expect(screen.queryByText("Rules saved successfully")).not.toBeInTheDocument();
    });

    it("locks the save control while a save is in flight", () => {
        // given
        stubSettings({});
        mocks.isPending = true;

        // when
        renderWithProviders(<AdminContentRules />);

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
    });
});
