import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { AuditLogEntry } from "../../types/api";
import { AdminAuditLog } from "./AdminAuditLog";

const mocks = vi.hoisted(() => ({
    useAuditLog: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({ useAuditLog: mocks.useAuditLog }));

function makeEntry(overrides: Partial<AuditLogEntry> = {}): AuditLogEntry {
    return {
        id: 1,
        actor_id: "staff-1",
        actor_name: "Virgilia",
        action: "ban_user",
        target_type: "user",
        target_id: "aaaaaaaabbbbcccc",
        details: 'reason="spam in the parlour" duration=7',
        created_at: "2026-02-03T10:00:00Z",
        subject_id: "target-1",
        subject_name: "Battler",
        subject_username: "battler",
        ...overrides,
    };
}

function stubLog(entries: AuditLogEntry[], total = entries.length, loading = false) {
    mocks.useAuditLog.mockReturnValue({ entries, total, loading, refresh: vi.fn() });
}

function renderPage() {
    return renderWithProviders(<AdminAuditLog />, { route: "/admin/audit-log" });
}

describe("AdminAuditLog", () => {
    it("waits while the log is still being fetched", () => {
        // given
        stubLog([], 0, true);

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading audit log...")).toBeInTheDocument();
        expect(screen.queryByRole("table")).not.toBeInTheDocument();
    });

    it("says so when nothing matches the current filter", () => {
        // given
        stubLog([]);

        // when
        renderPage();

        // then
        expect(screen.getByText("No audit log entries found")).toBeInTheDocument();
    });

    it("names the action and the kind of thing it was aimed at", () => {
        // given
        stubLog([makeEntry()]);

        // when
        renderPage();

        // then
        const table = within(screen.getByRole("table"));
        expect(table.getByText("Banned")).toBeInTheDocument();
        expect(table.getByText("User")).toBeInTheDocument();
        expect(table.getByText(/aaaaaaaa\.\.\./)).toBeInTheDocument();
    });

    it("links the subject of the entry to their admin page", () => {
        // given
        stubLog([makeEntry()]);

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: "Battler" })).toHaveAttribute("href", "/admin/users/target-1");
    });

    it("falls back to linking the target id when the subject was not resolved", () => {
        // given
        stubLog([makeEntry({ subject_id: undefined, subject_name: undefined, subject_username: undefined })]);

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: "aaaaaaaa..." })).toHaveAttribute(
            "href",
            "/admin/users/aaaaaaaabbbbcccc",
        );
    });

    it("leaves the subject blank when the entry is not about a user", () => {
        // given
        stubLog([
            makeEntry({
                action: "update_settings",
                target_type: "settings",
                target_id: "",
                subject_id: undefined,
                subject_name: undefined,
                subject_username: undefined,
            }),
        ]);

        // when
        renderPage();

        // then
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
        expect(screen.getByText("—")).toBeInTheDocument();
    });

    it("credits an entry with no actor to the system", () => {
        // given
        stubLog([makeEntry({ actor_name: "" })]);

        // when
        renderPage();

        // then
        expect(screen.getByText("system")).toBeInTheDocument();
    });

    it("breaks the details out into labelled parts", () => {
        // given
        stubLog([makeEntry()]);

        // when
        renderPage();

        // then
        expect(screen.getByText("reason")).toBeInTheDocument();
        expect(screen.getByText("spam in the parlour")).toBeInTheDocument();
        expect(screen.getByText("duration")).toBeInTheDocument();
        expect(screen.getByText("7")).toBeInTheDocument();
    });

    it("shows a plain detail string that carries no key at all", () => {
        // given
        stubLog([makeEntry({ details: "manual intervention" })]);

        // when
        renderPage();

        // then
        expect(screen.getByText("manual intervention")).toBeInTheDocument();
    });

    it("offers every known action as a filter, sorted by its label", () => {
        // given
        stubLog([]);

        // when
        renderPage();

        // then
        const options = within(screen.getByRole("combobox")).getAllByRole("option");
        const labels: string[] = [];
        for (const option of options.slice(1)) {
            labels.push(option.textContent ?? "");
        }
        expect(labels[0]).toBe("Account created");
        expect([...labels].sort((a, b) => a.localeCompare(b))).toEqual(labels);
    });

    it("asks the server for only the chosen action", async () => {
        // given
        stubLog([makeEntry()]);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.selectOptions(screen.getByRole("combobox"), "ban_user");

        // then
        expect(mocks.useAuditLog).toHaveBeenLastCalledWith("ban_user", 50, 0);
    });

    it("returns to the first page whenever the filter changes", async () => {
        // given
        stubLog([makeEntry()], 300);
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Next" }));
        expect(mocks.useAuditLog).toHaveBeenLastCalledWith("", 50, 50);

        // when
        await user.selectOptions(screen.getByRole("combobox"), "ban_user");

        // then
        expect(mocks.useAuditLog).toHaveBeenLastCalledWith("ban_user", 50, 0);
    });
});
