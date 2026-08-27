import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TypingIndicator } from "./TypingIndicator";

describe("TypingIndicator", () => {
    it("renders nothing when nobody is typing", () => {
        // given
        const names: string[] = [];

        // when
        const { container } = render(<TypingIndicator names={names} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("uses the singular verb for a lone typist", () => {
        // given
        const names = ["Beatrice"];

        // when
        render(<TypingIndicator names={names} />);

        // then
        expect(screen.getByText(/is typing/)).toHaveTextContent("Beatrice is typing...");
    });

    it("joins two typists with and", () => {
        // given
        const names = ["Beatrice", "Battler"];

        // when
        render(<TypingIndicator names={names} />);

        // then
        expect(screen.getByText(/are typing/)).toHaveTextContent("Beatrice and Battler are typing...");
    });

    it("comma separates three typists before the final and", () => {
        // given
        const names = ["Beatrice", "Battler", "Ange"];

        // when
        render(<TypingIndicator names={names} />);

        // then
        expect(screen.getByText(/are typing/)).toHaveTextContent("Beatrice, Battler and Ange are typing...");
    });

    it("collapses to a generic phrase once there are more than three typists", () => {
        // given
        const names = ["Beatrice", "Battler", "Ange", "Maria"];

        // when
        render(<TypingIndicator names={names} />);

        // then
        expect(screen.getByText("Multiple people are typing...")).toBeInTheDocument();
        expect(screen.queryByText(/Beatrice/)).not.toBeInTheDocument();
    });

    it("hides the animated dots from assistive technology", () => {
        // given
        const names = ["Beatrice"];

        // when
        const { container } = render(<TypingIndicator names={names} />);

        // then
        expect(container.querySelector("[aria-hidden='true']")).toBeInTheDocument();
    });
});
