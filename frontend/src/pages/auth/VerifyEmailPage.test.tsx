import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { VerifyEmailPage } from "./VerifyEmailPage";

const mocks = vi.hoisted(() => ({
    navigate: vi.fn(),
    verifyEmail: vi.fn(),
}));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

const verifyMutation = { mutateAsync: mocks.verifyEmail };

vi.mock("../../api/mutations/auth", () => ({
    useVerifyEmail: () => verifyMutation,
}));

function setup(route = "/verify-email?token=kakera-token") {
    const user = userEvent.setup();
    const result = renderWithProviders(<VerifyEmailPage />, { route });

    return { user, ...result };
}

beforeEach(() => {
    mocks.verifyEmail.mockResolvedValue(undefined);
});

describe("VerifyEmailPage", () => {
    it("rejects a link that carries no token without calling the server", () => {
        // given
        const route = "/verify-email";

        // when
        setup(route);

        // then
        expect(screen.getByText("This verification link is invalid or has expired.")).toBeInTheDocument();
        expect(mocks.verifyEmail).not.toHaveBeenCalled();
    });

    it("reports progress while the token is being checked", () => {
        // given
        mocks.verifyEmail.mockImplementation(() => new Promise(() => {}));

        // when
        setup();

        // then
        expect(screen.getByText("Verifying your email...")).toBeInTheDocument();
    });

    it("sends the token from the link to the verification endpoint", () => {
        // given
        const route = "/verify-email?token=kakera-token";

        // when
        setup(route);

        // then
        expect(mocks.verifyEmail).toHaveBeenCalledWith("kakera-token");
    });

    it("checks the token only once", () => {
        // given
        const { rerender } = setup();

        // when
        rerender(<VerifyEmailPage />);

        // then
        expect(mocks.verifyEmail).toHaveBeenCalledOnce();
    });

    it("thanks the member when the token is accepted", async () => {
        // given
        const route = "/verify-email?token=kakera-token";

        // when
        setup(route);

        // then
        expect(await screen.findByText("Your email is verified. Thank you!")).toBeInTheDocument();
    });

    it("leads the member onwards once the email is verified", async () => {
        // given
        const { user } = setup();
        await screen.findByRole("button", { name: "Continue" });

        // when
        await user.click(screen.getByRole("button", { name: "Continue" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/");
    });

    it("reports an expired or already used token", async () => {
        // given
        mocks.verifyEmail.mockRejectedValue(new Error("token expired"));

        // when
        setup();

        // then
        expect(await screen.findByText("This verification link is invalid or has expired.")).toBeInTheDocument();
    });

    it("offers a way home when the token could not be used", async () => {
        // given
        mocks.verifyEmail.mockRejectedValue(new Error("token expired"));
        const { user } = setup();
        await screen.findByRole("button", { name: "Go home" });

        // when
        await user.click(screen.getByRole("button", { name: "Go home" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/");
    });
});
