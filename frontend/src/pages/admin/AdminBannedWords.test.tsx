import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { BannedWordRule } from "../../types/api";
import { AdminBannedWords } from "./AdminBannedWords";

const mocks = vi.hoisted(() => ({
    useGlobalBannedWords: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({ useGlobalBannedWords: mocks.useGlobalBannedWords }));

vi.mock("../../api/mutations/admin", () => ({
    useCreateGlobalBannedWord: () => ({ mutateAsync: mocks.create, isPending: false }),
    useUpdateGlobalBannedWord: () => ({ mutateAsync: mocks.update, isPending: false }),
    useDeleteGlobalBannedWord: () => ({ mutateAsync: mocks.remove, isPending: false }),
}));

function makeRule(overrides: Partial<BannedWordRule> = {}): BannedWordRule {
    return {
        id: "rule-1",
        scope: "global",
        pattern: "goat",
        match_mode: "substring",
        case_sensitive: false,
        action: "delete",
        created_by_name: "Virgilia",
        created_at: "2026-01-02T00:00:00Z",
        ...overrides,
    };
}

function stubRules(rules: BannedWordRule[], loading = false) {
    mocks.useGlobalBannedWords.mockReturnValue({ rules, loading, refresh: vi.fn() });
}

function patternInput(): HTMLElement {
    return screen.getByPlaceholderText("Word or regex to block");
}

function modeSelect(): HTMLElement {
    return screen.getAllByRole("combobox")[0];
}

function actionSelect(): HTMLElement {
    return screen.getAllByRole("combobox")[1];
}

beforeEach(() => {
    mocks.create.mockResolvedValue(undefined);
    mocks.update.mockResolvedValue(undefined);
    mocks.remove.mockResolvedValue(undefined);
});

describe("AdminBannedWords", () => {
    it("waits while the rules are being fetched", () => {
        // given
        stubRules([], true);

        // when
        renderWithProviders(<AdminBannedWords />);

        // then
        expect(screen.getByText("Loading rules...")).toBeInTheDocument();
    });

    it("says so when no global rule has been written yet", () => {
        // given
        stubRules([]);

        // when
        renderWithProviders(<AdminBannedWords />);

        // then
        expect(screen.getByText("No global rules yet.")).toBeInTheDocument();
    });

    it("lists a rule with its pattern, mode, case handling and author", () => {
        // given
        stubRules([makeRule({ match_mode: "whole_word", case_sensitive: true, action: "kick" })]);

        // when
        renderWithProviders(<AdminBannedWords />);

        // then
        expect(screen.getByText("goat")).toBeInTheDocument();
        expect(screen.getByText("whole_word")).toBeInTheDocument();
        expect(screen.getByText("Yes")).toBeInTheDocument();
        expect(screen.getByText("kick")).toBeInTheDocument();
        expect(screen.getByText("Virgilia")).toBeInTheDocument();
    });

    it("refuses to add a rule with no pattern", () => {
        // given
        stubRules([]);

        // when
        renderWithProviders(<AdminBannedWords />);

        // then
        expect(screen.getByRole("button", { name: "Add rule" })).toBeDisabled();
    });

    it("adds a rule with the default substring and delete behaviour", async () => {
        // given
        stubRules([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);
        await user.type(patternInput(), "  goat  ");

        // when
        await user.click(screen.getByRole("button", { name: "Add rule" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith({
            pattern: "goat",
            match_mode: "substring",
            case_sensitive: false,
            action: "delete",
        });
    });

    it("carries the chosen mode, action and case sensitivity into the new rule", async () => {
        // given
        stubRules([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);
        await user.type(patternInput(), "goat");
        await user.selectOptions(modeSelect(), "whole_word");
        await user.selectOptions(actionSelect(), "kick");
        await user.click(screen.getByRole("checkbox"));

        // when
        await user.click(screen.getByRole("button", { name: "Add rule" }));

        // then
        expect(mocks.create).toHaveBeenCalledWith({
            pattern: "goat",
            match_mode: "whole_word",
            case_sensitive: true,
            action: "kick",
        });
    });

    it("rejects a regex that cannot be compiled", async () => {
        // given
        stubRules([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);

        // when
        await user.type(patternInput(), "goat(");
        await user.selectOptions(modeSelect(), "regex");

        // then
        expect(screen.getByText(/^Regex error:/)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Add rule" })).toBeDisabled();
    });

    it("accepts a regex that compiles", async () => {
        // given
        stubRules([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);

        // when
        await user.type(patternInput(), "go+at");
        await user.selectOptions(modeSelect(), "regex");

        // then
        expect(screen.queryByText(/^Regex error:/)).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Add rule" })).toBeEnabled();
    });

    it("loads a rule into the form for editing", async () => {
        // given
        stubRules([makeRule({ pattern: "goat", action: "kick" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // then
        expect(patternInput()).toHaveValue("goat");
        expect(actionSelect()).toHaveValue("kick");
        expect(screen.getByRole("button", { name: "Save changes" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    });

    it("saves an edit against the rule it came from", async () => {
        // given
        stubRules([makeRule({ id: "rule-9" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);
        await user.click(screen.getByRole("button", { name: "Edit" }));
        await user.clear(patternInput());
        await user.type(patternInput(), "goats");

        // when
        await user.click(screen.getByRole("button", { name: "Save changes" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            ruleId: "rule-9",
            req: { pattern: "goats", match_mode: "substring", case_sensitive: false, action: "delete" },
        });
    });

    it("clears the form when an edit is abandoned", async () => {
        // given
        stubRules([makeRule()]);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(patternInput()).toHaveValue("");
        expect(screen.getByRole("button", { name: "Add rule" })).toBeInTheDocument();
    });

    it("reports why a rule could not be saved", async () => {
        // given
        stubRules([]);
        mocks.create.mockRejectedValue(new Error("that pattern is already banned"));
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);
        await user.type(patternInput(), "goat");

        // when
        await user.click(screen.getByRole("button", { name: "Add rule" }));

        // then
        expect(await screen.findByText("that pattern is already banned")).toBeInTheDocument();
    });

    it("asks before removing a rule", async () => {
        // given
        stubRules([makeRule()]);
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(confirm).toHaveBeenCalledWith('Remove global rule for pattern "goat"?');
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("removes the rule once confirmed", async () => {
        // given
        stubRules([makeRule({ id: "rule-3" })]);
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<AdminBannedWords />);

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(mocks.remove).toHaveBeenCalledWith("rule-3");
    });
});
