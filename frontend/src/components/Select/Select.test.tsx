import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { Select } from "./Select";

function options() {
    return (
        <>
            <option value="beatrice">Beatrice</option>
            <option value="bernkastel">Bernkastel</option>
            <option value="lambdadelta">Lambdadelta</option>
        </>
    );
}

describe("Select", () => {
    it("renders every option it is given", () => {
        // given
        const markup = <Select aria-label="Witch">{options()}</Select>;

        // when
        renderWithProviders(markup);

        // then
        const rendered = screen.getAllByRole("option");
        expect(rendered).toHaveLength(3);
        expect(rendered.map(option => option.textContent)).toEqual(["Beatrice", "Bernkastel", "Lambdadelta"]);
    });

    it("is reachable through the label that points at it", () => {
        // given
        const markup = (
            <>
                <label htmlFor="witch">Favourite witch</label>
                <Select id="witch">{options()}</Select>
            </>
        );

        // when
        renderWithProviders(markup);

        // then
        expect(screen.getByLabelText("Favourite witch")).toBeInstanceOf(HTMLSelectElement);
    });

    it("reports the option that was chosen", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <Select aria-label="Witch" defaultValue="beatrice" onChange={onChange}>
                {options()}
            </Select>,
        );

        // when
        await user.selectOptions(screen.getByRole("combobox", { name: "Witch" }), "lambdadelta");

        // then
        expect(onChange).toHaveBeenCalledOnce();
        expect(screen.getByRole("combobox", { name: "Witch" })).toHaveValue("lambdadelta");
    });

    it("shows the value it is controlled with as the current selection", () => {
        // given
        const onChange = vi.fn();

        // when
        renderWithProviders(
            <Select aria-label="Witch" value="bernkastel" onChange={onChange}>
                {options()}
            </Select>,
        );

        // then
        expect(screen.getByRole("combobox", { name: "Witch" })).toHaveValue("bernkastel");
        expect(screen.getByRole<HTMLOptionElement>("option", { name: "Bernkastel" }).selected).toBe(true);
    });

    it("cannot be changed while it is disabled", () => {
        // given
        const onChange = vi.fn();

        // when
        renderWithProviders(
            <Select aria-label="Witch" disabled defaultValue="beatrice" onChange={onChange}>
                {options()}
            </Select>,
        );

        // then
        expect(screen.getByRole("combobox", { name: "Witch" })).toBeDisabled();
        expect(onChange).not.toHaveBeenCalled();
    });

    it("forwards native attributes to the underlying select", () => {
        // given
        const markup = (
            <Select aria-label="Witch" name="witch" required>
                {options()}
            </Select>
        );

        // when
        renderWithProviders(markup);

        // then
        const select = screen.getByRole("combobox", { name: "Witch" });
        expect(select).toHaveAttribute("name", "witch");
        expect(select).toBeRequired();
    });

    it("keeps a caller supplied class alongside its own classes", () => {
        // given
        const className = "compact-select";

        // when
        renderWithProviders(
            <Select aria-label="Witch" className={className}>
                {options()}
            </Select>,
        );

        // then
        const select = screen.getByRole("combobox", { name: "Witch" });
        expect(select).toHaveClass(className);
        expect(select.className.split(" ").length).toBeGreaterThan(1);
    });
});
