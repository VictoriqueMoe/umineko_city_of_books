import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserProfile } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { ReportButton } from "./ReportButton";

const mocks = vi.hoisted(() => ({
    useCreateReport: vi.fn(),
    mutateAsync: vi.fn(),
}));

vi.mock("../../api/mutations/misc", () => ({ useCreateReport: mocks.useCreateReport }));

const reporter = makeUser({ id: "user-1", username: "battler", display_name: "Battler" });

const REASON_PLACEHOLDER = "Why are you reporting this?";

function renderReportButton(contextId?: string, user: UserProfile | null = reporter) {
    return renderWithProviders(<ReportButton targetType="post" targetId="post-1" contextId={contextId} />, { user });
}

describe("ReportButton", () => {
    beforeEach(() => {
        mocks.mutateAsync.mockResolvedValue(undefined);
        mocks.useCreateReport.mockReturnValue({ mutateAsync: mocks.mutateAsync, isPending: false });
    });

    it("stays hidden from signed out visitors", () => {
        // given
        const user = null;

        // when
        const { container } = renderReportButton(undefined, user);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("keeps the dialog shut until the report button is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderReportButton();
        expect(screen.queryByRole("heading", { name: "Report Content" })).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Report" }));

        // then
        expect(screen.getByRole("heading", { name: "Report Content" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText(REASON_PLACEHOLDER)).toHaveValue("");
    });

    it("refuses to send a report with no reason behind it", async () => {
        // given
        const user = userEvent.setup();
        renderReportButton();
        await user.click(screen.getByRole("button", { name: "Report" }));
        expect(screen.getByRole("button", { name: "Submit Report" })).toBeDisabled();

        // when
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "   ");

        // then
        expect(screen.getByRole("button", { name: "Submit Report" })).toBeDisabled();
        expect(mocks.mutateAsync).not.toHaveBeenCalled();
    });

    it("sends the trimmed reason together with the target and its context", async () => {
        // given
        const user = userEvent.setup();
        renderReportButton("thread-9");
        await user.click(screen.getByRole("button", { name: "Report" }));

        // when
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "  harassing another player  ");
        await user.click(screen.getByRole("button", { name: "Submit Report" }));

        // then
        expect(mocks.mutateAsync).toHaveBeenCalledWith({
            targetType: "post",
            targetId: "post-1",
            reason: "harassing another player",
            contextId: "thread-9",
        });
    });

    it("sends no context when the report has none", async () => {
        // given
        const user = userEvent.setup();
        renderReportButton();
        await user.click(screen.getByRole("button", { name: "Report" }));

        // when
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "spoilers");
        await user.click(screen.getByRole("button", { name: "Submit Report" }));

        // then
        expect(mocks.mutateAsync).toHaveBeenCalledWith({
            targetType: "post",
            targetId: "post-1",
            reason: "spoilers",
            contextId: undefined,
        });
    });

    it("confirms the report once a moderator has been told", async () => {
        // given
        const user = userEvent.setup();
        renderReportButton();
        await user.click(screen.getByRole("button", { name: "Report" }));
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "spoilers");

        // when
        await user.click(screen.getByRole("button", { name: "Submit Report" }));

        // then
        expect(screen.getByText("Report submitted. A moderator will review it.")).toBeInTheDocument();
        expect(screen.queryByPlaceholderText(REASON_PLACEHOLDER)).not.toBeInTheDocument();
    });

    it("closes the whole dialog from the confirmation", async () => {
        // given
        const user = userEvent.setup();
        renderReportButton();
        await user.click(screen.getByRole("button", { name: "Report" }));
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "spoilers");
        await user.click(screen.getByRole("button", { name: "Submit Report" }));

        // when
        await user.click(screen.getByRole("button", { name: "Close" }));

        // then
        expect(screen.queryByRole("heading", { name: "Report Content" })).not.toBeInTheDocument();
    });

    it("explains why a report was rejected", async () => {
        // given
        mocks.mutateAsync.mockRejectedValue(new Error("You have already reported this"));
        const user = userEvent.setup();
        renderReportButton();
        await user.click(screen.getByRole("button", { name: "Report" }));
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "spoilers");

        // when
        await user.click(screen.getByRole("button", { name: "Submit Report" }));

        // then
        expect(screen.getByText("You have already reported this")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Submit Report" })).toBeEnabled();
    });

    it("falls back to a generic message when the failure carries no message", async () => {
        // given
        mocks.mutateAsync.mockRejectedValue("the golden truth denies it");
        const user = userEvent.setup();
        renderReportButton();
        await user.click(screen.getByRole("button", { name: "Report" }));
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "spoilers");

        // when
        await user.click(screen.getByRole("button", { name: "Submit Report" }));

        // then
        expect(screen.getByText("Failed to submit report")).toBeInTheDocument();
    });

    it("shows the report is on its way while it is still in flight", async () => {
        // given
        mocks.useCreateReport.mockReturnValue({ mutateAsync: mocks.mutateAsync, isPending: true });
        const user = userEvent.setup();
        renderReportButton();

        // when
        await user.click(screen.getByRole("button", { name: "Report" }));

        // then
        expect(screen.getByRole("button", { name: "Submitting..." })).toBeDisabled();
        expect(screen.queryByRole("button", { name: "Submit Report" })).not.toBeInTheDocument();
    });

    it("throws away a half written reason when the dialog is cancelled", async () => {
        // given
        const user = userEvent.setup();
        renderReportButton();
        await user.click(screen.getByRole("button", { name: "Report" }));
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "spoilers");

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));
        await user.click(screen.getByRole("button", { name: "Report" }));

        // then
        expect(screen.getByPlaceholderText(REASON_PLACEHOLDER)).toHaveValue("");
    });

    it("throws away the previous error when the dialog is reopened", async () => {
        // given
        mocks.mutateAsync.mockRejectedValue(new Error("You have already reported this"));
        const user = userEvent.setup();
        renderReportButton();
        await user.click(screen.getByRole("button", { name: "Report" }));
        await user.type(screen.getByPlaceholderText(REASON_PLACEHOLDER), "spoilers");
        await user.click(screen.getByRole("button", { name: "Submit Report" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));
        await user.click(screen.getByRole("button", { name: "Report" }));

        // then
        expect(screen.queryByText("You have already reported this")).not.toBeInTheDocument();
    });
});
