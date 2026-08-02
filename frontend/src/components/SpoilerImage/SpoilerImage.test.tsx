import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { SpoilerImage } from "./SpoilerImage";

const SRC = "https://waifuvault.moe/f/beatrice.png";

describe("SpoilerImage", () => {
    it("shows an ordinary image with no cover when it is not a spoiler", () => {
        // given
        const isSpoiler = false;

        // when
        renderWithProviders(<SpoilerImage src={SRC} alt="the golden truth" isSpoiler={isSpoiler} />);

        // then
        expect(screen.getByAltText("the golden truth")).toHaveAttribute("src", SRC);
        expect(screen.queryByText("Spoiler")).not.toBeInTheDocument();
    });

    it("covers a spoiler with a prompt to reveal it", () => {
        // given
        const isSpoiler = true;

        // when
        renderWithProviders(<SpoilerImage src={SRC} alt="the golden truth" isSpoiler={isSpoiler} />);

        // then
        expect(screen.getByText("Spoiler")).toBeInTheDocument();
        expect(screen.getByText("Click to reveal")).toBeInTheDocument();
    });

    it("uses an empty alt text by default", () => {
        // given
        const isSpoiler = false;

        // when
        const { container } = renderWithProviders(<SpoilerImage src={SRC} isSpoiler={isSpoiler} />);

        // then
        expect(container.querySelector("img")).toHaveAttribute("alt", "");
    });

    it("reveals the image on the first click instead of calling onClick", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<SpoilerImage src={SRC} alt="the golden truth" isSpoiler onClick={onClick} />);

        // when
        await user.click(screen.getByAltText("the golden truth"));

        // then
        expect(screen.queryByText("Spoiler")).not.toBeInTheDocument();
        expect(onClick).not.toHaveBeenCalled();
    });

    it("calls onClick once the spoiler has been revealed", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<SpoilerImage src={SRC} alt="the golden truth" isSpoiler onClick={onClick} />);
        await user.click(screen.getByAltText("the golden truth"));

        // when
        await user.click(screen.getByAltText("the golden truth"));

        // then
        expect(onClick).toHaveBeenCalledOnce();
    });

    it("calls onClick straight away when nothing is hidden", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<SpoilerImage src={SRC} alt="the golden truth" isSpoiler={false} onClick={onClick} />);

        // when
        await user.click(screen.getByAltText("the golden truth"));

        // then
        expect(onClick).toHaveBeenCalledOnce();
    });

    it("stays clickable without an onClick handler", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<SpoilerImage src={SRC} alt="the golden truth" isSpoiler />);

        // when
        await user.click(screen.getByAltText("the golden truth"));
        await user.click(screen.getByAltText("the golden truth"));

        // then
        expect(screen.queryByText("Spoiler")).not.toBeInTheDocument();
    });

    it("reports a broken image through onError", () => {
        // given
        const onError = vi.fn();
        renderWithProviders(<SpoilerImage src={SRC} alt="the golden truth" isSpoiler={false} onError={onError} />);

        // when
        fireEvent.error(screen.getByAltText("the golden truth"));

        // then
        expect(onError).toHaveBeenCalledOnce();
    });
});
