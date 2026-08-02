import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { Input } from "./Input";

describe("Input", () => {
    it("is reachable through the label that points at it", () => {
        // given
        const markup = (
            <>
                <label htmlFor="favourite">Favourite character</label>
                <Input id="favourite" />
            </>
        );

        // when
        renderWithProviders(markup);

        // then
        expect(screen.getByLabelText("Favourite character")).toBeInstanceOf(HTMLInputElement);
    });

    it("reports every keystroke through its change handler", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Input placeholder="Your theory" onChange={onChange} />);

        // when
        await user.type(screen.getByPlaceholderText("Your theory"), "gold");

        // then
        expect(onChange).toHaveBeenCalledTimes(4);
        expect(screen.getByPlaceholderText("Your theory")).toHaveValue("gold");
    });

    it("shows the value it is controlled with and does not drift when typed into", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Input placeholder="Your theory" value="beatrice" onChange={onChange} />);

        // when
        await user.type(screen.getByPlaceholderText("Your theory"), "x");

        // then
        expect(onChange).toHaveBeenCalledOnce();
        expect(screen.getByPlaceholderText("Your theory")).toHaveValue("beatrice");
    });

    it("refuses input while it is disabled", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Input placeholder="Your theory" disabled onChange={onChange} />);

        // when
        await user.type(screen.getByPlaceholderText("Your theory"), "gold");

        // then
        expect(screen.getByPlaceholderText("Your theory")).toBeDisabled();
        expect(onChange).not.toHaveBeenCalled();
    });

    it("refuses input while it is read only", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Input placeholder="Your theory" readOnly value="sealed" onChange={onChange} />);

        // when
        await user.type(screen.getByPlaceholderText("Your theory"), "gold");

        // then
        expect(onChange).not.toHaveBeenCalled();
        expect(screen.getByPlaceholderText("Your theory")).toHaveValue("sealed");
    });

    it("forwards native validation attributes to the underlying input", () => {
        // given
        const markup = <Input type="email" required maxLength={40} autoComplete="off" placeholder="Email" />;

        // when
        renderWithProviders(markup);

        // then
        const input = screen.getByPlaceholderText("Email");
        expect(input).toHaveAttribute("type", "email");
        expect(input).toBeRequired();
        expect(input).toHaveAttribute("maxlength", "40");
        expect(input).toHaveAttribute("autocomplete", "off");
    });

    it("marks itself as invalid when it is described by an error message", () => {
        // given
        const markup = (
            <>
                <Input placeholder="Email" aria-invalid aria-describedby="email-error" />
                <span id="email-error">That address is not allowed</span>
            </>
        );

        // when
        renderWithProviders(markup);

        // then
        const input = screen.getByPlaceholderText("Email");
        expect(input).toHaveAttribute("aria-invalid", "true");
        expect(input).toHaveAccessibleDescription("That address is not allowed");
    });

    it("keeps a caller supplied class alongside its own classes", () => {
        // given
        const className = "search-field";

        // when
        renderWithProviders(<Input placeholder="Search" className={className} />);

        // then
        const input = screen.getByPlaceholderText("Search");
        expect(input).toHaveClass(className);
        expect(input.className.split(" ").length).toBeGreaterThan(1);
    });

    it("styles itself differently once it is asked to fill the width", () => {
        // given
        const markup = (
            <>
                <Input placeholder="Narrow" />
                <Input placeholder="Wide" fullWidth />
            </>
        );

        // when
        renderWithProviders(markup);

        // then
        const narrow = screen.getByPlaceholderText("Narrow");
        const wide = screen.getByPlaceholderText("Wide");
        expect(wide.className).not.toBe(narrow.className);
        expect(wide.className.split(" ").length).toBe(narrow.className.split(" ").length + 1);
    });

    it("does not leak the fullWidth flag onto the DOM element", () => {
        // given
        const markup = <Input placeholder="Wide" fullWidth />;

        // when
        renderWithProviders(markup);

        // then
        expect(screen.getByPlaceholderText("Wide")).not.toHaveAttribute("fullwidth");
    });
});
