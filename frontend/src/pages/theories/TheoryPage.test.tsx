import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { Response as TheoryResponse, TheoryDetail } from "../../types/api";
import { TheoryPage } from "./TheoryPage";

const { useTheory, useVoteTheory, useDeleteTheory, useRefuteTheory, navigate } = vi.hoisted(() => ({
    useTheory: vi.fn(),
    useVoteTheory: vi.fn(),
    useDeleteTheory: vi.fn(),
    useRefuteTheory: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/theory", () => ({ useTheory }));
vi.mock("../../api/mutations/theory", () => ({ useVoteTheory, useDeleteTheory, useRefuteTheory }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});
vi.mock("../../components/theory/EvidenceList/EvidenceList", () => ({
    EvidenceList: ({ evidence }: { evidence: unknown[] }) => (
        <div data-testid="evidence-list">{evidence.length} evidence</div>
    ),
}));
vi.mock("../../components/theory/ResponseCard/ResponseCard", () => ({
    ResponseList: ({ responses }: { responses: TheoryResponse[] }) => (
        <div data-testid="response-list">{responses.map(r => r.body).join(", ")}</div>
    ),
}));
vi.mock("../../components/theory/ResponseEditor/ResponseEditor", () => ({
    ResponseEditor: ({ theoryId }: { theoryId: string }) => <div data-testid="response-editor">{theoryId}</div>,
}));

const author = { id: "author-1", username: "beatrice", display_name: "Beatrice" };
const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });
const authorUser = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });

function makeResponse(overrides: Partial<TheoryResponse> = {}): TheoryResponse {
    return {
        id: "response-1",
        author,
        side: "with_love",
        body: "I agree entirely.",
        evidence: [],
        vote_score: 0,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

function makeTheoryDetail(overrides: Partial<TheoryDetail> = {}): TheoryDetail {
    return {
        id: "theory-1",
        title: "The culprit is Kanon",
        body: "Without love it cannot be seen.",
        episode: 0,
        series: "umineko",
        author,
        vote_score: 5,
        with_love_count: 1,
        without_love_count: 0,
        user_vote: 0,
        credibility_score: 55,
        status: "open" as const,
        created_at: "2026-07-01T10:00:00Z",
        evidence: [],
        responses: [],
        ...overrides,
    };
}

interface StubOptions {
    theory?: TheoryDetail | null;
    loading?: boolean;
    vote?: () => Promise<unknown>;
}

function stubTheory(options: StubOptions = {}) {
    const refresh = vi.fn();
    useTheory.mockReturnValue({
        theory: options.theory === undefined ? makeTheoryDetail() : options.theory,
        loading: options.loading ?? false,
        refresh,
    });
    const voteAsync = vi.fn(options.vote ?? (() => Promise.resolve({})));
    const deleteAsync = vi.fn(() => Promise.resolve({}));
    const refuteAsync = vi.fn();
    useVoteTheory.mockReturnValue({ mutateAsync: voteAsync });
    useDeleteTheory.mockReturnValue({ mutateAsync: deleteAsync });
    useRefuteTheory.mockReturnValue({ mutateAsync: refuteAsync });

    return { refresh, voteAsync, deleteAsync, refuteAsync };
}

function renderPage(user: ReturnType<typeof makeUser> | null = null, route = "/theory/theory-1") {
    return renderWithProviders(<TheoryPage />, { user, route, path: "/theory/:id" });
}

describe("TheoryPage", () => {
    it("consults the game board while the theory is loading", () => {
        // given
        stubTheory({ loading: true, theory: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Consulting the game board...")).toBeInTheDocument();
    });

    it("says the theory is missing when the server has none", () => {
        // given
        stubTheory({ theory: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Theory not found.")).toBeInTheDocument();
    });

    it("shows the declaration, the body and the evidence", () => {
        // given
        stubTheory({ theory: makeTheoryDetail({ evidence: [{ id: 1, note: "n", lang: "en", sort_order: 0 }] }) });

        // when
        renderPage(reader);

        // then
        expect(screen.getByRole("heading", { name: "The culprit is Kanon" })).toBeInTheDocument();
        expect(screen.getByText("Without love it cannot be seen.")).toBeInTheDocument();
        expect(screen.getByText("declares in blue:", { exact: false })).toBeInTheDocument();
        expect(screen.getByTestId("evidence-list")).toHaveTextContent("1 evidence");
    });

    it("warns a reader whose progress has not reached the theory's episode", () => {
        // given
        stubTheory({ theory: makeTheoryDetail({ episode: 5 }) });

        // when
        renderPage(makeUser({ id: "reader-1", episode_progress: 3 }));

        // then
        expect(screen.getByRole("heading", { name: "Spoiler Warning" })).toBeInTheDocument();
        expect(screen.getByText(/Episode 5/)).toBeInTheDocument();
        expect(screen.queryByText("Without love it cannot be seen.")).not.toBeInTheDocument();
    });

    it("reveals the theory once the reader chooses to continue anyway", async () => {
        // given
        stubTheory({ theory: makeTheoryDetail({ episode: 5 }) });
        const user = userEvent.setup();
        renderPage(makeUser({ id: "reader-1", episode_progress: 3 }));

        // when
        await user.click(screen.getByRole("button", { name: "Continue anyway" }));

        // then
        expect(screen.getByText("Without love it cannot be seen.")).toBeInTheDocument();
    });

    it("does not warn a reader who is already past the theory's episode", () => {
        // given
        stubTheory({ theory: makeTheoryDetail({ episode: 5 }) });

        // when
        renderPage(makeUser({ id: "reader-1", episode_progress: 8 }));

        // then
        expect(screen.queryByRole("heading", { name: "Spoiler Warning" })).not.toBeInTheDocument();
        expect(screen.getByText("Without love it cannot be seen.")).toBeInTheDocument();
    });

    it("does not warn a visitor who has recorded no progress at all", () => {
        // given
        stubTheory({ theory: makeTheoryDetail({ episode: 5 }) });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("heading", { name: "Spoiler Warning" })).not.toBeInTheDocument();
    });

    it("splits the debate into supporters and deniers with their counts", () => {
        // given
        stubTheory({
            theory: makeTheoryDetail({
                responses: [
                    makeResponse({ id: "r1", side: "with_love", body: "Supporting" }),
                    makeResponse({ id: "r2", side: "with_love", body: "Also supporting" }),
                    makeResponse({ id: "r3", side: "without_love", body: "Denying" }),
                ],
            }),
        });

        // when
        renderPage(reader);

        // then
        expect(screen.getByRole("heading", { name: "With love, it can be seen (2)" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Without love, it cannot be seen (1)" })).toBeInTheDocument();
        expect(screen.getAllByTestId("response-list")[0]).toHaveTextContent("Supporting, Also supporting");
    });

    it("shows both debate empty states when nobody has responded", () => {
        // given
        stubTheory({ theory: makeTheoryDetail({ responses: [] }) });

        // when
        renderPage(reader);

        // then
        expect(screen.getByText("No supporters yet.")).toBeInTheDocument();
        expect(screen.getByText("No deniers yet.")).toBeInTheDocument();
    });

    it("offers the response editor to a signed in reader who is not the author", () => {
        // given
        stubTheory();

        // when
        renderPage(reader);

        // then
        expect(screen.getByTestId("response-editor")).toHaveTextContent("theory-1");
        expect(screen.getByRole("button", { name: "Report" })).toBeInTheDocument();
    });

    it("does not let the author respond to or report their own theory", () => {
        // given
        stubTheory();

        // when
        renderPage(authorUser);

        // then
        expect(screen.queryByTestId("response-editor")).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
    });

    it("asks a signed out visitor to sign in before responding", async () => {
        // given
        stubTheory();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Sign in to respond" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/login");
        expect(screen.queryByTestId("response-editor")).not.toBeInTheDocument();
    });

    it("hides the author actions from a reader who neither wrote nor moderates", () => {
        // given
        stubTheory();

        // when
        renderPage(reader);

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("lets the author edit their own theory", async () => {
        // given
        stubTheory();
        const user = userEvent.setup();
        renderPage(authorUser);

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/theory/theory-1/edit");
    });

    it("lets a moderator edit and delete a theory they did not write", () => {
        // given
        stubTheory();

        // when
        renderPage(makeUser({ id: "mod-1", role: "moderator" }));

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("keeps the theory when the author backs out of the final confirmation", async () => {
        // given
        const { deleteAsync } = stubTheory();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(authorUser);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));
        await user.click(screen.getByRole("button", { name: "Delete Theory" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Are you sure you want to delete this theory?");
        expect(deleteAsync).not.toHaveBeenCalled();
    });

    it("deletes the theory and returns to the series feed once confirmed", async () => {
        // given
        const { deleteAsync } = stubTheory({ theory: makeTheoryDetail({ series: "higurashi" }) });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(authorUser);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));
        await user.click(screen.getByRole("button", { name: "Delete Theory" }));

        // then
        expect(deleteAsync).toHaveBeenCalledWith("theory-1");
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/theories/higurashi");
        });
    });

    it("closes the delete dialogue without deleting when cancelled", async () => {
        // given
        const { deleteAsync } = stubTheory();
        const user = userEvent.setup();
        renderPage(authorUser);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByRole("button", { name: "Delete Theory" })).not.toBeInTheDocument();
        expect(deleteAsync).not.toHaveBeenCalled();
    });

    it("records an upvote and moves the score straight away", async () => {
        // given
        const { voteAsync } = stubTheory();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "Upvote" }));

        // then
        expect(voteAsync).toHaveBeenCalledWith(1);
        expect(screen.getByText("6")).toBeInTheDocument();
    });

    it("puts the score back when the vote is rejected", async () => {
        // given
        stubTheory({ vote: () => Promise.reject(new Error("no")) });
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: "Downvote" }));

        // then
        await waitFor(() => {
            expect(screen.getByText("5")).toBeInTheDocument();
        });
    });

    it("goes back a step in history from the back button", async () => {
        // given
        stubTheory();
        const user = userEvent.setup();
        renderPage(reader);

        // when
        await user.click(screen.getByRole("button", { name: /Back/ }));

        // then
        expect(navigate).toHaveBeenCalledWith(-1);
    });
});
