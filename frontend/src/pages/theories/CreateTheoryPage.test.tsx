import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { EvidenceInput } from "../../types/api";
import { CreateTheoryPage } from "./CreateTheoryPage";

const { useCreateTheory, navigate } = vi.hoisted(() => ({
    useCreateTheory: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/mutations/theory", () => ({ useCreateTheory }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});
vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: ({ page }: { page: string }) => <div data-testid="rules-box">{page}</div>,
}));

interface TheoryFormStubProps {
    submitLabel: string;
    submittingLabel: string;
    series?: string;
    onSubmit: (data: { title: string; body: string; episode: number; evidence: EvidenceInput[] }) => Promise<void>;
}

vi.mock("../../components/theory/TheoryForm/TheoryForm", () => ({
    TheoryForm: (props: TheoryFormStubProps) => (
        <section aria-label="theory form">
            <p>{`${props.submitLabel} / ${props.submittingLabel} / ${props.series}`}</p>
            <button
                onClick={() => {
                    props
                        .onSubmit({
                            title: "The culprit is Kanon",
                            body: "Without love it cannot be seen.",
                            episode: 4,
                            evidence: [{ note: "a red truth" }],
                        })
                        .catch(() => {});
                }}
            >
                declare
            </button>
        </section>
    ),
}));

function stubCreate(result: { id: string } = { id: "theory-9" }) {
    const mutateAsync = vi.fn(() => Promise.resolve(result));
    useCreateTheory.mockReturnValue({ mutateAsync });

    return { mutateAsync };
}

describe("CreateTheoryPage", () => {
    it("invites the visitor to declare a blue truth", () => {
        // given
        stubCreate();

        // when
        renderWithProviders(<CreateTheoryPage />, { route: "/theory/new" });

        // then
        expect(screen.getByRole("heading", { name: "Declare Your Blue Truth" })).toBeInTheDocument();
        expect(screen.getByText("Declare Blue Truth / Declaring... / umineko")).toBeInTheDocument();
    });

    it("shows the umineko theory rules by default", () => {
        // given
        stubCreate();

        // when
        renderWithProviders(<CreateTheoryPage />, { route: "/theory/new" });

        // then
        expect(screen.getByTestId("rules-box")).toHaveTextContent("theories");
    });

    it("shows the rules of whichever series the composer belongs to", () => {
        // given
        stubCreate();

        // when
        renderWithProviders(<CreateTheoryPage series="ciconia" />, { route: "/theory/ciconia/new" });

        // then
        expect(screen.getByTestId("rules-box")).toHaveTextContent("theories_ciconia");
        expect(screen.getByText("Declare Blue Truth / Declaring... / ciconia")).toBeInTheDocument();
    });

    it("stamps the series onto the payload it sends", async () => {
        // given
        const { mutateAsync } = stubCreate();
        const user = userEvent.setup();
        renderWithProviders(<CreateTheoryPage series="higurashi" />, { route: "/theory/higurashi/new" });

        // when
        await user.click(screen.getByRole("button", { name: "declare" }));

        // then
        expect(mutateAsync).toHaveBeenCalledWith({
            title: "The culprit is Kanon",
            body: "Without love it cannot be seen.",
            episode: 4,
            evidence: [{ note: "a red truth" }],
            series: "higurashi",
        });
    });

    it("opens the newly declared theory once the server accepts it", async () => {
        // given
        stubCreate({ id: "theory-42" });
        const user = userEvent.setup();
        renderWithProviders(<CreateTheoryPage />, { route: "/theory/new" });

        // when
        await user.click(screen.getByRole("button", { name: "declare" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/theory/theory-42");
        });
    });

    it("stays on the composer when the declaration is rejected", async () => {
        // given
        const mutateAsync = vi.fn(() => Promise.reject(new Error("the witch refuses")));
        useCreateTheory.mockReturnValue({ mutateAsync });
        const user = userEvent.setup();
        renderWithProviders(<CreateTheoryPage />, { route: "/theory/new" });

        // when
        await user.click(screen.getByRole("button", { name: "declare" }));

        // then
        expect(mutateAsync).toHaveBeenCalledOnce();
        expect(navigate).not.toHaveBeenCalled();
    });
});
