import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import type { PostMedia } from "../../../types/api";
import { renderWithProviders } from "../../../test-utils/render";
import { MediaGallery } from "./MediaGallery";

function makeMedia(overrides: Partial<PostMedia> = {}): PostMedia {
    return {
        id: 1,
        media_url: "https://waifuvault.moe/f/one.png",
        media_type: "image",
        sort_order: 0,
        ...overrides,
    };
}

function galleryImages(container: HTMLElement): HTMLImageElement[] {
    return Array.from(container.querySelectorAll("img"));
}

function firstVideo(container: HTMLElement): HTMLVideoElement {
    const video = container.querySelector("video");
    if (!video) {
        throw new Error("expected a video to be rendered");
    }

    return video;
}

describe("MediaGallery", () => {
    it("renders nothing when the post has no media", () => {
        // given
        const media: PostMedia[] = [];

        // when
        const { container } = renderWithProviders(<MediaGallery media={media} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("renders one lazily loaded picture per image", () => {
        // given
        const media = [
            makeMedia({ id: 1, media_url: "https://waifuvault.moe/f/one.png" }),
            makeMedia({ id: 2, media_url: "https://waifuvault.moe/f/two.png", sort_order: 1 }),
            makeMedia({ id: 3, media_url: "https://waifuvault.moe/f/three.png", sort_order: 2 }),
        ];

        // when
        const { container } = renderWithProviders(<MediaGallery media={media} />);

        // then
        const images = galleryImages(container);
        expect(images.map(img => img.getAttribute("src"))).toEqual([
            "https://waifuvault.moe/f/one.png",
            "https://waifuvault.moe/f/two.png",
            "https://waifuvault.moe/f/three.png",
        ]);
        expect(images[0]).toHaveAttribute("loading", "lazy");
    });

    it("gives a video its thumbnail as a poster", () => {
        // given
        const media = [
            makeMedia({
                id: 4,
                media_type: "video",
                media_url: "https://waifuvault.moe/f/clip.mp4",
                thumbnail_url: "https://waifuvault.moe/f/clip.png",
            }),
        ];

        // when
        const { container } = renderWithProviders(<MediaGallery media={media} />);

        // then
        const video = firstVideo(container);
        expect(video).toHaveAttribute("src", "https://waifuvault.moe/f/clip.mp4");
        expect(video).toHaveAttribute("poster", "https://waifuvault.moe/f/clip.png");
        expect(video).toHaveAttribute("controls");
    });

    it("leaves the poster off a video that has no thumbnail", () => {
        // given
        const media = [makeMedia({ id: 5, media_type: "video", media_url: "https://waifuvault.moe/f/clip.mp4" })];

        // when
        const { container } = renderWithProviders(<MediaGallery media={media} />);

        // then
        expect(firstVideo(container)).not.toHaveAttribute("poster");
    });

    it("mixes images and videos in the same gallery", () => {
        // given
        const media = [
            makeMedia({ id: 1 }),
            makeMedia({ id: 2, media_type: "video", media_url: "https://waifuvault.moe/f/clip.mp4", sort_order: 1 }),
        ];

        // when
        const { container } = renderWithProviders(<MediaGallery media={media} />);

        // then
        expect(galleryImages(container)).toHaveLength(1);
        expect(container.querySelectorAll("video")).toHaveLength(1);
    });

    it("keeps the lightbox shut until an image is clicked", () => {
        // given
        const media = [makeMedia()];

        // when
        renderWithProviders(<MediaGallery media={media} />);

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("opens the lightbox on the image that was clicked", async () => {
        // given
        const user = userEvent.setup();
        const media = [
            makeMedia({ id: 1, media_url: "https://waifuvault.moe/f/one.png" }),
            makeMedia({ id: 2, media_url: "https://waifuvault.moe/f/two.png", sort_order: 1 }),
        ];
        const { container } = renderWithProviders(<MediaGallery media={media} />);

        // when
        await user.click(galleryImages(container)[1]);

        // then
        const dialog = screen.getByRole("dialog");
        expect(dialog.querySelector("img")).toHaveAttribute("src", "https://waifuvault.moe/f/two.png");
    });

    it("closes the lightbox again", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<MediaGallery media={[makeMedia()]} />);
        await user.click(galleryImages(container)[0]);

        // when
        await user.click(screen.getByRole("button", { name: "Close" }));

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    it("does not open a lightbox for a video", async () => {
        // given
        const user = userEvent.setup();
        const media = [makeMedia({ id: 9, media_type: "video", media_url: "https://waifuvault.moe/f/clip.mp4" })];
        const { container } = renderWithProviders(<MediaGallery media={media} />);

        // when
        await user.click(firstVideo(container));

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
});
