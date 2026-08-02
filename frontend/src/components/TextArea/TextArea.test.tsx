import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { TextArea } from "./TextArea";

describe("TextArea", () => {
    it("is reachable through the label that points at it", () => {
        // given
        const markup = (
            <>
                <label htmlFor="theory">Your theory</label>
                <TextArea id="theory" />
            </>
        );

        // when
        renderWithProviders(markup);

        // then
        expect(screen.getByLabelText("Your theory")).toBeInstanceOf(HTMLTextAreaElement);
    });

    it("collects the text that is typed into it", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<TextArea placeholder="Say something" onChange={onChange} />);

        // when
        await user.type(screen.getByPlaceholderText("Say something"), "without love");

        // then
        expect(screen.getByPlaceholderText("Say something")).toHaveValue("without love");
        expect(onChange).toHaveBeenCalledTimes("without love".length);
    });

    it("keeps newlines that are typed into it", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<TextArea placeholder="Say something" />);

        // when
        await user.type(screen.getByPlaceholderText("Say something"), "red truth{enter}blue truth");

        // then
        expect(screen.getByPlaceholderText("Say something")).toHaveValue("red truth\nblue truth");
    });

    it("shows the value it is controlled with", () => {
        // given
        const onChange = vi.fn();

        // when
        renderWithProviders(<TextArea placeholder="Say something" value="the cat box" onChange={onChange} />);

        // then
        expect(screen.getByPlaceholderText("Say something")).toHaveValue("the cat box");
    });

    it("refuses input while it is disabled", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<TextArea placeholder="Say something" disabled onChange={onChange} />);

        // when
        await user.type(screen.getByPlaceholderText("Say something"), "gold");

        // then
        expect(screen.getByPlaceholderText("Say something")).toBeDisabled();
        expect(onChange).not.toHaveBeenCalled();
    });

    it("forwards native attributes to the underlying textarea", () => {
        // given
        const markup = <TextArea placeholder="Say something" rows={7} maxLength={300} required name="body" />;

        // when
        renderWithProviders(markup);

        // then
        const textarea = screen.getByPlaceholderText("Say something");
        expect(textarea).toHaveAttribute("rows", "7");
        expect(textarea).toHaveAttribute("maxlength", "300");
        expect(textarea).toHaveAttribute("name", "body");
        expect(textarea).toBeRequired();
    });

    it("stops accepting text once the maximum length is reached", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<TextArea placeholder="Say something" maxLength={4} />);

        // when
        await user.type(screen.getByPlaceholderText("Say something"), "beatrice");

        // then
        expect(screen.getByPlaceholderText("Say something")).toHaveValue("beat");
    });

    it("keeps a caller supplied class alongside its own classes", () => {
        // given
        const className = "tall-box";

        // when
        renderWithProviders(<TextArea placeholder="Say something" className={className} />);

        // then
        const textarea = screen.getByPlaceholderText("Say something");
        expect(textarea).toHaveClass(className);
        expect(textarea.className.split(" ").length).toBeGreaterThan(1);
    });
});
