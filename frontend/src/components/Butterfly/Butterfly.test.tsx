import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Butterfly } from "./Butterfly";

function svgOf(container: HTMLElement): SVGSVGElement {
    const svg = container.querySelector("svg");
    if (!svg) {
        throw new Error("expected a butterfly svg to be rendered");
    }

    return svg;
}

describe("Butterfly", () => {
    it("draws at sixteen pixels when no size is asked for", () => {
        // given
        const colour = "#ff3333";

        // when
        const { container } = render(<Butterfly colour={colour} />);

        // then
        const svg = svgOf(container);
        expect(svg).toHaveAttribute("width", "16");
        expect(svg).toHaveAttribute("height", "16");
    });

    it("draws at the size it was given", () => {
        // given
        const size = 42;

        // when
        const { container } = render(<Butterfly colour="#ff3333" size={size} />);

        // then
        const svg = svgOf(container);
        expect(svg).toHaveAttribute("width", "42");
        expect(svg).toHaveAttribute("height", "42");
    });

    it("keeps the viewbox fixed so any size stays in proportion", () => {
        // given
        const size = 128;

        // when
        const { container } = render(<Butterfly colour="#3399ff" size={size} />);

        // then
        expect(svgOf(container)).toHaveAttribute("viewBox", "0 0 24 24");
    });

    it("paints the wings, the body and the head in the colour it was given", () => {
        // given
        const colour = "#aa71ff";

        // when
        const { container } = render(<Butterfly colour={colour} />);

        // then
        const wings = container.querySelector("g");
        expect(wings).toHaveAttribute("fill", colour);
        expect(wings).toHaveAttribute("stroke", colour);
        expect(container.querySelector("g + path")).toHaveAttribute("stroke", colour);
        expect(container.querySelector("circle")).toHaveAttribute("fill", colour);
    });

    it("draws one path per wing", () => {
        // given
        const colour = "#3ed47a";

        // when
        const { container } = render(<Butterfly colour={colour} />);

        // then
        expect(container.querySelectorAll("g path")).toHaveLength(2);
    });

    it("hides itself from assistive technology and from the tab order", () => {
        // given
        const colour = "#ffaa00";

        // when
        const { container } = render(<Butterfly colour={colour} />);

        // then
        const svg = svgOf(container);
        expect(svg).toHaveAttribute("aria-hidden", "true");
        expect(svg).toHaveAttribute("focusable", "false");
    });

    it("carries the class name it was given", () => {
        // given
        const className = "floating";

        // when
        const { container } = render(<Butterfly colour="#ebcdf0" className={className} />);

        // then
        expect(svgOf(container)).toHaveClass(className);
    });
});
