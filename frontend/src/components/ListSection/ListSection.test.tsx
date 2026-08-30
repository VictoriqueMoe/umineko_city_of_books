import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { ListSection, type ListSectionPage } from "./ListSection";

interface Witch {
    id: string;
    name: string;
}

const witches: Witch[] = [
    { id: "beato", name: "Beatrice" },
    { id: "bern", name: "Bernkastel" },
    { id: "lambda", name: "Lambdadelta" },
];

function makePage(overrides: Partial<ListSectionPage> = {}): ListSectionPage {
    return {
        offset: 0,
        limit: 2,
        hasPrev: true,
        goNext: () => {},
        goPrev: () => {},
        ...overrides,
    };
}

function renderNames(items: Witch[]): ReactNode {
    return (
        <ul>
            {items.map(witch => (
                <li key={witch.id}>{witch.name}</li>
            ))}
        </ul>
    );
}

describe("ListSection", () => {
    it("shows the loading text and calls neither the render prop nor the pagination", () => {
        // given
        const children = vi.fn(renderNames);

        // when
        renderWithProviders(
            <ListSection
                items={witches}
                loading
                total={40}
                page={makePage()}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {children}
            </ListSection>,
        );

        // then
        expect(screen.getByText("Loading theories...")).toBeInTheDocument();
        expect(children).not.toHaveBeenCalled();
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("shows the empty text when the list came back empty", () => {
        // given
        const children = vi.fn(renderNames);

        // when
        renderWithProviders(
            <ListSection
                items={[]}
                loading={false}
                total={0}
                page={makePage()}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {children}
            </ListSection>,
        );

        // then
        expect(screen.getByText("No theories yet.")).toBeInTheDocument();
        expect(children).not.toHaveBeenCalled();
        expect(screen.queryByText("Loading theories...")).not.toBeInTheDocument();
    });

    it("hands the whole list to the render prop in a single call", () => {
        // given
        const children = vi.fn(renderNames);

        // when
        renderWithProviders(
            <ListSection
                items={witches}
                loading={false}
                total={3}
                page={null}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {children}
            </ListSection>,
        );

        // then
        expect(children).toHaveBeenCalledTimes(1);
        expect(children).toHaveBeenCalledWith(witches);
        expect(screen.getAllByRole("listitem")).toHaveLength(3);
        expect(screen.queryByText("No theories yet.")).not.toBeInTheDocument();
    });

    it("swaps the loading text for the list once the items arrive", () => {
        // given
        const { rerender } = renderWithProviders(
            <ListSection
                items={[]}
                loading
                total={0}
                page={null}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {renderNames}
            </ListSection>,
        );

        // when
        rerender(
            <ListSection
                items={witches}
                loading={false}
                total={3}
                page={null}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {renderNames}
            </ListSection>,
        );

        // then
        expect(screen.queryByText("Loading theories...")).not.toBeInTheDocument();
        expect(screen.getByText("Bernkastel")).toBeInTheDocument();
    });

    it("shows the pagination when the total runs past one page", () => {
        // given
        const page = makePage({ offset: 0, limit: 2, hasPrev: false });

        // when
        renderWithProviders(
            <ListSection
                items={witches}
                loading={false}
                total={5}
                page={page}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {renderNames}
            </ListSection>,
        );

        // then
        expect(screen.getByText("1-2 of 5")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    });

    it("works the next page out from the total rather than being told about it", () => {
        // given
        const page = makePage({ offset: 4, limit: 2 });

        // when
        renderWithProviders(
            <ListSection
                items={witches}
                loading={false}
                total={5}
                page={page}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {renderNames}
            </ListSection>,
        );

        // then
        expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Previous" })).toBeEnabled();
    });

    it("shows no pagination when everything fits on one page", () => {
        // given
        const page = makePage({ limit: 20 });

        // when
        renderWithProviders(
            <ListSection
                items={witches}
                loading={false}
                total={20}
                page={page}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {renderNames}
            </ListSection>,
        );

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("shows no pagination for a list that does not page at all", () => {
        // when
        renderWithProviders(
            <ListSection
                items={witches}
                loading={false}
                total={900}
                page={null}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {renderNames}
            </ListSection>,
        );

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("wires the pagination buttons to the page controls it was given", async () => {
        // given
        const goNext = vi.fn();
        const goPrev = vi.fn();
        renderWithProviders(
            <ListSection
                items={witches}
                loading={false}
                total={5}
                page={makePage({ offset: 2, goNext, goPrev })}
                loadingText="Loading theories..."
                emptyText="No theories yet."
            >
                {renderNames}
            </ListSection>,
        );

        // when
        await userEvent.click(screen.getByRole("button", { name: "Next" }));
        await userEvent.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(goNext).toHaveBeenCalledTimes(1);
        expect(goPrev).toHaveBeenCalledTimes(1);
    });
});
