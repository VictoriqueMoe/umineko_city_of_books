import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReportItem } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { AdminReports } from "./AdminReports";

const mocks = vi.hoisted(() => ({
    useReports: vi.fn(),
    resolve: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({ useReports: mocks.useReports }));

vi.mock("../../api/mutations/admin", () => ({
    useResolveReport: () => ({ mutateAsync: mocks.resolve, isPending: false }),
}));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => mocks.navigate };
});

function makeReport(overrides: Partial<ReportItem> = {}): ReportItem {
    return {
        id: 11,
        reporter_name: "beatrice",
        reporter_avatar: "",
        target_type: "theory",
        target_id: "theory-1",
        reason: "endless repetition of the same red truth",
        status: "open",
        created_at: "2026-02-03T10:00:00Z",
        ...overrides,
    };
}

function stubReports(reports: ReportItem[], loading = false) {
    mocks.useReports.mockReturnValue({ reports, loading, refresh: vi.fn() });
}

beforeEach(() => {
    mocks.resolve.mockResolvedValue(undefined);
});

describe("AdminReports", () => {
    it("waits while the reports are being fetched", () => {
        // given
        stubReports([], true);

        // when
        renderWithProviders(<AdminReports />);

        // then
        expect(screen.getByText("Loading reports...")).toBeInTheDocument();
    });

    it("says so when the queue is clear", () => {
        // given
        stubReports([]);

        // when
        renderWithProviders(<AdminReports />);

        // then
        expect(screen.getByText("No reports found")).toBeInTheDocument();
    });

    it("shows who reported what and why", () => {
        // given
        stubReports([makeReport()]);

        // when
        renderWithProviders(<AdminReports />);

        // then
        expect(screen.getByText("beatrice")).toBeInTheDocument();
        expect(screen.getByText("theory")).toBeInTheDocument();
        expect(screen.getByText("endless repetition of the same red truth")).toBeInTheDocument();
    });

    it("falls back to an upper case initial when the reporter has no avatar", () => {
        // given
        stubReports([makeReport({ reporter_avatar: "" })]);

        // when
        const { container } = renderWithProviders(<AdminReports />);

        // then
        expect(screen.getByText("B")).toBeInTheDocument();
        expect(container.querySelector("img")).toBeNull();
    });

    it("starts on the open reports and can switch to the resolved ones", async () => {
        // given
        stubReports([]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);
        expect(mocks.useReports).toHaveBeenLastCalledWith("open");

        // when
        await user.selectOptions(screen.getByRole("combobox"), "resolved");

        // then
        expect(mocks.useReports).toHaveBeenLastCalledWith("resolved");
    });

    it("only offers to resolve a report that is still open", () => {
        // given
        stubReports([makeReport({ status: "resolved" })]);

        // when
        renderWithProviders(<AdminReports />);

        // then
        expect(screen.queryByRole("button", { name: "Resolve" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "View" })).toBeInTheDocument();
    });

    it("resolves a report with the message typed for the reporter", async () => {
        // given
        stubReports([makeReport({ id: 42 })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);
        await user.click(screen.getByRole("button", { name: "Resolve" }));

        // when
        const textarea = screen.getByPlaceholderText("Let them know what action was taken...");
        await user.type(textarea, "the theory has been removed");
        await user.click(within(textarea.closest("div") as HTMLElement).getByRole("button", { name: "Resolve" }));

        // then
        expect(mocks.resolve).toHaveBeenCalledWith({ id: 42, comment: "the theory has been removed" });
    });

    it("closes the resolve dialogue without acting when it is cancelled", async () => {
        // given
        stubReports([makeReport()]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);
        await user.click(screen.getByRole("button", { name: "Resolve" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(mocks.resolve).not.toHaveBeenCalled();
        expect(screen.queryByPlaceholderText("Let them know what action was taken...")).not.toBeInTheDocument();
    });

    it("opens a reported theory on its own page", async () => {
        // given
        stubReports([makeReport({ target_type: "theory", target_id: "theory-9" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);

        // when
        await user.click(screen.getByRole("button", { name: "View" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/theory/theory-9");
    });

    it("opens a reported response anchored inside its parent theory", async () => {
        // given
        stubReports([makeReport({ target_type: "response", target_id: "response-3", context_id: "theory-9" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);

        // when
        await user.click(screen.getByRole("button", { name: "View" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/theory/theory-9#response-response-3");
    });

    it("opens a reported gallery comment anchored inside its artwork", async () => {
        // given
        stubReports([makeReport({ target_type: "art_comment", target_id: "c-1", context_id: "art-4" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);

        // when
        await user.click(screen.getByRole("button", { name: "View" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/gallery/art/art-4#comment-c-1");
    });

    it("opens a reported journal on its own page", async () => {
        // given
        stubReports([makeReport({ target_type: "journal", target_id: "journal-2" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);

        // when
        await user.click(screen.getByRole("button", { name: "View" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/journals/journal-2");
    });

    it("opens a reported secret comment anchored inside its hunt", async () => {
        // given
        stubReports([makeReport({ target_type: "secret_comment", target_id: "c-1", context_id: "secret-4" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);

        // when
        await user.click(screen.getByRole("button", { name: "View" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/secrets/secret-4#comment-c-1");
    });

    it("opens a reported character comment anchored inside its profile", async () => {
        // given
        stubReports([makeReport({ target_type: "oc_comment", target_id: "c-1", context_id: "oc-4" })]);
        const user = userEvent.setup();
        renderWithProviders(<AdminReports />);

        // when
        await user.click(screen.getByRole("button", { name: "View" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/oc/oc-4#comment-c-1");
    });

    it("offers no View button when a reported comment has lost the thread it belonged to", () => {
        // given
        stubReports([makeReport({ target_type: "comment", target_id: "c-1", context_id: undefined })]);

        // when
        renderWithProviders(<AdminReports />);

        // then
        expect(screen.queryByRole("button", { name: "View" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Resolve" })).toBeInTheDocument();
    });

    it("offers no View button for a target type the admin pages cannot open", () => {
        // given
        stubReports([makeReport({ target_type: "user", target_id: "user-1", context_id: undefined })]);

        // when
        renderWithProviders(<AdminReports />);

        // then
        expect(screen.queryByRole("button", { name: "View" })).not.toBeInTheDocument();
    });
});
