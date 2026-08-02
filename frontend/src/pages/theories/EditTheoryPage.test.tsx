import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { EvidenceInput, EvidenceItem, TheoryDetail } from "../../types/api";
import { EditTheoryPage } from "./EditTheoryPage";

const { useTheory, useUpdateTheory, navigate } = vi.hoisted(() => ({
    useTheory: vi.fn(),
    useUpdateTheory: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/theory", () => ({ useTheory }));
vi.mock("../../api/mutations/theory", () => ({ useUpdateTheory }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

interface TheoryFormStubProps {
    initialTitle?: string;
    initialBody?: string;
    initialEpisode?: number;
    initialEvidence?: EvidenceItem[];
    submitLabel: string;
    submittingLabel: string;
    series?: string;
    onSubmit: (data: { title: string; body: string; episode: number; evidence: EvidenceInput[] }) => Promise<void>;
}

vi.mock("../../components/theory/TheoryForm/TheoryForm", () => ({
    TheoryForm: (props: TheoryFormStubProps) => (
        <section aria-label="theory form">
            <p>{`title: ${props.initialTitle}`}</p>
            <p>{`body: ${props.initialBody}`}</p>
            <p>{`episode: ${props.initialEpisode}`}</p>
            <p>{`evidence: ${props.initialEvidence?.length ?? 0}`}</p>
            <p>{`${props.submitLabel} / ${props.submittingLabel} / ${props.series}`}</p>
            <button
                onClick={() => {
                    props
                        .onSubmit({
                            title: "A revised blue truth",
                            body: "Now with more love.",
                            episode: 6,
                            evidence: [],
                        })
                        .catch(() => {});
                }}
            >
                save
            </button>
        </section>
    ),
}));

function makeTheoryDetail(overrides: Partial<TheoryDetail> = {}): TheoryDetail {
    return {
        id: "theory-1",
        title: "The culprit is Kanon",
        body: "Without love it cannot be seen.",
        episode: 3,
        series: "umineko",
        author: { id: "author-1", username: "beatrice", display_name: "Beatrice" },
        vote_score: 5,
        with_love_count: 1,
        without_love_count: 0,
        credibility_score: 55,
        created_at: "2026-07-01T10:00:00Z",
        evidence: [{ id: 7, note: "a quote", lang: "en", sort_order: 0 }],
        responses: [],
        ...overrides,
    };
}

interface StubOptions {
    theory?: TheoryDetail | null;
    loading?: boolean;
    update?: () => Promise<unknown>;
}

function stubTheory(options: StubOptions = {}) {
    useTheory.mockReturnValue({
        theory: options.theory === undefined ? makeTheoryDetail() : options.theory,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
    const mutateAsync = vi.fn(options.update ?? (() => Promise.resolve({})));
    useUpdateTheory.mockReturnValue({ mutateAsync });

    return { mutateAsync };
}

function renderPage() {
    return renderWithProviders(<EditTheoryPage />, { route: "/theory/theory-1/edit", path: "/theory/:id/edit" });
}

describe("EditTheoryPage", () => {
    it("waits on the form while the theory is still loading", () => {
        // given
        stubTheory({ loading: true, theory: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading theory...")).toBeInTheDocument();
        expect(screen.queryByRole("region", { name: "theory form" })).not.toBeInTheDocument();
    });

    it("sends the editor home when the theory does not exist", async () => {
        // given
        stubTheory({ theory: null });

        // when
        renderPage();

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/");
        });
    });

    it("does not send the editor home while the theory is still loading", () => {
        // given
        stubTheory({ loading: true, theory: null });

        // when
        renderPage();

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("seeds the form with the theory as it stands", () => {
        // given
        stubTheory();

        // when
        renderPage();

        // then
        expect(screen.getByText("title: The culprit is Kanon")).toBeInTheDocument();
        expect(screen.getByText("body: Without love it cannot be seen.")).toBeInTheDocument();
        expect(screen.getByText("episode: 3")).toBeInTheDocument();
        expect(screen.getByText("evidence: 1")).toBeInTheDocument();
        expect(screen.getByText("Save Changes / Saving... / umineko")).toBeInTheDocument();
    });

    it("keeps the theory on its own series when editing", () => {
        // given
        stubTheory({ theory: makeTheoryDetail({ series: "higurashi" }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("Save Changes / Saving... / higurashi")).toBeInTheDocument();
    });

    it("falls back to umineko when the theory has no series recorded", () => {
        // given
        stubTheory({ theory: makeTheoryDetail({ series: "" }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("Save Changes / Saving... / umineko")).toBeInTheDocument();
    });

    it("saves the edited theory with its series attached", async () => {
        // given
        const { mutateAsync } = stubTheory({ theory: makeTheoryDetail({ series: "ciconia" }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "save" }));

        // then
        expect(mutateAsync).toHaveBeenCalledWith({
            title: "A revised blue truth",
            body: "Now with more love.",
            episode: 6,
            evidence: [],
            series: "ciconia",
        });
    });

    it("returns to the theory once the edit is saved", async () => {
        // given
        stubTheory();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "save" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/theory/theory-1");
        });
    });

    it("stays on the editor when the save is rejected", async () => {
        // given
        const { mutateAsync } = stubTheory({ update: () => Promise.reject(new Error("nope")) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "save" }));

        // then
        expect(mutateAsync).toHaveBeenCalledOnce();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("goes back a step in history when the edit is cancelled", async () => {
        // given
        stubTheory();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: /Cancel/ }));

        // then
        expect(navigate).toHaveBeenCalledWith(-1);
    });
});
