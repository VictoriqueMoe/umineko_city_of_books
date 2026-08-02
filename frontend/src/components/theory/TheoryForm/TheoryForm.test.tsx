import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { Quote } from "../../../types/api";
import { TheoryForm } from "./TheoryForm";

const { pickerQuote } = vi.hoisted(() => ({
    pickerQuote: {
        text: "The seven stakes of purgatory are the culprits.",
        textHtml: "",
        characterId: "battler",
        character: "Battler",
        audioId: "ep2_0042",
        episode: 2,
        contentType: "dialogue",
        hasRedTruth: true,
        hasBlueTruth: false,
        hasGoldTruth: false,
        hasPurpleTruth: false,
        index: 42,
    },
}));

vi.mock("../../truth/TruthPicker/TruthPicker", () => ({
    TruthPicker: ({ isOpen, onSelect }: { isOpen: boolean; onSelect: (quote: Quote, lang: string) => void }) =>
        isOpen ? (
            <button type="button" onClick={() => onSelect(pickerQuote, "en")}>
                pick the stakes
            </button>
        ) : null,
}));

function formOf(element: HTMLElement): HTMLFormElement {
    const form = element.closest("form");
    if (!form) {
        throw new Error("expected the field to sit inside a form");
    }

    return form;
}

function titleField(): HTMLElement {
    return screen.getByPlaceholderText("Theory title...");
}

function bodyField(): HTMLElement {
    return screen.getByPlaceholderText("State your theory...");
}

function renderForm(onSubmit: (data: unknown) => Promise<void>, series?: "umineko" | "higurashi") {
    return renderWithProviders(
        <TheoryForm submitLabel="Publish" submittingLabel="Publishing..." series={series} onSubmit={onSubmit} />,
    );
}

describe("TheoryForm", () => {
    it("keeps the submit button disabled until a title and a body are both written", async () => {
        // given
        const user = userEvent.setup();
        renderForm(vi.fn(() => Promise.resolve()));

        // when
        await user.type(titleField(), "Kanon is the culprit");

        // then
        expect(screen.getByRole("button", { name: "Publish" })).toBeDisabled();
        await user.type(bodyField(), "He was never there.");
        expect(screen.getByRole("button", { name: "Publish" })).toBeEnabled();
    });

    it("refuses a submission whose title is only whitespace", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderForm(onSubmit);
        await user.type(titleField(), "   ");
        await user.type(bodyField(), "He was never there.");

        // when
        fireEvent.submit(formOf(titleField()));

        // then
        expect(onSubmit).not.toHaveBeenCalled();
    });

    it("refuses a submission whose body is only whitespace", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderForm(onSubmit);
        await user.type(titleField(), "Kanon is the culprit");
        await user.type(bodyField(), "   ");

        // when
        fireEvent.submit(formOf(titleField()));

        // then
        expect(onSubmit).not.toHaveBeenCalled();
    });

    it("submits the trimmed title and body with the chosen episode", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderForm(onSubmit);

        // when
        await user.type(titleField(), "  Kanon is the culprit  ");
        await user.type(bodyField(), "  He was never there.  ");
        await user.selectOptions(screen.getByRole("combobox"), "3");
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(onSubmit).toHaveBeenCalledExactlyOnceWith({
            title: "Kanon is the culprit",
            body: "He was never there.",
            episode: 3,
            evidence: [],
        });
    });

    it("defaults to a general theory when no episode is chosen", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderForm(onSubmit);

        // when
        await user.type(titleField(), "A general truth");
        await user.type(bodyField(), "It applies everywhere.");
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(screen.getByRole("option", { name: "General (no specific episode)" })).toBeInTheDocument();
        expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ episode: 0 }));
    });

    it("shows the submitting label and blocks a second submission while one is in flight", async () => {
        // given
        let release: () => void = () => {};
        const onSubmit = vi.fn(
            () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        );
        const user = userEvent.setup();
        renderForm(onSubmit);
        await user.type(titleField(), "Kanon is the culprit");
        await user.type(bodyField(), "He was never there.");

        // when
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(screen.getByRole("button", { name: "Publishing..." })).toBeDisabled();
        fireEvent.submit(formOf(titleField()));
        expect(onSubmit).toHaveBeenCalledOnce();
        release();
        await waitFor(() => expect(screen.getByRole("button", { name: "Publish" })).toBeEnabled());
    });

    it("starts from the values it was given when a theory is being edited", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderWithProviders(
            <TheoryForm
                initialTitle="Beatrice exists"
                initialBody="She is the golden witch."
                initialEpisode={4}
                submitLabel="Save"
                submittingLabel="Saving..."
                onSubmit={onSubmit}
            />,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(titleField()).toHaveValue("Beatrice exists");
        expect(screen.getByRole("combobox")).toHaveValue("4");
        expect(onSubmit).toHaveBeenCalledWith({
            title: "Beatrice exists",
            body: "She is the golden witch.",
            episode: 4,
            evidence: [],
        });
    });

    it("offers higurashi arcs instead of episode numbers and submits their position", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderForm(onSubmit, "higurashi");

        // when
        await user.type(titleField(), "Keiichi is innocent");
        await user.type(bodyField(), "The syndrome explains it.");
        await user.selectOptions(screen.getByRole("combobox"), "2");
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(screen.getByRole("option", { name: "General (no specific arc)" })).toBeInTheDocument();
        expect(screen.getByRole("option", { name: "Watanagashi" })).toBeInTheDocument();
        expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ episode: 2 }));
    });

    it("attaches a quote chosen from the picker together with its note", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderForm(onSubmit);
        await user.type(titleField(), "Kanon is the culprit");
        await user.type(bodyField(), "He was never there.");

        // when
        await user.click(screen.getByRole("button", { name: "+ Attach Evidence" }));
        await user.click(screen.getByRole("button", { name: "pick the stakes" }));
        await user.type(screen.getByPlaceholderText("Why is this relevant?"), "Battler says so in red");
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(screen.getByText(pickerQuote.text)).toBeInTheDocument();
        expect(onSubmit).toHaveBeenCalledWith(
            expect.objectContaining({
                evidence: [
                    { audio_id: "ep2_0042", quote_index: undefined, note: "Battler says so in red", lang: "en" },
                ],
            }),
        );
    });

    it("ignores a quote that has already been attached", async () => {
        // given
        const user = userEvent.setup();
        renderForm(vi.fn(() => Promise.resolve()));

        // when
        await user.click(screen.getByRole("button", { name: "+ Attach Evidence" }));
        await user.click(screen.getByRole("button", { name: "pick the stakes" }));
        await user.click(screen.getByRole("button", { name: "+ Attach Evidence" }));
        await user.click(screen.getByRole("button", { name: "pick the stakes" }));

        // then
        expect(screen.getAllByPlaceholderText("Why is this relevant?")).toHaveLength(1);
    });

    it("drops an attached quote when it is removed again", async () => {
        // given
        const user = userEvent.setup();
        renderForm(vi.fn(() => Promise.resolve()));
        await user.click(screen.getByRole("button", { name: "+ Attach Evidence" }));
        await user.click(screen.getByRole("button", { name: "pick the stakes" }));

        // when
        await user.click(screen.getByRole("button", { name: "✕" }));

        // then
        expect(screen.queryByText(pickerQuote.text)).not.toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Why is this relevant?")).not.toBeInTheDocument();
    });
});
