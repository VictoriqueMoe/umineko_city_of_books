import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserProfile } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { VerifyEmailBanner } from "./VerifyEmailBanner";

const { useResendVerification, mutateAsync } = vi.hoisted(() => ({
    useResendVerification: vi.fn(),
    mutateAsync: vi.fn(),
}));

vi.mock("../../api/mutations/auth", () => ({ useResendVerification }));

function unverified(overrides: Partial<UserProfile> = {}): UserProfile {
    return makeUser({
        email: "beatrice@example.com",
        private: { email_verified: false },
        ...overrides,
    });
}

function renderBanner(user: UserProfile | null) {
    return renderWithProviders(<VerifyEmailBanner />, { user });
}

beforeEach(() => {
    mutateAsync.mockResolvedValue(undefined);
    useResendVerification.mockReturnValue({ mutateAsync, isPending: false });
});

describe("VerifyEmailBanner", () => {
    it("says nothing to a signed out visitor", () => {
        // given
        const user = null;

        // when
        const { container } = renderBanner(user);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("says nothing once the email has been verified", () => {
        // given
        const user = makeUser({ email: "beatrice@example.com", private: { email_verified: true } });

        // when
        const { container } = renderBanner(user);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("treats an account with no verification flag as settled", () => {
        // given
        const user = makeUser({ email: "beatrice@example.com" });

        // when
        const { container } = renderBanner(user);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("asks a user with no email address to add one", () => {
        // given
        const user = unverified({ email: "" });

        // when
        renderBanner(user);

        // then
        expect(screen.getByText(/Add an email address to your account/)).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Add email" })).toHaveAttribute("href", "/set-email");
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("counts the days left in the grace period", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
        const user = unverified({ private: { email_verified: false, verify_grace_until: "2026-01-04T00:00:00Z" } });

        // when
        renderBanner(user);

        // then
        expect(
            screen.getByText("Verify your email (beatrice@example.com) within 3 days to keep posting."),
        ).toBeInTheDocument();
    });

    it("uses the singular when only one day is left", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
        const user = unverified({ private: { email_verified: false, verify_grace_until: "2026-01-02T00:00:00Z" } });

        // when
        renderBanner(user);

        // then
        expect(
            screen.getByText("Verify your email (beatrice@example.com) within 1 day to keep posting."),
        ).toBeInTheDocument();
    });

    it("warns that the account is read only once the grace period has run out", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-01-10T00:00:00Z"));
        const user = unverified({ private: { email_verified: false, verify_grace_until: "2026-01-01T00:00:00Z" } });

        // when
        renderBanner(user);

        // then
        expect(
            screen.getByText(
                "Verify your email (beatrice@example.com) to post again. Your account is read-only until you do.",
            ),
        ).toBeInTheDocument();
    });

    it("treats a missing deadline as an expired grace period", () => {
        // given
        const user = unverified();

        // when
        renderBanner(user);

        // then
        expect(screen.getByText(/Your account is read-only until you do\./)).toBeInTheDocument();
    });

    it("treats an unparseable deadline as an expired grace period", () => {
        // given
        const user = unverified({ private: { email_verified: false, verify_grace_until: "not a date" } });

        // when
        renderBanner(user);

        // then
        expect(screen.getByText(/Your account is read-only until you do\./)).toBeInTheDocument();
    });

    it("resends the verification email and confirms it went out", async () => {
        // given
        const user = userEvent.setup();
        renderBanner(unverified());

        // when
        await user.click(screen.getByRole("button", { name: "Resend email" }));

        // then
        expect(mutateAsync).toHaveBeenCalledOnce();
        expect(await screen.findByText("Sent. Check your inbox.")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Resend email" })).not.toBeInTheDocument();
    });

    it("blocks a second send while the first is still in flight", () => {
        // given
        useResendVerification.mockReturnValue({ mutateAsync, isPending: true });

        // when
        renderBanner(unverified());

        // then
        expect(screen.getByRole("button", { name: "Sending..." })).toBeDisabled();
    });

    it("surfaces the reason a resend failed", async () => {
        // given
        mutateAsync.mockRejectedValue(new Error("too many requests, try later"));
        const user = userEvent.setup();
        renderBanner(unverified());

        // when
        await user.click(screen.getByRole("button", { name: "Resend email" }));

        // then
        expect(await screen.findByText("too many requests, try later")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Resend email" })).toBeInTheDocument();
    });

    it("falls back to a generic message when the failure is not an error", async () => {
        // given
        mutateAsync.mockRejectedValue("something odd");
        const user = userEvent.setup();
        renderBanner(unverified());

        // when
        await user.click(screen.getByRole("button", { name: "Resend email" }));

        // then
        expect(await screen.findByText("Failed to resend")).toBeInTheDocument();
    });
});
