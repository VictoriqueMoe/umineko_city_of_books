import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { AdminUserItem } from "../../types/api";
import { AdminUsers } from "./AdminUsers";

const mocks = vi.hoisted(() => ({
    useAdminUsers: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({ useAdminUsers: mocks.useAdminUsers }));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

function makeAdminUserItem(overrides: Partial<AdminUserItem> = {}): AdminUserItem {
    return {
        id: "user-1",
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "",
        banned: false,
        locked: false,
        created_at: "2026-01-02T00:00:00Z",
        ...overrides,
    };
}

function stubUsers(users: AdminUserItem[], total = users.length, loading = false, error = false) {
    mocks.useAdminUsers.mockReturnValue({ users, total, loading, error, refresh: vi.fn() });
}

function renderPage() {
    return renderWithProviders(<AdminUsers />, { route: "/admin/users" });
}

describe("AdminUsers", () => {
    it("waits while the roster is still loading", () => {
        // given
        stubUsers([], 0, true);

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading users...")).toBeInTheDocument();
        expect(screen.queryByRole("table")).not.toBeInTheDocument();
    });

    it("reports a failed roster request instead of claiming there are no users", () => {
        // given
        stubUsers([], 0, false, true);

        // when
        renderPage();

        // then
        expect(screen.getByText("Could not load the user list.")).toBeInTheDocument();
        expect(screen.queryByText("No users found")).not.toBeInTheDocument();
    });

    it("says so when the search turned up nobody", () => {
        // given
        stubUsers([]);

        // when
        renderPage();

        // then
        expect(screen.getByText("No users found")).toBeInTheDocument();
    });

    it("lists a member with their username, display name and join date", () => {
        // given
        stubUsers([makeAdminUserItem()]);

        // when
        renderPage();

        // then
        expect(screen.getByText("beatrice")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText(new Date("2026-01-02T00:00:00Z").toLocaleDateString())).toBeInTheDocument();
    });

    it("separates banned members from active ones", () => {
        // given
        stubUsers([
            makeAdminUserItem({ id: "a", username: "beatrice", banned: false }),
            makeAdminUserItem({ id: "b", username: "erika", display_name: "Erika", banned: true }),
        ]);

        // when
        renderPage();

        // then
        expect(screen.getByText("Active")).toBeInTheDocument();
        expect(screen.getByText("Banned")).toBeInTheDocument();
    });

    it("shows the staff badge for a member who holds a site role", () => {
        // given
        stubUsers([makeAdminUserItem({ role: "moderator" })]);

        // when
        renderPage();

        // then
        expect(screen.getByText("Witch")).toBeInTheDocument();
    });

    it("falls back to an initial when a member has no avatar", () => {
        // given
        stubUsers([makeAdminUserItem({ avatar_url: "" })]);

        // when
        const { container } = renderPage();

        // then
        expect(screen.getByText("B")).toBeInTheDocument();
        expect(container.querySelector("img")).toBeNull();
    });

    it("opens the detail page for the member whose row was clicked", async () => {
        // given
        stubUsers([makeAdminUserItem({ id: "user-9" })]);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("beatrice"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/admin/users/user-9");
    });

    it("only asks the server for the typed search once the form is submitted", async () => {
        // given
        stubUsers([makeAdminUserItem()]);
        const user = userEvent.setup();
        const { container } = renderPage();

        // when
        await user.type(screen.getByPlaceholderText("Search users..."), "erika");

        // then
        expect(mocks.useAdminUsers).toHaveBeenLastCalledWith("", 20, 0);

        fireEvent.submit(container.querySelector("form") as HTMLFormElement);
        expect(mocks.useAdminUsers).toHaveBeenLastCalledWith("erika", 20, 0);
    });

    it("asks for the next page of members when the reader pages forward", async () => {
        // given
        stubUsers([makeAdminUserItem()], 45);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useAdminUsers).toHaveBeenLastCalledWith("", 20, 20);
    });

    it("never pages back past the first member", async () => {
        // given
        stubUsers([makeAdminUserItem()], 45);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(mocks.useAdminUsers).toHaveBeenLastCalledWith("", 20, 0);
    });
});
