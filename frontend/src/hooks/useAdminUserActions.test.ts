import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { AdminUserDetail, SiteRole } from "../types/api";
import { useAdminUserActions } from "./useAdminUserActions";

const mocks = vi.hoisted(() => ({
    useAdminUser: vi.fn(),
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

vi.mock("./queries/admin", () => ({
    useAdminUser: mocks.useAdminUser,
}));

vi.mock("./mutations/admin", () => ({
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

function makeTarget(overrides: Partial<AdminUserDetail> = {}): AdminUserDetail {
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

function renderActions(
    options: { target?: AdminUserDetail | null; loading?: boolean; actor?: SiteRole; id?: string } = {},
) {
    mocks.useAdminUser.mockReturnValue({
        user: options.target === undefined ? makeTarget() : options.target,
        loading: options.loading ?? false,
    });

    return renderHook(() => useAdminUserActions(options.id === undefined ? "target-1" : options.id), {
        wrapper: providerWrapper({
            user: makeUser({ id: "staff-1", username: "virgilia", role: options.actor ?? "admin" }),
        }),
    });
}

beforeEach(() => {
    for (const mutation of MUTATIONS) {
        mutation.mockReset();
        mutation.mockResolvedValue(undefined);
    }
    mocks.navigate.mockReset();
});

describe("useAdminUserActions capabilities", () => {
    it("closes locking alone against an admin target while banning stays open", () => {
        // given a target who is themselves an administrator
        const { result } = renderActions({ target: makeTarget({ role: "admin" }), actor: "super_admin" });

        // then
        expect(result.current.capabilities.canBan).toBe(true);
        expect(result.current.capabilities.canRevokeSessions).toBe(true);
        expect(result.current.capabilities.canLock).toBe(false);
    });

    it("closes every gate against a super admin target", () => {
        // given
        const { result } = renderActions({ target: makeTarget({ role: "super_admin" }), actor: "super_admin" });

        // then
        expect(result.current.capabilities.isProtected).toBe(true);
        expect(result.current.capabilities.canBan).toBe(false);
        expect(result.current.capabilities.canLock).toBe(false);
        expect(result.current.capabilities.canManageEmail).toBe(false);
        expect(result.current.capabilities.canDeleteUser).toBe(false);
    });

    it("leaves an ordinary target open to both banning and locking", () => {
        // given
        const { result } = renderActions({ target: makeTarget(), actor: "moderator" });

        // then
        expect(result.current.capabilities.canBan).toBe(true);
        expect(result.current.capabilities.canLock).toBe(true);
    });
});

describe("useAdminUserActions guards", () => {
    it("does nothing at all without an account id", async () => {
        // given
        const { result } = renderActions({ id: "" });
        act(() => result.current.setBanReason("goat butchery"));

        // when
        await act(async () => {
            await result.current.banUser();
        });

        // then
        expect(mocks.ban).not.toHaveBeenCalled();
        expect(result.current.feedback).toBe("");
    });

    it("holds an email save back until an address is typed", async () => {
        // given
        const { result } = renderActions();
        act(() => result.current.setEmailInput("   "));

        // when
        await act(async () => {
            await result.current.saveEmail();
        });

        // then
        expect(mocks.setEmail).not.toHaveBeenCalled();
    });

    it("holds a role removal back when the account holds no role", async () => {
        // given
        const { result } = renderActions({ target: makeTarget({ role: undefined }) });

        // when
        await act(async () => {
            await result.current.removeRole();
        });

        // then
        expect(mocks.removeRole).not.toHaveBeenCalled();
    });
});

describe("useAdminUserActions reporting", () => {
    it("confirms a successful action in its own words", async () => {
        // given
        const { result } = renderActions();

        // when
        await act(async () => {
            await result.current.unbanUser();
        });

        // then
        expect(mocks.unban).toHaveBeenCalledWith("target-1");
        expect(result.current.feedback).toBe("User unbanned");
    });

    it("repeats the server's reason when an action is refused", async () => {
        // given
        mocks.lock.mockRejectedValue(new Error("the witch forbids it"));
        const { result } = renderActions();
        act(() => result.current.setLockReason("noisy furniture"));

        // when
        await act(async () => {
            await result.current.lockUser();
        });

        // then
        expect(result.current.error).toBe("the witch forbids it");
        expect(result.current.feedback).toBe("");
    });

    it("falls back to the action's own message when the failure carries no reason", async () => {
        // given
        mocks.unlock.mockRejectedValue(new Error(""));
        const { result } = renderActions();

        // when
        await act(async () => {
            await result.current.unlockUser();
        });

        // then
        expect(result.current.error).toBe("Failed to unlock user");
    });

    it("clears the ban reason once the ban lands", async () => {
        // given
        const { result } = renderActions();
        act(() => result.current.setBanReason("  goat butchery  "));

        // when
        await act(async () => {
            await result.current.banUser();
        });

        // then
        expect(mocks.ban).toHaveBeenCalledWith({ id: "target-1", reason: "goat butchery" });
        expect(result.current.banReason).toBe("");
    });

    it("keeps the ban reason when the ban is refused", async () => {
        // given
        mocks.ban.mockRejectedValue(new Error("nope"));
        const { result } = renderActions();
        act(() => result.current.setBanReason("goat butchery"));

        // when
        await act(async () => {
            await result.current.banUser();
        });

        // then
        expect(result.current.banReason).toBe("goat butchery");
    });

    it("names the lock state the toggle is moving away from", async () => {
        // given
        const { result } = renderActions({ target: makeTarget({ display_name_locked: true }) });

        // when
        await act(async () => {
            await result.current.toggleDisplayNameLock();
        });

        // then
        expect(mocks.setDisplayNameLock).toHaveBeenCalledWith({ id: "target-1", locked: false });
        expect(result.current.feedback).toBe("Display name unlocked");
    });
});

describe("useAdminUserActions mystery scores", () => {
    it("sends the typed detective score as a whole number", async () => {
        // given
        const { result } = renderActions();
        act(() => result.current.setDetectiveScoreInput("35"));

        // when
        await act(async () => {
            await result.current.saveDetectiveScore();
        });

        // then
        expect(mocks.detectiveScore).toHaveBeenCalledWith({ id: "target-1", desiredScore: 35 });
    });

    it("refuses anything that is not a whole number", () => {
        // given
        const { result } = renderActions();

        // when
        act(() => result.current.setGMScoreInput("1a2"));

        // then
        expect(result.current.gmScoreInput).toBe("4");
    });

    it("treats a lone minus sign as zero", async () => {
        // given
        const { result } = renderActions();
        act(() => result.current.setGMScoreInput("-"));

        // when
        await act(async () => {
            await result.current.saveGMScore();
        });

        // then
        expect(mocks.gmScore).toHaveBeenCalledWith({ id: "target-1", desiredScore: 0 });
    });

    it("reports a refused detective score instead of swallowing it", async () => {
        // given
        mocks.detectiveScore.mockRejectedValue(new Error("the score ledger is sealed"));
        const { result } = renderActions();

        // when
        await act(async () => {
            await result.current.saveDetectiveScore();
        });

        // then
        expect(result.current.error).toBe("the score ledger is sealed");
    });

    it("reports a refused game master score instead of swallowing it", async () => {
        // given
        mocks.gmScore.mockRejectedValue(new Error(""));
        const { result } = renderActions();

        // when
        await act(async () => {
            await result.current.saveGMScore();
        });

        // then
        expect(result.current.error).toBe("Failed to update game master score");
    });

    it("says nothing at all when a score save succeeds", async () => {
        // given
        const { result } = renderActions();

        // when
        await act(async () => {
            await result.current.saveDetectiveScore();
        });

        // then
        expect(result.current.feedback).toBe("");
    });
});

describe("useAdminUserActions confirmations", () => {
    it("asks before unverifying an email rather than acting on the click", () => {
        // given
        const { result } = renderActions();

        // when
        act(() => result.current.requestUnverifyEmail());

        // then
        expect(result.current.confirmation).toBe("unverifyEmail");
        expect(mocks.unverifyEmail).not.toHaveBeenCalled();
    });

    it("drops a cancelled confirmation without acting", () => {
        // given
        const { result } = renderActions();
        act(() => result.current.requestForceLogout());

        // when
        act(() => result.current.cancelPending());

        // then
        expect(result.current.confirmation).toBeNull();
        expect(mocks.forceLogout).not.toHaveBeenCalled();
    });

    it("dismisses the dialogue and then acts once the confirmation is accepted", async () => {
        // given
        const { result } = renderActions();
        act(() => result.current.requestUnverifyEmail());

        // when
        await act(async () => {
            await result.current.confirmPending();
        });

        // then
        expect(mocks.unverifyEmail).toHaveBeenCalledWith("target-1");
        expect(result.current.confirmation).toBeNull();
        expect(result.current.feedback).toBe("Email marked as unverified");
    });

    it("hands back the new password a confirmed reset generated", async () => {
        // given
        mocks.resetPassword.mockResolvedValue({ password: "kakera-golden-77" });
        const { result } = renderActions();
        act(() => result.current.requestResetPassword());

        // when
        await act(async () => {
            await result.current.confirmPending();
        });

        // then
        expect(result.current.resetPasswordResult).toBe("kakera-golden-77");
    });

    it("returns to the roster after a confirmed delete", async () => {
        // given
        const { result } = renderActions();
        act(() => result.current.requestDeleteUser());

        // when
        await act(async () => {
            await result.current.confirmPending();
        });

        // then
        expect(mocks.deleteUser).toHaveBeenCalledWith("target-1");
        expect(mocks.navigate).toHaveBeenCalledWith("/admin/users");
    });

    it("closes the delete dialogue and reports the reason when the delete is refused", async () => {
        // given
        mocks.deleteUser.mockRejectedValue(new Error("the head of the family says no"));
        const { result } = renderActions();
        act(() => result.current.requestDeleteUser());

        // when
        await act(async () => {
            await result.current.confirmPending();
        });

        // then
        expect(result.current.confirmation).toBeNull();
        expect(result.current.error).toBe("the head of the family says no");
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("does nothing when nothing is waiting to be confirmed", async () => {
        // given
        const { result } = renderActions();

        // when
        await act(async () => {
            await result.current.confirmPending();
        });

        // then
        expect(mocks.deleteUser).not.toHaveBeenCalled();
        expect(mocks.unverifyEmail).not.toHaveBeenCalled();
        expect(mocks.forceLogout).not.toHaveBeenCalled();
        expect(mocks.resetPassword).not.toHaveBeenCalled();
    });
});
