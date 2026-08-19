import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { AdminUserDetail as AdminUserDetailType, AdminUserItem, AuditLogEntry } from "../../types/api";
import type { SiteRole } from "../../utils/permissions";
import { AdminUserDetail } from "./AdminUserDetail";

const mocks = vi.hoisted(() => ({
    useAdminUser: vi.fn(),
    useUserAuditLog: vi.fn(),
    useUserIPMatches: vi.fn(),
    navigate: vi.fn(),
    setRole: vi.fn(),
    removeRole: vi.fn(),
    ban: vi.fn(),
    unban: vi.fn(),
    lock: vi.fn(),
    unlock: vi.fn(),
    approve: vi.fn(),
    unapprove: vi.fn(),
    deleteUser: vi.fn(),
    detectiveScore: vi.fn(),
    gmScore: vi.fn(),
    resetPassword: vi.fn(),
    setEmail: vi.fn(),
    verifyEmail: vi.fn(),
    unverifyEmail: vi.fn(),
    setDisplayName: vi.fn(),
    setDisplayNameLock: vi.fn(),
    forceLogout: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({
    useAdminUser: mocks.useAdminUser,
    useUserAuditLog: mocks.useUserAuditLog,
    useUserIPMatches: mocks.useUserIPMatches,
}));

vi.mock("../../api/mutations/admin", () => ({
    useSetUserRole: () => ({ mutateAsync: mocks.setRole, isPending: false }),
    useRemoveUserRole: () => ({ mutateAsync: mocks.removeRole, isPending: false }),
    useBanUser: () => ({ mutateAsync: mocks.ban, isPending: false }),
    useUnbanUser: () => ({ mutateAsync: mocks.unban, isPending: false }),
    useLockUser: () => ({ mutateAsync: mocks.lock, isPending: false }),
    useUnlockUser: () => ({ mutateAsync: mocks.unlock, isPending: false }),
    useApproveUser: () => ({ mutateAsync: mocks.approve, isPending: false }),
    useUnapproveUser: () => ({ mutateAsync: mocks.unapprove, isPending: false }),
    useAdminDeleteUser: () => ({ mutateAsync: mocks.deleteUser, isPending: false }),
    useUpdateDetectiveScore: () => ({ mutateAsync: mocks.detectiveScore, isPending: false }),
    useUpdateGMScore: () => ({ mutateAsync: mocks.gmScore, isPending: false }),
    useResetUserPassword: () => ({ mutateAsync: mocks.resetPassword, isPending: false }),
    useSetUserEmail: () => ({ mutateAsync: mocks.setEmail, isPending: false }),
    useVerifyUserEmail: () => ({ mutateAsync: mocks.verifyEmail, isPending: false }),
    useUnverifyUserEmail: () => ({ mutateAsync: mocks.unverifyEmail, isPending: false }),
    useSetUserDisplayName: () => ({ mutateAsync: mocks.setDisplayName, isPending: false }),
    useSetDisplayNameLock: () => ({ mutateAsync: mocks.setDisplayNameLock, isPending: false }),
    useForceLogoutUser: () => ({ mutateAsync: mocks.forceLogout, isPending: false }),
}));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

const MUTATIONS = [
    mocks.setRole,
    mocks.removeRole,
    mocks.ban,
    mocks.unban,
    mocks.lock,
    mocks.unlock,
    mocks.approve,
    mocks.unapprove,
    mocks.deleteUser,
    mocks.detectiveScore,
    mocks.gmScore,
    mocks.resetPassword,
    mocks.setEmail,
    mocks.verifyEmail,
    mocks.unverifyEmail,
    mocks.setDisplayName,
    mocks.setDisplayNameLock,
    mocks.forceLogout,
];

function makeTarget(overrides: Partial<AdminUserDetailType> = {}): AdminUserDetailType {
    return {
        id: "target-1",
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        banned: false,
        locked: false,
        created_at: "2026-01-02T00:00:00Z",
        email: "battler@example.com",
        email_verified: true,
        display_name_locked: false,
        restricted: false,
        theory_count: 3,
        response_count: 5,
        mystery_score_adjustment: 0,
        detective_score: 12,
        gm_score_adjustment: 0,
        gm_score: 4,
        ...overrides,
    };
}

function makeHistoryEntry(overrides: Partial<AuditLogEntry> = {}): AuditLogEntry {
    return {
        id: 1,
        actor_id: "staff-1",
        actor_name: "Virgilia",
        action: "ban_user",
        target_type: "user",
        target_id: "target-1",
        details: 'reason="spam in the parlour"',
        created_at: "2026-02-03T10:00:00Z",
        ...overrides,
    };
}

function stubTarget(target: AdminUserDetailType | null, loading = false) {
    mocks.useAdminUser.mockReturnValue({ user: target, loading });
}

function stubIPMatches(overrides: { users?: AdminUserItem[]; loading?: boolean; failed?: boolean } = {}) {
    mocks.useUserIPMatches.mockReturnValue({
        ip: "203.0.113.9",
        users: overrides.users ?? [],
        loading: overrides.loading ?? false,
        failed: overrides.failed ?? false,
    });
}

function stubHistory(
    overrides: { entries?: AuditLogEntry[]; total?: number; loading?: boolean; failed?: boolean } = {},
) {
    mocks.useUserAuditLog.mockReturnValue({
        entries: overrides.entries ?? [],
        total: overrides.total ?? overrides.entries?.length ?? 0,
        loading: overrides.loading ?? false,
        failed: overrides.failed ?? false,
    });
}

function renderDetail(role: SiteRole = "admin") {
    return renderWithProviders(<AdminUserDetail />, {
        user: makeUser({ id: "staff-1", username: "virgilia", display_name: "Virgilia", role }),
        route: "/admin/users/target-1",
        path: "/admin/users/:id",
    });
}

function fieldFor(label: string): HTMLElement {
    const field = screen.getByText(label).closest("div");
    if (!field) {
        throw new Error(`no field wrapping ${label}`);
    }

    return field;
}

beforeEach(() => {
    stubIPMatches();
    stubHistory();
    for (const mutation of MUTATIONS) {
        mutation.mockResolvedValue(undefined);
    }
});

describe("AdminUserDetail loading and identity", () => {
    it("waits while the account is still being fetched", () => {
        // given
        stubTarget(null, true);

        // when
        renderDetail();

        // then
        expect(screen.getByText("Loading user...")).toBeInTheDocument();
    });

    it("says so when the account could not be loaded", () => {
        // given
        stubTarget(null);

        // when
        renderDetail();

        // then
        expect(screen.getByText("Could not load this user.")).toBeInTheDocument();
    });

    it("summarises the account with its email, counts and join date", () => {
        // given
        stubTarget(makeTarget());

        // when
        renderDetail();

        // then
        expect(screen.getByText("battler@example.com")).toBeInTheDocument();
        expect(screen.getByText("Verified")).toBeInTheDocument();
        expect(within(fieldFor("Theories")).getByText("3")).toBeInTheDocument();
        expect(within(fieldFor("Responses")).getByText("5")).toBeInTheDocument();
        expect(screen.getByText(new Date("2026-01-02T00:00:00Z").toLocaleDateString())).toBeInTheDocument();
    });

    it("says when an account has no email address at all", () => {
        // given
        stubTarget(makeTarget({ email: undefined }));

        // when
        renderDetail();

        // then
        expect(screen.getByText("No email set")).toBeInTheDocument();
    });

    it("spells out why a banned account was banned and by whom", () => {
        // given
        stubTarget(
            makeTarget({
                banned: true,
                ban_reason: "declared a red truth in bad faith",
                banned_at: "2026-03-01T00:00:00Z",
                banned_by: { id: "staff-1", username: "virgilia", display_name: "Virgilia" },
            }),
        );

        // when
        renderDetail();

        // then
        expect(screen.getByText("declared a red truth in bad faith")).toBeInTheDocument();
        expect(screen.getByText("Banned By")).toBeInTheDocument();
        expect(screen.getByText("Banned At")).toBeInTheDocument();
    });

    it("goes back to the roster from the back link", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail();

        // when
        await user.click(screen.getByText(/Back to Users/));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/admin/users");
    });
});

describe("AdminUserDetail permission gates", () => {
    it("gives a moderator the account tools but not the email or role ones", () => {
        // given
        stubTarget(makeTarget());

        // when
        renderDetail("moderator");

        // then
        expect(screen.getByRole("heading", { name: "Display Name" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Ban Management" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Lock Management" })).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Change Email" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Role" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Password" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Danger Zone" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Account History" })).not.toBeInTheDocument();
    });

    it("gives an admin the email, role, password and deletion tools too", () => {
        // given
        stubTarget(makeTarget());

        // when
        renderDetail("admin");

        // then
        expect(screen.getByRole("heading", { name: "Change Email" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Role" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Password" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Danger Zone" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Account History" })).toBeInTheDocument();
    });

    it("offers a super admin the same tools as an admin", () => {
        // given
        stubTarget(makeTarget());

        // when
        renderDetail("super_admin");

        // then
        expect(screen.getByRole("heading", { name: "Change Email" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Danger Zone" })).toBeInTheDocument();
    });

    it("offers no action at all against a super admin target", () => {
        // given
        stubTarget(makeTarget({ role: "super_admin" }));

        // when
        renderDetail("super_admin");

        // then
        expect(screen.queryByRole("heading", { name: "Change Email" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Display Name" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Mystery Scores" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Role" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Ban Management" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Lock Management" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Sessions" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Password" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Danger Zone" })).not.toBeInTheDocument();
    });

    it("keeps a moderator out of a super admin's mystery scores", () => {
        // given
        stubTarget(makeTarget({ role: "super_admin" }));

        // when
        renderDetail("moderator");

        // then
        expect(screen.queryByRole("heading", { name: "Mystery Scores" })).not.toBeInTheDocument();
        expect(screen.queryByText("Detective Score")).not.toBeInTheDocument();
        expect(screen.queryByText("Game Master Score")).not.toBeInTheDocument();
    });

    it("withholds the lock tool from an admin target while leaving the ban tool", () => {
        // given
        stubTarget(makeTarget({ role: "admin" }));

        // when
        renderDetail("super_admin");

        // then
        expect(screen.getByRole("heading", { name: "Ban Management" })).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Lock Management" })).not.toBeInTheDocument();
    });

    it("keeps the email verification toggle away from a moderator", () => {
        // given
        stubTarget(makeTarget({ email_verified: true }));

        // when
        renderDetail("moderator");

        // then
        expect(screen.queryByRole("button", { name: "Mark Unverified" })).not.toBeInTheDocument();
    });
});

describe("AdminUserDetail email management", () => {
    it("offers to mark a verified address unverified", () => {
        // given
        stubTarget(makeTarget({ email_verified: true }));

        // when
        renderDetail("admin");

        // then
        expect(screen.getByRole("button", { name: "Mark Unverified" })).toBeInTheDocument();
    });

    it("offers to mark an unverified address verified", async () => {
        // given
        stubTarget(makeTarget({ email_verified: false }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Mark Verified" }));

        // then
        expect(mocks.verifyEmail).toHaveBeenCalledWith("target-1");
        expect(await screen.findByText("Email marked as verified")).toBeInTheDocument();
    });

    it("warns before taking a verified address away", async () => {
        // given
        stubTarget(makeTarget({ email_verified: true }));
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Mark Unverified" }));

        // then
        expect(confirm).toHaveBeenCalled();
        expect(mocks.unverifyEmail).not.toHaveBeenCalled();
    });

    it("holds the email save back until the address actually changes", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");

        // then
        expect(screen.getByRole("button", { name: "Save Email" })).toBeDisabled();

        // when
        await user.clear(screen.getByPlaceholderText("user@example.com"));
        await user.type(screen.getByPlaceholderText("user@example.com"), "beato@example.com");

        // then
        expect(screen.getByRole("button", { name: "Save Email" })).toBeEnabled();
    });

    it("sends the trimmed new address and confirms the verification mail", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");
        await user.clear(screen.getByPlaceholderText("user@example.com"));
        await user.type(screen.getByPlaceholderText("user@example.com"), "  beato@example.com  ");

        // when
        await user.click(screen.getByRole("button", { name: "Save Email" }));

        // then
        expect(mocks.setEmail).toHaveBeenCalledWith({ id: "target-1", email: "beato@example.com" });
        expect(
            await screen.findByText("Email updated. A verification link was sent to the new address."),
        ).toBeInTheDocument();
    });

    it("reports why an email change was refused", async () => {
        // given
        stubTarget(makeTarget());
        mocks.setEmail.mockRejectedValue(new Error("that address is already spoken for"));
        const user = userEvent.setup();
        renderDetail("admin");
        await user.clear(screen.getByPlaceholderText("user@example.com"));
        await user.type(screen.getByPlaceholderText("user@example.com"), "beato@example.com");

        // when
        await user.click(screen.getByRole("button", { name: "Save Email" }));

        // then
        expect(await screen.findByText("that address is already spoken for")).toBeInTheDocument();
    });
});

describe("AdminUserDetail display name management", () => {
    it("holds the name save back until the display name actually changes", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");

        // then
        expect(screen.getByRole("button", { name: "Save Name" })).toBeDisabled();

        // when
        await user.clear(screen.getByPlaceholderText("Display name"));
        await user.type(screen.getByPlaceholderText("Display name"), "Endless Sorcerer");

        // then
        expect(screen.getByRole("button", { name: "Save Name" })).toBeEnabled();
    });

    it("sends the trimmed display name", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");
        await user.clear(screen.getByPlaceholderText("Display name"));
        await user.type(screen.getByPlaceholderText("Display name"), " Endless Sorcerer ");

        // when
        await user.click(screen.getByRole("button", { name: "Save Name" }));

        // then
        expect(mocks.setDisplayName).toHaveBeenCalledWith({ id: "target-1", displayName: "Endless Sorcerer" });
        expect(await screen.findByText("Display name updated")).toBeInTheDocument();
    });

    it("locks an unlocked display name", async () => {
        // given
        stubTarget(makeTarget({ display_name_locked: false }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Lock Name" }));

        // then
        expect(mocks.setDisplayNameLock).toHaveBeenCalledWith({ id: "target-1", locked: true });
        expect(await screen.findByText("Display name locked")).toBeInTheDocument();
    });

    it("unlocks a locked display name", async () => {
        // given
        stubTarget(makeTarget({ display_name_locked: true }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Unlock Name" }));

        // then
        expect(mocks.setDisplayNameLock).toHaveBeenCalledWith({ id: "target-1", locked: false });
        expect(await screen.findByText("Display name unlocked")).toBeInTheDocument();
    });
});

describe("AdminUserDetail roles, bans and locks", () => {
    it("assigns the chosen role to an account with none", async () => {
        // given
        stubTarget(makeTarget({ role: undefined }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.selectOptions(screen.getByRole("combobox"), "moderator");
        await user.click(screen.getByRole("button", { name: "Assign Role" }));

        // then
        expect(mocks.setRole).toHaveBeenCalledWith({ id: "target-1", role: "moderator" });
        expect(await screen.findByText("Role assigned")).toBeInTheDocument();
    });

    it("removes the role an account already holds", async () => {
        // given
        stubTarget(makeTarget({ role: "moderator" }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Remove Role" }));

        // then
        expect(mocks.removeRole).toHaveBeenCalledWith({ id: "target-1", role: "moderator" });
        expect(await screen.findByText("Role removed")).toBeInTheDocument();
    });

    it("refuses to ban without a reason", () => {
        // given
        stubTarget(makeTarget());

        // when
        renderDetail("admin");

        // then
        expect(screen.getByRole("button", { name: "Ban User" })).toBeDisabled();
    });

    it("bans with the trimmed reason and clears the field", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");
        await user.type(screen.getByPlaceholderText("Reason for ban..."), "  goat butchery  ");

        // when
        await user.click(screen.getByRole("button", { name: "Ban User" }));

        // then
        expect(mocks.ban).toHaveBeenCalledWith({ id: "target-1", reason: "goat butchery" });
        expect(await screen.findByText("User banned")).toBeInTheDocument();
    });

    it("reports why a ban was refused", async () => {
        // given
        stubTarget(makeTarget());
        mocks.ban.mockRejectedValue(new Error("the witch forbids it"));
        const user = userEvent.setup();
        renderDetail("admin");
        await user.type(screen.getByPlaceholderText("Reason for ban..."), "goat butchery");

        // when
        await user.click(screen.getByRole("button", { name: "Ban User" }));

        // then
        expect(await screen.findByText("the witch forbids it")).toBeInTheDocument();
    });

    it("offers to unban an account that is already banned", async () => {
        // given
        stubTarget(makeTarget({ banned: true }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Unban User" }));

        // then
        expect(mocks.unban).toHaveBeenCalledWith("target-1");
        expect(await screen.findByText("User unbanned")).toBeInTheDocument();
    });

    it("locks with the trimmed reason", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");
        await user.type(screen.getByPlaceholderText("Reason for lock..."), " noisy furniture ");

        // when
        await user.click(screen.getByRole("button", { name: "Lock User" }));

        // then
        expect(mocks.lock).toHaveBeenCalledWith({ id: "target-1", reason: "noisy furniture" });
        expect(await screen.findByText("User locked")).toBeInTheDocument();
    });

    it("offers to unlock an account that is already locked", async () => {
        // given
        stubTarget(makeTarget({ locked: true, lock_reason: "noisy furniture" }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Unlock User" }));

        // then
        expect(mocks.unlock).toHaveBeenCalledWith("target-1");
        expect(await screen.findByText("User unlocked")).toBeInTheDocument();
    });
});

describe("AdminUserDetail new account approval", () => {
    it("offers to approve an account still inside the restriction window", async () => {
        // given a member who signed up today
        stubTarget(makeTarget({ restricted: true }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Approve Account" }));

        // then
        expect(mocks.approve).toHaveBeenCalledWith("target-1");
        expect(await screen.findByText("Account approved, restrictions lifted")).toBeInTheDocument();
    });

    it("marks a restricted account on the summary", () => {
        // given
        stubTarget(makeTarget({ restricted: true }));

        // when
        renderDetail("admin");

        // then
        expect(within(fieldFor("New Account")).getByText("Restricted")).toBeInTheDocument();
    });

    it("offers to revoke an approval that was already granted", async () => {
        // given
        stubTarget(makeTarget({ restricted: false, approved_at: "2026-02-03T10:00:00Z" }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Revoke Approval" }));

        // then
        expect(mocks.unapprove).toHaveBeenCalledWith("target-1");
        expect(await screen.findByText("Approval revoked")).toBeInTheDocument();
    });

    it("names who approved the account, so the decision is attributable", () => {
        // given
        stubTarget(
            makeTarget({
                approved_at: "2026-02-03T10:00:00Z",
                approved_by: { id: "staff-9", username: "ronove", display_name: "Ronove" },
            }),
        );

        // when
        renderDetail("admin");

        // then
        expect(within(fieldFor("Approved By")).getByText("Ronove")).toBeInTheDocument();
    });

    it("stays out of the way for an established account that was never approved", () => {
        // given a member well past the window
        stubTarget(makeTarget({ restricted: false }));

        // when
        renderDetail("admin");

        // then there is nothing to approve, so the card is not shown at all
        expect(screen.queryByText("New Account Restriction")).not.toBeInTheDocument();
    });

    it("lets a moderator approve, not just an admin", async () => {
        // given moderators carry manage_user_account, and they are the ones watching new joins
        stubTarget(makeTarget({ restricted: true }));
        const user = userEvent.setup();
        renderDetail("moderator");

        // when
        await user.click(screen.getByRole("button", { name: "Approve Account" }));

        // then
        expect(mocks.approve).toHaveBeenCalledWith("target-1");
    });

    it("never offers to approve a super admin", () => {
        // given a protected target, which every other account action also refuses
        stubTarget(makeTarget({ restricted: true, role: "super_admin" }));

        // when
        renderDetail("admin");

        // then
        expect(screen.queryByRole("button", { name: "Approve Account" })).not.toBeInTheDocument();
    });
});

describe("AdminUserDetail sessions, password and deletion", () => {
    it("warns before revoking every session", async () => {
        // given
        stubTarget(makeTarget());
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Revoke All Sessions" }));

        // then
        expect(mocks.forceLogout).not.toHaveBeenCalled();
    });

    it("revokes every session once the warning is accepted", async () => {
        // given
        stubTarget(makeTarget());
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Revoke All Sessions" }));

        // then
        expect(mocks.forceLogout).toHaveBeenCalledWith("target-1");
        expect(await screen.findByText("All sessions revoked")).toBeInTheDocument();
    });

    it("shows the freshly generated password once and only once", async () => {
        // given
        stubTarget(makeTarget());
        mocks.resetPassword.mockResolvedValue({ password: "kakera-golden-77" });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Reset Password" }));

        // then
        expect(mocks.resetPassword).toHaveBeenCalledWith("target-1");
        expect(await screen.findByText("kakera-golden-77")).toBeInTheDocument();
    });

    it("leaves the password alone when the warning is dismissed", async () => {
        // given
        stubTarget(makeTarget());
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Reset Password" }));

        // then
        expect(mocks.resetPassword).not.toHaveBeenCalled();
    });

    it("asks once in the delete dialogue and then returns to the roster", async () => {
        // given
        stubTarget(makeTarget());
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Delete User" }));
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).not.toHaveBeenCalled();
        expect(mocks.deleteUser).toHaveBeenCalledWith("target-1");
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/admin/users");
        });
    });

    it("keeps the account when the delete dialogue is cancelled", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Delete User" }));
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(mocks.deleteUser).not.toHaveBeenCalled();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });
});

describe("AdminUserDetail mystery scores", () => {
    it("prefills both scores from the account", () => {
        // given
        stubTarget(makeTarget({ detective_score: 12, gm_score: 4 }));

        // when
        renderDetail("admin");

        // then
        expect(within(fieldFor("Detective Score")).getByRole("textbox")).toHaveValue("12");
        expect(within(fieldFor("Game Master Score")).getByRole("textbox")).toHaveValue("4");
    });

    it("saves the detective score the moderator typed", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("moderator");
        const field = fieldFor("Detective Score");
        await user.clear(within(field).getByRole("textbox"));
        await user.type(within(field).getByRole("textbox"), "35");

        // when
        await user.click(within(field).getByRole("button", { name: "Save" }));

        // then
        expect(mocks.detectiveScore).toHaveBeenCalledWith({ id: "target-1", desiredScore: 35 });
    });

    it("saves the game master score the moderator typed", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("moderator");
        const field = fieldFor("Game Master Score");
        await user.clear(within(field).getByRole("textbox"));
        await user.type(within(field).getByRole("textbox"), "-6");

        // when
        await user.click(within(field).getByRole("button", { name: "Save" }));

        // then
        expect(mocks.gmScore).toHaveBeenCalledWith({ id: "target-1", desiredScore: -6 });
    });

    it("ignores anything that is not a whole number", async () => {
        // given
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");
        const field = fieldFor("Detective Score");

        // when
        await user.clear(within(field).getByRole("textbox"));
        await user.type(within(field).getByRole("textbox"), "1a2");

        // then
        expect(within(field).getByRole("textbox")).toHaveValue("12");
    });
});

describe("AdminUserDetail shared IP addresses", () => {
    it("leaves the IP section out when the account has no recorded address", () => {
        // given
        stubTarget(makeTarget({ ip: undefined }));

        // when
        renderDetail("admin");

        // then
        expect(screen.queryByRole("heading", { name: "Other Accounts On This IP" })).not.toBeInTheDocument();
    });

    it("says when nobody else shares the address", () => {
        // given
        stubTarget(makeTarget({ ip: "203.0.113.9" }));
        stubIPMatches({ users: [] });

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText("No other accounts share this IP address.")).toBeInTheDocument();
    });

    it("reports when the shared address lookup failed", () => {
        // given
        stubTarget(makeTarget({ ip: "203.0.113.9" }));
        stubIPMatches({ failed: true });

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText("Could not load accounts for this IP address.")).toBeInTheDocument();
    });

    it("lets staff jump straight to another account on the same address", async () => {
        // given
        stubTarget(makeTarget({ ip: "203.0.113.9" }));
        stubIPMatches({
            users: [
                {
                    id: "alt-1",
                    username: "erika",
                    display_name: "Erika",
                    avatar_url: "",
                    banned: true,
                    locked: false,
                    created_at: "2026-01-05T00:00:00Z",
                },
            ],
        });
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Manage" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/admin/users/alt-1");
    });
});

describe("AdminUserDetail account history", () => {
    it("waits while the history is loading", () => {
        // given
        stubTarget(makeTarget());
        stubHistory({ loading: true });

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("says when nothing has ever been recorded", () => {
        // given
        stubTarget(makeTarget());
        stubHistory({ entries: [] });

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText("Nothing has been recorded against this account.")).toBeInTheDocument();
    });

    it("reports when the history could not be loaded", () => {
        // given
        stubTarget(makeTarget());
        stubHistory({ failed: true });

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText("Could not load the account history.")).toBeInTheDocument();
    });

    it("labels each recorded action and unpacks its details", () => {
        // given
        stubTarget(makeTarget());
        stubHistory({ entries: [makeHistoryEntry()] });

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText("Banned")).toBeInTheDocument();
        expect(screen.getByText("reason")).toBeInTheDocument();
        expect(screen.getByText("spam in the parlour")).toBeInTheDocument();
        expect(screen.getByText("Virgilia")).toBeInTheDocument();
    });

    it("credits an entry with no actor to the system", () => {
        // given
        stubTarget(makeTarget());
        stubHistory({ entries: [makeHistoryEntry({ actor_name: "", action: "user_created", details: "" })] });

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText("Account created")).toBeInTheDocument();
        expect(screen.getByText("system")).toBeInTheDocument();
    });

    it("pages the history forward ten entries at a time", async () => {
        // given
        stubTarget(makeTarget());
        stubHistory({ entries: [makeHistoryEntry()], total: 25 });
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(mocks.useUserAuditLog).toHaveBeenLastCalledWith("target-1", true, 10, 10);
    });
});
