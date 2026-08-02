import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { WaifuvaultEmbed } from "./WaifuvaultEmbed";

const IMAGE_URL = "https://waifuvault.moe/f/beatrice.png";
const VIDEO_URL = "https://waifuvault.moe/f/beatrice.mp4";

function onlyImage(container: HTMLElement): HTMLImageElement {
    const img = container.querySelector("img");
    if (!img) {
        throw new Error("expected an image to be rendered");
    }

    return img;
}

function onlyVideo(container: HTMLElement): HTMLVideoElement {
    const video = container.querySelector("video");
    if (!video) {
        throw new Error("expected a video to be rendered");
    }

    return video;
}

describe("WaifuvaultEmbed", () => {
    it("shows a controllable player for video media", () => {
        // given
        const kind = "video" as const;

        // when
        const { container } = renderWithProviders(<WaifuvaultEmbed url={VIDEO_URL} kind={kind} />);

        // then
        const video = onlyVideo(container);
        expect(video).toHaveAttribute("src", VIDEO_URL);
        expect(video).toHaveAttribute("controls");
        expect(video).toHaveAttribute("preload", "metadata");
        expect(container.querySelector("img")).toBeNull();
    });

    it("shows a lazily loaded image for image media", () => {
        // given
        const kind = "image" as const;

        // when
        const { container } = renderWithProviders(<WaifuvaultEmbed url={IMAGE_URL} kind={kind} />);

        // then
        const img = onlyImage(container);
        expect(img).toHaveAttribute("src", IMAGE_URL);
        expect(img).toHaveAttribute("loading", "lazy");
        expect(container.querySelector("video")).toBeNull();
    });

    it("keeps the lightbox closed until the image is clicked", () => {
        // given
        const kind = "image" as const;

        // when
        renderWithProviders(<WaifuvaultEmbed url={IMAGE_URL} kind={kind} />);

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("opens the image in a lightbox when it is clicked", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<WaifuvaultEmbed url={IMAGE_URL} kind="image" />);

        // when
        await user.click(onlyImage(container));

        // then
        const dialog = screen.getByRole("dialog");
        expect(dialog).toBeInTheDocument();
        expect(dialog.querySelector("img")).toHaveAttribute("src", IMAGE_URL);
    });

    it("closes the lightbox from its close control", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<WaifuvaultEmbed url={IMAGE_URL} kind="image" />);
        await user.click(onlyImage(container));

        // when
        await user.click(screen.getByRole("button", { name: "Close" }));

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("closes the lightbox when escape is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<WaifuvaultEmbed url={IMAGE_URL} kind="image" />);
        await user.click(onlyImage(container));

        // when
        await user.keyboard("{Escape}");

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("never opens a lightbox for a video", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<WaifuvaultEmbed url={VIDEO_URL} kind="video" />);

        // when
        await user.click(onlyVideo(container));

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
});
