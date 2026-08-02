import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { JournalForm } from "./JournalForm";

const TITLE_PLACEHOLDER = "e.g. My first Umineko read-through";

function noSubmit() {
    return Promise.resolve();
}

function formOf(container: HTMLElement): HTMLFormElement {
    const form = container.querySelector("form");
    if (!form) {
        throw new Error("the journal form was not rendered");
    }
    return form;
}

describe("JournalForm", () => {
    it("starts blank on the general work", () => {
        // given
        const onSubmit = vi.fn(noSubmit);

        // when
        renderWithProviders(<JournalForm submitLabel="Create" submittingLabel="Creating..." onSubmit={onSubmit} />);

        // then
        expect(screen.getByPlaceholderText(TITLE_PLACEHOLDER)).toHaveValue("");
        expect(screen.getByRole("combobox")).toHaveValue("general");
    });

    it("starts from the title and work it was given", () => {
        // given
        const onSubmit = vi.fn(noSubmit);

        // when
        renderWithProviders(
            <JournalForm
                initialTitle="Beatrice's Epitaph"
                initialWork="ciconia"
                submitLabel="Save"
                submittingLabel="Saving..."
                onSubmit={onSubmit}
            />,
        );

        // then
        expect(screen.getByPlaceholderText(TITLE_PLACEHOLDER)).toHaveValue("Beatrice's Epitaph");
        expect(screen.getByRole("combobox")).toHaveValue("ciconia");
    });

    it("offers every journal work to choose from", () => {
        // given
        const onSubmit = vi.fn(noSubmit);

        // when
        renderWithProviders(<JournalForm submitLabel="Save" submittingLabel="Saving..." onSubmit={onSubmit} />);

        // then
        const labels = screen.getAllByRole("option").map(option => option.textContent);
        expect(labels).toEqual(["General", "Umineko", "Higurashi", "Ciconia", "Higanbana", "Rose Guns Days"]);
    });

    it("keeps the submit disabled until a real title is typed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<JournalForm submitLabel="Save" submittingLabel="Saving..." onSubmit={vi.fn(noSubmit)} />);

        // when
        await user.type(screen.getByPlaceholderText(TITLE_PLACEHOLDER), "   ");

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
        await user.type(screen.getByPlaceholderText(TITLE_PLACEHOLDER), "Ep 1");
        expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("submits the trimmed title together with the chosen work", async () => {
        // given
        const onSubmit = vi.fn(noSubmit);
        const user = userEvent.setup();
        renderWithProviders(<JournalForm submitLabel="Save" submittingLabel="Saving..." onSubmit={onSubmit} />);

        // when
        await user.type(screen.getByPlaceholderText(TITLE_PLACEHOLDER), "  Legend of the Golden Witch  ");
        await user.selectOptions(screen.getByRole("combobox"), "higurashi");
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(onSubmit).toHaveBeenCalledWith({ title: "Legend of the Golden Witch", work: "higurashi" });
    });

    it("refuses to submit a title made only of spaces", async () => {
        // given
        const onSubmit = vi.fn(noSubmit);
        const user = userEvent.setup();
        const { container } = renderWithProviders(
            <JournalForm submitLabel="Save" submittingLabel="Saving..." onSubmit={onSubmit} />,
        );
        await user.type(screen.getByPlaceholderText(TITLE_PLACEHOLDER), "   ");

        // when
        fireEvent.submit(formOf(container));

        // then
        expect(onSubmit).not.toHaveBeenCalled();
    });

    it("shows the submitting label while the save is still in flight", async () => {
        // given
        let release: () => void = () => {};
        const onSubmit = vi.fn(
            () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        );
        const user = userEvent.setup();
        renderWithProviders(<JournalForm submitLabel="Save" submittingLabel="Saving..." onSubmit={onSubmit} />);
        await user.type(screen.getByPlaceholderText(TITLE_PLACEHOLDER), "Ep 2");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
        release();
    });

    it("ignores a second submit while the first one is still running", async () => {
        // given
        const onSubmit = vi.fn(() => new Promise<void>(() => {}));
        const user = userEvent.setup();
        const { container } = renderWithProviders(
            <JournalForm submitLabel="Save" submittingLabel="Saving..." onSubmit={onSubmit} />,
        );
        await user.type(screen.getByPlaceholderText(TITLE_PLACEHOLDER), "Ep 3");
        await user.click(screen.getByRole("button", { name: "Save" }));

        // when
        fireEvent.submit(formOf(container));

        // then
        expect(onSubmit).toHaveBeenCalledOnce();
    });

    it("surfaces the reason a save was rejected and lets the author retry", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.reject(new Error("You already have too many journals")));
        const user = userEvent.setup();
        renderWithProviders(<JournalForm submitLabel="Save" submittingLabel="Saving..." onSubmit={onSubmit} />);
        await user.type(screen.getByPlaceholderText(TITLE_PLACEHOLDER), "Ep 4");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(screen.getByText("You already have too many journals")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("falls back to a generic message when the failure carries no message", async () => {
        // given
        const onSubmit = vi.fn(() => Promise.reject("the golden truth denies it"));
        const user = userEvent.setup();
        renderWithProviders(<JournalForm submitLabel="Save" submittingLabel="Saving..." onSubmit={onSubmit} />);
        await user.type(screen.getByPlaceholderText(TITLE_PLACEHOLDER), "Ep 5");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(screen.getByText("Failed to save")).toBeInTheDocument();
    });
});
