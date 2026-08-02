import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { Pagination } from "./Pagination";

function noop() {}

describe("Pagination", () => {
    it("renders nothing when there is nothing to page through", () => {
        // given
        const total = 0;

        // when
        const { container } = renderWithProviders(
            <Pagination
                offset={0}
                limit={10}
                total={total}
                hasNext={false}
                hasPrev={false}
                onNext={noop}
                onPrev={noop}
            />,
        );

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows the one-based range of the current page", () => {
        // given
        const offset = 20;

        // when
        renderWithProviders(
            <Pagination offset={offset} limit={10} total={57} hasNext hasPrev onNext={noop} onPrev={noop} />,
        );

        // then
        expect(screen.getByText("21-30 of 57")).toBeInTheDocument();
    });

    it("clamps the upper bound of the last page to the total", () => {
        // given
        const offset = 50;

        // when
        renderWithProviders(
            <Pagination offset={offset} limit={10} total={57} hasNext={false} hasPrev onNext={noop} onPrev={noop} />,
        );

        // then
        expect(screen.getByText("51-57 of 57")).toBeInTheDocument();
    });

    it("disables the previous control on the first page", () => {
        // given
        const hasPrev = false;

        // when
        renderWithProviders(
            <Pagination offset={0} limit={10} total={57} hasNext hasPrev={hasPrev} onNext={noop} onPrev={noop} />,
        );

        // then
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
    });

    it("calls onNext when the next control is pressed", async () => {
        // given
        const onNext = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <Pagination offset={0} limit={10} total={57} hasNext hasPrev={false} onNext={onNext} onPrev={noop} />,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(onNext).toHaveBeenCalledOnce();
    });

    it("only offers first and last controls when handlers are supplied", () => {
        // given
        const onFirst = vi.fn();

        // when
        renderWithProviders(
            <Pagination
                offset={20}
                limit={10}
                total={57}
                hasNext
                hasPrev
                onNext={noop}
                onPrev={noop}
                onFirst={onFirst}
            />,
        );

        // then
        expect(screen.getByRole("button", { name: "« First" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Last »" })).not.toBeInTheDocument();
    });
});
