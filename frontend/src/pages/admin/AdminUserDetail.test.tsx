import { screen, waitFor, within } from "@testing-library/react";
import userEvent, { type UserEvent } from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { AdminUserDetail as AdminUserDetailType, AdminUserItem, AuditLogEntry, SiteRole } from "../../types/api";
import { AdminUserDetail } from "./AdminUserDetail";

const mocks = vi.hoisted(() => ({
    useAdminUser: vi.fn(),
    useUserAuditLog: vi.fn(),
    useUserIPMatches: vi.fn(),
    navigate: vi.fn(),
}));

const mutations = vi.hoisted(() => ({
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

vi.mock("../../hooks/queries/admin", () => ({
    useAdminUser: mocks.useAdminUser,
    useUserAuditLog: mocks.useUserAuditLog,
    useUserIPMatches: mocks.useUserIPMatches,
}));

vi.mock("../../hooks/mutations/admin", () => ({
    useSetUserRole: () => ({ mutateAsync: mutations.setRole, isPending: false }),
    useRemoveUserRole: () => ({ mutateAsync: mutations.removeRole, isPending: false }),
    useBanUser: () => ({ mutateAsync: mutations.ban, isPending: false }),
    useUnbanUser: () => ({ mutateAsync: mutations.unban, isPending: false }),
    useLockUser: () => ({ mutateAsync: mutations.lock, isPending: false }),
    useUnlockUser: () => ({ mutateAsync: mutations.unlock, isPending: false }),
    useApproveUser: () => ({ mutateAsync: mutations.approve, isPending: false }),
    useUnapproveUser: () => ({ mutateAsync: mutations.unapprove, isPending: false }),
    useAdminDeleteUser: () => ({ mutateAsync: mutations.deleteUser, isPending: false }),
    useUpdateDetectiveScore: () => ({ mutateAsync: mutations.detectiveScore, isPending: false }),
    useUpdateGMScore: () => ({ mutateAsync: mutations.gmScore, isPending: false }),
    useResetUserPassword: () => ({ mutateAsync: mutations.resetPassword, isPending: false }),
    useSetUserEmail: () => ({ mutateAsync: mutations.setEmail, isPending: false }),
    useVerifyUserEmail: () => ({ mutateAsync: mutations.verifyEmail, isPending: false }),
    useUnverifyUserEmail: () => ({ mutateAsync: mutations.unverifyEmail, isPending: false }),
    useSetUserDisplayName: () => ({ mutateAsync: mutations.setDisplayName, isPending: false }),
    useSetDisplayNameLock: () => ({ mutateAsync: mutations.setDisplayNameLock, isPending: false }),
    useForceLogoutUser: () => ({ mutateAsync: mutations.forceLogout, isPending: false }),
}));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

const ADMIN_TOOLS = ["Change Email", "Role", "Password", "Danger Zone", "Account History"];

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

async function retype(user: UserEvent, box: HTMLElement, text: string) {
    await user.clear(box);
    await user.type(box, text);
}

beforeEach(() => {
    stubIPMatches();
    stubHistory();
    for (const mutation of Object.values(mutations)) {
        mutation.mockResolvedValue(undefined);
    }
});

describe("AdminUserDetail loading and identity", () => {
    const pendingCases = [
        { name: "waits while the account is still being fetched", loading: true, shown: "Loading user..." },
        { name: "says so when the account could not be loaded", loading: false, shown: "Could not load this user." },
    ];

    it.each(pendingCases)("$name", ({ loading, shown }) => {
        // given no account yet, and whether it is still loading, from the table row
        stubTarget(null, loading);

        // when
        renderDetail();

        // then
        expect(screen.getByText(shown)).toBeInTheDocument();
    });

    const summaryCases: { name: string; target: Partial<AdminUserDetailType>; fields: [string, string][] }[] = [
        {
            name: "summarises the account with its email, counts and join date",
            target: {},
            fields: [
                ["Email", "battler@example.com"],
                ["Email", "Verified"],
                ["Theories", "3"],
                ["Responses", "5"],
                ["Joined", new Date("2026-01-02T00:00:00Z").toLocaleDateString()],
            ],
        },
        {
            name: "says when an account has no email address at all",
            target: { email: undefined },
            fields: [["Email", "No email set"]],
        },
        {
            name: "spells out why a banned account was banned and by whom",
            target: {
                banned: true,
                ban_reason: "declared a red truth in bad faith",
                banned_at: "2026-03-01T00:00:00Z",
                banned_by: { id: "staff-1", username: "virgilia", display_name: "Virgilia" },
            },
            fields: [
                ["Ban Reason", "declared a red truth in bad faith"],
                ["Banned By", "Virgilia"],
                ["Banned At", new Date("2026-03-01T00:00:00Z").toLocaleDateString()],
            ],
        },
        {
            name: "marks a restricted account on the summary",
            target: { restricted: true },
            fields: [["New Account", "Restricted"]],
        },
        {
            name: "names who approved the account, so the decision is attributable",
            target: {
                approved_at: "2026-02-03T10:00:00Z",
                approved_by: { id: "staff-9", username: "ronove", display_name: "Ronove" },
            },
            fields: [["Approved By", "Ronove"]],
        },
    ];

    it.each(summaryCases)("$name", ({ target, fields }) => {
        // given the account and the summary fields it should fill, from the table row
        stubTarget(makeTarget(target));

        // when
        renderDetail("admin");

        // then
        for (const [label, value] of fields) {
            expect(within(fieldFor(label)).getByText(value)).toBeInTheDocument();
        }
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

describe("AdminUserDetail section gates", () => {
    interface GateCase {
        name: string;
        actor: SiteRole;
        target: Partial<AdminUserDetailType>;
        shown?: string[];
        hidden?: string[];
        hiddenButtons?: string[];
        hiddenText?: string[];
    }

    const gateCases: GateCase[] = [
        {
            name: "gives a moderator the account tools but not the email or role ones",
            actor: "moderator",
            target: {},
            shown: ["Display Name", "Ban Management", "Lock Management"],
            hidden: ADMIN_TOOLS,
        },
        {
            name: "gives an admin the email, role, password and deletion tools too",
            actor: "admin",
            target: {},
            shown: ADMIN_TOOLS,
        },
        {
            name: "offers a super admin the same tools as an admin",
            actor: "super_admin",
            target: {},
            shown: ADMIN_TOOLS,
        },
        {
            name: "offers no action at all against a super admin target",
            actor: "super_admin",
            target: { role: "super_admin" },
            hidden: [
                "Change Email",
                "Display Name",
                "Mystery Scores",
                "Role",
                "Ban Management",
                "Lock Management",
                "Sessions",
                "Password",
                "Danger Zone",
            ],
        },
        {
            name: "keeps a moderator out of a super admin's mystery scores",
            actor: "moderator",
            target: { role: "super_admin" },
            hidden: ["Mystery Scores"],
            hiddenText: ["Detective Score", "Game Master Score"],
        },
        {
            name: "withholds the lock tool from an admin target while leaving the ban tool",
            actor: "super_admin",
            target: { role: "admin" },
            shown: ["Ban Management"],
            hidden: ["Lock Management"],
        },
        {
            name: "keeps the email verification toggle away from a moderator",
            actor: "moderator",
            target: { email_verified: true },
            hiddenButtons: ["Mark Unverified"],
        },
        {
            name: "stays out of the way for an established account well past the window that was never approved, since there is nothing to approve",
            actor: "admin",
            target: { restricted: false },
            hiddenText: ["New Account Restriction"],
        },
        {
            name: "never offers to approve a super admin, a protected target every other account action also refuses",
            actor: "admin",
            target: { restricted: true, role: "super_admin" },
            hiddenButtons: ["Approve Account"],
        },
        {
            name: "leaves the IP section out when the account has no recorded address",
            actor: "admin",
            target: { ip: undefined },
            hidden: ["Other Accounts On This IP"],
        },
    ];

    it.each(gateCases)("$name", ({ actor, target, shown = [], hidden = [], hiddenButtons = [], hiddenText = [] }) => {
        // given the staff member and the account they are looking at, from the table row
        stubTarget(makeTarget(target));

        // when
        renderDetail(actor);

        // then
        for (const heading of shown) {
            expect(screen.getByRole("heading", { name: heading })).toBeInTheDocument();
        }
        for (const heading of hidden) {
            expect(screen.queryByRole("heading", { name: heading })).not.toBeInTheDocument();
        }
        for (const button of hiddenButtons) {
            expect(screen.queryByRole("button", { name: button })).not.toBeInTheDocument();
        }
        for (const text of hiddenText) {
            expect(screen.queryByText(text)).not.toBeInTheDocument();
        }
    });
});

describe("AdminUserDetail account actions", () => {
    interface ClickCase {
        name: string;
        actor?: SiteRole;
        target: Partial<AdminUserDetailType>;
        clicks: string[];
        mutation: Mock;
        reply?: unknown;
        sent: unknown;
        shown: string;
    }

    const clickCases: ClickCase[] = [
        {
            name: "offers to mark an unverified address verified",
            target: { email_verified: false },
            clicks: ["Mark Verified"],
            mutation: mutations.verifyEmail,
            sent: "target-1",
            shown: "Email marked as verified",
        },
        {
            name: "takes the verification away once the warning is accepted",
            target: { email_verified: true },
            clicks: ["Mark Unverified", "Unverify"],
            mutation: mutations.unverifyEmail,
            sent: "target-1",
            shown: "Email marked as unverified",
        },
        {
            name: "locks an unlocked display name",
            target: { display_name_locked: false },
            clicks: ["Lock Name"],
            mutation: mutations.setDisplayNameLock,
            sent: { id: "target-1", locked: true },
            shown: "Display name locked",
        },
        {
            name: "unlocks a locked display name",
            target: { display_name_locked: true },
            clicks: ["Unlock Name"],
            mutation: mutations.setDisplayNameLock,
            sent: { id: "target-1", locked: false },
            shown: "Display name unlocked",
        },
        {
            name: "removes the role an account already holds",
            target: { role: "moderator" },
            clicks: ["Remove Role"],
            mutation: mutations.removeRole,
            sent: { id: "target-1", role: "moderator" },
            shown: "Role removed",
        },
        {
            name: "offers to unban an account that is already banned",
            target: { banned: true },
            clicks: ["Unban User"],
            mutation: mutations.unban,
            sent: "target-1",
            shown: "User unbanned",
        },
        {
            name: "offers to unlock an account that is already locked",
            target: { locked: true, lock_reason: "noisy furniture" },
            clicks: ["Unlock User"],
            mutation: mutations.unlock,
            sent: "target-1",
            shown: "User unlocked",
        },
        {
            name: "offers to approve a member who signed up today, still inside the restriction window",
            target: { restricted: true },
            clicks: ["Approve Account"],
            mutation: mutations.approve,
            sent: "target-1",
            shown: "Account approved, restrictions lifted",
        },
        {
            name: "lets a moderator approve, not just an admin, since moderators carry manage_user_account and watch new joins",
            actor: "moderator",
            target: { restricted: true },
            clicks: ["Approve Account"],
            mutation: mutations.approve,
            sent: "target-1",
            shown: "Account approved, restrictions lifted",
        },
        {
            name: "offers to revoke an approval that was already granted",
            target: { restricted: false, approved_at: "2026-02-03T10:00:00Z" },
            clicks: ["Revoke Approval"],
            mutation: mutations.unapprove,
            sent: "target-1",
            shown: "Approval revoked",
        },
        {
            name: "revokes every session once the warning is accepted",
            target: {},
            clicks: ["Revoke All Sessions", "Revoke Sessions"],
            mutation: mutations.forceLogout,
            sent: "target-1",
            shown: "All sessions revoked",
        },
        {
            name: "shows the freshly generated password once and only once",
            target: {},
            clicks: ["Reset Password", "Reset"],
            mutation: mutations.resetPassword,
            reply: { password: "kakera-golden-77" },
            sent: "target-1",
            shown: "kakera-golden-77",
        },
    ];

    it.each(clickCases)("$name", async ({ actor, target, clicks, mutation, reply, sent, shown }) => {
        // given the account, the buttons to press and what the server answers, from the table row
        stubTarget(makeTarget(target));
        mutation.mockResolvedValue(reply);
        const user = userEvent.setup();
        renderDetail(actor);

        // when
        for (const label of clicks) {
            await user.click(screen.getByRole("button", { name: label }));
        }

        // then
        expect(mutation).toHaveBeenCalledWith(sent);
        expect(await screen.findByText(shown)).toBeInTheDocument();
    });

    const saveCases = [
        {
            name: "holds the email save back until the address changes, then sends it trimmed",
            section: "Change Email",
            typed: "  beato@example.com  ",
            button: "Save Email",
            mutation: mutations.setEmail,
            sent: { id: "target-1", email: "beato@example.com" },
            shown: "Email updated. A verification link was sent to the new address.",
            settled: "battler@example.com",
        },
        {
            name: "holds the name save back until the display name changes, then sends it trimmed",
            section: "Display Name",
            typed: " Endless Sorcerer ",
            button: "Save Name",
            mutation: mutations.setDisplayName,
            sent: { id: "target-1", displayName: "Endless Sorcerer" },
            shown: "Display name updated",
            settled: "Battler",
        },
        {
            name: "refuses to ban without a reason, then bans with the trimmed reason and clears the field",
            section: "Ban Management",
            typed: "  goat butchery  ",
            button: "Ban User",
            mutation: mutations.ban,
            sent: { id: "target-1", reason: "goat butchery" },
            shown: "User banned",
            settled: "",
        },
        {
            name: "refuses to lock without a reason, then locks with the trimmed reason",
            section: "Lock Management",
            typed: " noisy furniture ",
            button: "Lock User",
            mutation: mutations.lock,
            sent: { id: "target-1", reason: "noisy furniture" },
            shown: "User locked",
            settled: "",
        },
    ];

    it.each(saveCases)("$name", async ({ section, typed, button, mutation, sent, shown, settled }) => {
        // given the section to fill in and what it should send, from the table row
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");
        const field = within(fieldFor(section));
        const box = field.getByRole("textbox");
        const save = field.getByRole("button", { name: button });

        // then nothing can be sent before the field changes
        expect(save).toBeDisabled();

        // when
        await retype(user, box, typed);

        // then
        expect(save).toBeEnabled();

        // when
        await user.click(save);

        // then
        expect(mutation).toHaveBeenCalledWith(sent);
        expect(await screen.findByText(shown)).toBeInTheDocument();
        expect(box).toHaveValue(settled);
    });

    const refusalCases = [
        {
            name: "reports why an email change was refused",
            section: "Change Email",
            typed: "beato@example.com",
            button: "Save Email",
            mutation: mutations.setEmail,
            reason: "that address is already spoken for",
        },
        {
            name: "reports why a ban was refused",
            section: "Ban Management",
            typed: "goat butchery",
            button: "Ban User",
            mutation: mutations.ban,
            reason: "the witch forbids it",
        },
    ];

    it.each(refusalCases)("$name", async ({ section, typed, button, mutation, reason }) => {
        // given a refusal carrying its own reason, from the table row
        stubTarget(makeTarget());
        mutation.mockRejectedValue(new Error(reason));
        const user = userEvent.setup();
        renderDetail("admin");
        const field = within(fieldFor(section));
        await retype(user, field.getByRole("textbox"), typed);

        // when
        await user.click(field.getByRole("button", { name: button }));

        // then
        expect(await screen.findByText(reason)).toBeInTheDocument();
    });

    it("assigns the chosen role to an account with none", async () => {
        // given
        stubTarget(makeTarget({ role: undefined }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.selectOptions(screen.getByRole("combobox"), "moderator");
        await user.click(screen.getByRole("button", { name: "Assign Role" }));

        // then
        expect(mutations.setRole).toHaveBeenCalledWith({ id: "target-1", role: "moderator" });
        expect(await screen.findByText("Role assigned")).toBeInTheDocument();
    });
});

describe("AdminUserDetail confirmations", () => {
    it("warns before taking a verified address away", async () => {
        // given
        stubTarget(makeTarget({ email_verified: true }));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: "Mark Unverified" }));

        // then the warning spells out what the user loses
        expect(
            screen.getByText(
                "Mark this email unverified? The user will be blocked from posting, commenting and messaging until they verify it again.",
            ),
        ).toBeInTheDocument();

        // when the warning is declined
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(mutations.unverifyEmail).not.toHaveBeenCalled();
    });

    const cancelCases = [
        {
            name: "warns before revoking every session",
            opener: "Revoke All Sessions",
            confirmLabel: "Revoke Sessions",
            mutation: mutations.forceLogout,
        },
        {
            name: "leaves the password alone when the warning is dismissed",
            opener: "Reset Password",
            confirmLabel: "Reset",
            mutation: mutations.resetPassword,
        },
        {
            name: "keeps the account when the delete dialogue is cancelled",
            opener: "Delete User",
            confirmLabel: "Delete",
            mutation: mutations.deleteUser,
        },
    ];

    it.each(cancelCases)("$name", async ({ opener, confirmLabel, mutation }) => {
        // given the button that opens the warning, from the table row
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(screen.getByRole("button", { name: opener }));
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(mutation).not.toHaveBeenCalled();
        expect(screen.queryByRole("button", { name: confirmLabel })).not.toBeInTheDocument();
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
        expect(mutations.deleteUser).toHaveBeenCalledWith("target-1");
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/admin/users");
        });
    });
});

describe("AdminUserDetail mystery scores", () => {
    it("prefills both scores and ignores anything that is not a whole number", async () => {
        // given
        stubTarget(makeTarget({ detective_score: 12, gm_score: 4 }));
        const user = userEvent.setup();
        renderDetail("admin");
        const detective = within(fieldFor("Detective Score")).getByRole("textbox");

        // then
        expect(detective).toHaveValue("12");
        expect(within(fieldFor("Game Master Score")).getByRole("textbox")).toHaveValue("4");

        // when
        await retype(user, detective, "3a5");

        // then
        expect(detective).toHaveValue("35");
    });

    const scoreCases = [
        {
            name: "saves the detective score the moderator typed",
            section: "Detective Score",
            typed: "35",
            mutation: mutations.detectiveScore,
            desiredScore: 35,
        },
        {
            name: "saves the game master score the moderator typed",
            section: "Game Master Score",
            typed: "-6",
            mutation: mutations.gmScore,
            desiredScore: -6,
        },
    ];

    it.each(scoreCases)("$name", async ({ section, typed, mutation, desiredScore }) => {
        // given the score field and what the moderator types into it, from the table row
        stubTarget(makeTarget());
        const user = userEvent.setup();
        renderDetail("moderator");
        const field = within(fieldFor(section));
        await retype(user, field.getByRole("textbox"), typed);

        // when
        await user.click(field.getByRole("button", { name: "Save" }));

        // then
        expect(mutation).toHaveBeenCalledWith({ id: "target-1", desiredScore });
    });

    const scoreRefusalCases = [
        {
            name: "reports why a detective score save was refused instead of swallowing it",
            section: "Detective Score",
            mutation: mutations.detectiveScore,
            prefilled: 12,
            reason: "the score ledger is sealed",
            shown: "the score ledger is sealed",
        },
        {
            name: "reports why a game master score save was refused instead of swallowing it",
            section: "Game Master Score",
            mutation: mutations.gmScore,
            prefilled: 4,
            reason: "",
            shown: "Failed to update game master score",
        },
    ];

    it.each(scoreRefusalCases)("$name", async ({ section, mutation, prefilled, reason, shown }) => {
        // given a refusal, with or without a reason of its own, against the untouched prefilled score from the table row
        stubTarget(makeTarget({ detective_score: 12, gm_score: 4 }));
        mutation.mockRejectedValue(new Error(reason));
        const user = userEvent.setup();
        renderDetail("admin");

        // when
        await user.click(within(fieldFor(section)).getByRole("button", { name: "Save" }));

        // then
        expect(mutation).toHaveBeenCalledWith({ id: "target-1", desiredScore: prefilled });
        expect(await screen.findByText(shown)).toBeInTheDocument();
    });
});

describe("AdminUserDetail shared IP addresses", () => {
    const lookupCases = [
        {
            name: "says when nobody else shares the address",
            lookup: { users: [] },
            shown: "No other accounts share this IP address.",
        },
        {
            name: "reports when the shared address lookup failed",
            lookup: { failed: true },
            shown: "Could not load accounts for this IP address.",
        },
    ];

    it.each(lookupCases)("$name", ({ lookup, shown }) => {
        // given an account with a recorded address, and the lookup result from the table row
        stubTarget(makeTarget({ ip: "203.0.113.9" }));
        stubIPMatches(lookup);

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText(shown)).toBeInTheDocument();
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
    const stateCases = [
        { name: "waits while the history is loading", history: { loading: true }, shown: "Loading..." },
        {
            name: "says when nothing has ever been recorded",
            history: { entries: [] },
            shown: "Nothing has been recorded against this account.",
        },
        {
            name: "reports when the history could not be loaded",
            history: { failed: true },
            shown: "Could not load the account history.",
        },
    ];

    it.each(stateCases)("$name", ({ history, shown }) => {
        // given the history lookup result, from the table row
        stubTarget(makeTarget());
        stubHistory(history);

        // when
        renderDetail("admin");

        // then
        expect(screen.getByText(shown)).toBeInTheDocument();
    });

    const entryCases: { name: string; entry: Partial<AuditLogEntry>; shown: string[] }[] = [
        {
            name: "labels each recorded action and unpacks its details",
            entry: {},
            shown: ["Banned", "reason", "spam in the parlour", "Virgilia"],
        },
        {
            name: "credits an entry with no actor to the system",
            entry: { actor_name: "", action: "user_created", details: "" },
            shown: ["Account created", "system"],
        },
    ];

    it.each(entryCases)("$name", ({ entry, shown }) => {
        // given the recorded entry, from the table row
        stubTarget(makeTarget());
        stubHistory({ entries: [makeHistoryEntry(entry)] });

        // when
        renderDetail("admin");

        // then
        for (const text of shown) {
            expect(screen.getByText(text)).toBeInTheDocument();
        }
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
