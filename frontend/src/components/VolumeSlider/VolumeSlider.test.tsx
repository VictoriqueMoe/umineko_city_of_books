import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { VolumeSlider } from "./VolumeSlider";

const MUTED_ICON = "\u{1F507}";
const SOUND_ICON = "\u{1F50A}";

describe("VolumeSlider", () => {
    it("labels the slider Volume when no label is given", () => {
        // given
        const value = 0.5;

        // when
        renderWithProviders(<VolumeSlider value={value} onChange={vi.fn()} />);

        // then
        expect(screen.getByRole("slider", { name: "Volume" })).toBeInTheDocument();
    });

    it("uses the accessible label it was given", () => {
        // given
        const ariaLabel = "Stream volume";

        // when
        renderWithProviders(<VolumeSlider value={0.5} onChange={vi.fn()} ariaLabel={ariaLabel} />);

        // then
        expect(screen.getByRole("slider", { name: "Stream volume" })).toBeInTheDocument();
    });

    it("shows a muted icon only when the volume is all the way down", () => {
        // given
        const value = 0;

        // when
        const { rerender } = renderWithProviders(<VolumeSlider value={value} onChange={vi.fn()} />);

        // then
        expect(screen.getByText(MUTED_ICON)).toBeInTheDocument();
        expect(screen.queryByText(SOUND_ICON)).not.toBeInTheDocument();
        rerender(<VolumeSlider value={0.01} onChange={vi.fn()} />);
        expect(screen.getByText(SOUND_ICON)).toBeInTheDocument();
        expect(screen.queryByText(MUTED_ICON)).not.toBeInTheDocument();
    });

    it("reflects the volume it was given", () => {
        // given
        const value = 0.35;

        // when
        renderWithProviders(<VolumeSlider value={value} onChange={vi.fn()} />);

        // then
        expect(screen.getByRole("slider")).toHaveValue("0.35");
    });

    it("covers the whole range in fine steps", () => {
        // given
        const value = 0.5;

        // when
        renderWithProviders(<VolumeSlider value={value} onChange={vi.fn()} />);

        // then
        const slider = screen.getByRole("slider");
        expect(slider).toHaveAttribute("min", "0");
        expect(slider).toHaveAttribute("max", "1");
        expect(slider).toHaveAttribute("step", "0.01");
    });

    it("reports the new volume as a number rather than a string", () => {
        // given
        const onChange = vi.fn();
        renderWithProviders(<VolumeSlider value={0.5} onChange={onChange} />);

        // when
        fireEvent.change(screen.getByRole("slider"), { target: { value: "0.42" } });

        // then
        expect(onChange).toHaveBeenCalledWith(0.42);
    });

    it("reports zero when the slider is dragged all the way down", () => {
        // given
        const onChange = vi.fn();
        renderWithProviders(<VolumeSlider value={0.5} onChange={onChange} />);

        // when
        fireEvent.change(screen.getByRole("slider"), { target: { value: "0" } });

        // then
        expect(onChange).toHaveBeenCalledWith(0);
    });
});
