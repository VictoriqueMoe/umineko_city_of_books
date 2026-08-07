import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { ToggleSwitch } from "./ToggleSwitch";

function noop() {}

describe("ToggleSwitch", () => {
    it("exposes itself as a switch named after its label", () => {
        // given
        const label = "Show spoilers";

        // when
        renderWithProviders(<ToggleSwitch enabled={false} onChange={noop} label={label} />);

        // then
        expect(screen.getByRole("switch", { name: label })).toBeInTheDocument();
    });

    it("reports that it is off when it is not enabled", () => {
        // given
        const enabled = false;

        // when
        renderWithProviders(<ToggleSwitch enabled={enabled} onChange={noop} label="Show spoilers" />);

        // then
        expect(screen.getByRole("switch", { name: "Show spoilers" })).toHaveAttribute("aria-checked", "false");
    });

    it("reports that it is on when it is enabled", () => {
        // given
        const enabled = true;

        // when
        renderWithProviders(<ToggleSwitch enabled={enabled} onChange={noop} label="Show spoilers" />);

        // then
        expect(screen.getByRole("switch", { name: "Show spoilers" })).toHaveAttribute("aria-checked", "true");
    });

    it("asks to be switched on when it is clicked while off", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<ToggleSwitch enabled={false} onChange={onChange} label="Show spoilers" />);

        // when
        await user.click(screen.getByRole("switch", { name: "Show spoilers" }));

        // then
        expect(onChange).toHaveBeenCalledWith(true);
    });

    it("asks to be switched off when it is clicked while on", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<ToggleSwitch enabled onChange={onChange} label="Show spoilers" />);

        // when
        await user.click(screen.getByRole("switch", { name: "Show spoilers" }));

        // then
        expect(onChange).toHaveBeenCalledWith(false);
    });

    it("can be operated from the keyboard", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<ToggleSwitch enabled={false} onChange={onChange} label="Show spoilers" />);

        // when
        await user.tab();
        await user.keyboard("{Enter}");
        await user.keyboard(" ");

        // then
        expect(screen.getByRole("switch", { name: "Show spoilers" })).toHaveFocus();
        expect(onChange).toHaveBeenCalledTimes(2);
        expect(onChange).toHaveBeenNthCalledWith(2, true);
    });

    it("shows the description beneath the label when one is given", () => {
        // given
        const description = "Theories will reveal later episodes";

        // when
        renderWithProviders(
            <ToggleSwitch enabled={false} onChange={noop} label="Show spoilers" description={description} />,
        );

        // then
        expect(screen.getByText("Show spoilers")).toBeInTheDocument();
        expect(screen.getByText(description)).toBeInTheDocument();
    });

    it("leaves out the description when none is given", () => {
        // given
        const description = undefined;

        // when
        renderWithProviders(
            <ToggleSwitch enabled={false} onChange={noop} label="Show spoilers" description={description} />,
        );

        // then
        expect(screen.getByRole("switch", { name: "Show spoilers" }).textContent).toBe("Show spoilers");
    });

    it("stays usable when nothing disables it", () => {
        // given
        const disabled = undefined;

        // when
        renderWithProviders(<ToggleSwitch enabled={false} onChange={noop} label="Show spoilers" disabled={disabled} />);

        // then
        expect(screen.getByRole("switch", { name: "Show spoilers" })).toBeEnabled();
    });

    it("refuses to be operated once it is disabled", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<ToggleSwitch enabled={false} onChange={onChange} label="Show spoilers" disabled />);

        // when
        await user.click(screen.getByRole("switch", { name: "Show spoilers" }));

        // then
        expect(screen.getByRole("switch", { name: "Show spoilers" })).toBeDisabled();
        expect(onChange).not.toHaveBeenCalled();
    });

    it("still reports the state it is showing while disabled", () => {
        // given
        const enabled = true;

        // when
        renderWithProviders(<ToggleSwitch enabled={enabled} onChange={noop} label="Show spoilers" disabled />);

        // then
        expect(screen.getByRole("switch", { name: "Show spoilers" })).toHaveAttribute("aria-checked", "true");
    });

    it("does not submit a surrounding form when it is toggled", async () => {
        // given
        const onSubmit = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <form onSubmit={onSubmit}>
                <ToggleSwitch enabled={false} onChange={noop} label="Show spoilers" />
            </form>,
        );

        // when
        await user.click(screen.getByRole("switch", { name: "Show spoilers" }));

        // then
        expect(onSubmit).not.toHaveBeenCalled();
    });
});
