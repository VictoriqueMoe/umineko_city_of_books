import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RootErrorBoundary } from "./RootErrorBoundary";

let thrown: unknown = null;

function Fragile() {
    if (thrown !== null) {
        throw thrown;
    }

    return <p>The board is intact</p>;
}

beforeEach(() => {
    thrown = null;
    vi.spyOn(console, "error").mockImplementation(() => {});
});

describe("RootErrorBoundary", () => {
    it("stays out of the way while nothing has gone wrong", () => {
        // given
        const onError = vi.fn();

        // when
        render(
            <RootErrorBoundary onError={onError}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // then
        expect(screen.getByText("The board is intact")).toBeInTheDocument();
        expect(onError).not.toHaveBeenCalled();
    });

    it("shows a fallback instead of a blank page when a child throws", () => {
        // given
        thrown = new Error("the golden land is closed");

        // when
        render(
            <RootErrorBoundary onError={vi.fn()}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // then
        expect(screen.getByRole("heading", { name: "The game board has shattered" })).toBeInTheDocument();
        expect(screen.queryByText("The board is intact")).not.toBeInTheDocument();
    });

    it("hands the error and the component stack to the reporter it was given", () => {
        // given
        thrown = new Error("the golden land is closed");
        const onError = vi.fn();

        // when
        render(
            <RootErrorBoundary onError={onError}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // then
        expect(onError).toHaveBeenCalledTimes(1);
        expect(onError).toHaveBeenCalledWith(thrown, expect.stringContaining("Fragile"));
    });

    it("shows what the error said so the reader can quote it in a report", () => {
        // given
        thrown = new TypeError("witches do not exist");

        // when
        render(
            <RootErrorBoundary onError={vi.fn()}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // then
        expect(screen.getByText("TypeError: witches do not exist")).toBeInTheDocument();
    });

    it("says nothing about a thrown value it cannot describe", () => {
        // given
        thrown = 1986;

        // when
        render(
            <RootErrorBoundary onError={vi.fn()}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // then
        expect(screen.getByRole("heading", { name: "The game board has shattered" })).toBeInTheDocument();
        expect(screen.queryByText("1986")).not.toBeInTheDocument();
    });

    it("gives the reader their work back when they try again", async () => {
        // given
        thrown = new Error("the golden land is closed");
        render(
            <RootErrorBoundary onError={vi.fn()}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // when
        thrown = null;
        await userEvent.click(screen.getByRole("button", { name: "Try again" }));

        // then
        expect(screen.getByText("The board is intact")).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "The game board has shattered" })).not.toBeInTheDocument();
    });

    it("offers a reload for when trying again is not enough", async () => {
        // given
        thrown = new Error("the golden land is closed");
        const reload = vi.fn();
        vi.stubGlobal("location", { ...globalThis.location, reload });
        render(
            <RootErrorBoundary onError={vi.fn()}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // when
        await userEvent.click(screen.getByRole("button", { name: "Reload the page" }));

        // then
        expect(reload).toHaveBeenCalledTimes(1);
    });

    it("offers the way back to the city of books", () => {
        // given
        thrown = new Error("the golden land is closed");

        // when
        render(
            <RootErrorBoundary onError={vi.fn()}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // then
        expect(screen.getByRole("link", { name: "Back to the City of Books" })).toHaveAttribute("href", "/");
    });

    it("still shows the fallback when the reporter itself throws", () => {
        // given
        thrown = new Error("the golden land is closed");
        const onError = vi.fn(() => {
            throw new Error("the sink is gone too");
        });

        // when
        render(
            <RootErrorBoundary onError={onError}>
                <Fragile />
            </RootErrorBoundary>,
        );

        // then
        expect(screen.getByRole("heading", { name: "The game board has shattered" })).toBeInTheDocument();
    });
});
