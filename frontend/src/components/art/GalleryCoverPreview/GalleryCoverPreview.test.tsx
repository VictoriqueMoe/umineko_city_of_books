import { fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { makeGallery as makeContentGallery, makePublicUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { Gallery } from "../../../types/api";
import { GalleryCoverPreview } from "./GalleryCoverPreview";

function makeGallery(overrides: Partial<Gallery> = {}): Gallery {
    return makeContentGallery({ author: makePublicUser({ id: "beatrice-id" }), ...overrides });
}

function images(): HTMLImageElement[] {
    return screen.getAllByRole("presentation");
}

afterEach(() => {
    vi.restoreAllMocks();
});

describe("GalleryCoverPreview", () => {
    it("prefers the cover thumbnail over the full cover", () => {
        // given
        const gallery = makeGallery({ cover_image_url: "/cover-full.png", cover_thumbnail_url: "/cover-thumb.png" });

        // when
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // then
        expect(images()[0]).toHaveAttribute("src", "/cover-thumb.png");
    });

    it("shows the full cover when there is no thumbnail of it", () => {
        // given
        const gallery = makeGallery({ cover_image_url: "/cover-full.png" });

        // when
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // then
        expect(images()[0]).toHaveAttribute("src", "/cover-full.png");
    });

    it("falls back to the full cover when the thumbnail will not load", () => {
        // given
        const gallery = makeGallery({ cover_image_url: "/cover-full.png", cover_thumbnail_url: "/cover-thumb.png" });
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // when
        fireEvent.error(images()[0]);

        // then
        expect(images()[0]).toHaveAttribute("src", "/cover-full.png");
    });

    it("swaps in the fallback once and never again, so a fallback that also fails cannot loop", () => {
        // given
        const gallery = makeGallery({ cover_image_url: "/cover-full.png", cover_thumbnail_url: "/cover-thumb.png" });
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);
        const setSrc = vi.spyOn(HTMLImageElement.prototype, "src", "set");

        // when
        fireEvent.error(images()[0]);
        fireEvent.error(images()[0]);
        fireEvent.error(images()[0]);

        // then
        expect(setSrc).toHaveBeenCalledTimes(1);
        expect(images()[0]).toHaveAttribute("data-fallback-tried", "1");
    });

    it("leaves a cover with nothing to fall back to alone", () => {
        // given
        const gallery = makeGallery({ cover_thumbnail_url: "/cover-thumb.png" });
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);
        const setSrc = vi.spyOn(HTMLImageElement.prototype, "src", "set");

        // when
        fireEvent.error(images()[0]);

        // then
        expect(setSrc).not.toHaveBeenCalled();
        expect(images()[0]).toHaveAttribute("src", "/cover-thumb.png");
    });

    it("shows a lone preview image on its own", () => {
        // given
        const gallery = makeGallery({ preview_images: [{ thumbnail_url: "", full_url: "/one-full.png" }] });

        // when
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // then
        expect(images()).toHaveLength(1);
        expect(images()[0]).toHaveAttribute("src", "/one-full.png");
    });

    it("shows a pair of preview images side by side", () => {
        // given
        const gallery = makeGallery({
            preview_images: [
                { thumbnail_url: "/one-thumb.png", full_url: "/one-full.png" },
                { thumbnail_url: "/two-thumb.png", full_url: "/two-full.png" },
            ],
        });

        // when
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // then
        expect(images().map(img => img.getAttribute("src"))).toEqual(["/one-thumb.png", "/two-thumb.png"]);
    });

    it("shows three preview images as a mosaic and ignores any beyond the third", () => {
        // given
        const gallery = makeGallery({
            preview_images: [
                { thumbnail_url: "/one-thumb.png", full_url: "/one-full.png" },
                { thumbnail_url: "/two-thumb.png", full_url: "/two-full.png" },
                { thumbnail_url: "/three-thumb.png", full_url: "/three-full.png" },
                { thumbnail_url: "/four-thumb.png", full_url: "/four-full.png" },
            ],
        });

        // when
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // then
        expect(images()).toHaveLength(3);
        expect(images()[0]).toHaveAttribute("src", "/one-thumb.png");
    });

    it("falls back to the full picture when a preview thumbnail will not load", () => {
        // given
        const gallery = makeGallery({
            preview_images: [
                { thumbnail_url: "/one-thumb.png", full_url: "/one-full.png" },
                { thumbnail_url: "/two-thumb.png", full_url: "/two-full.png" },
            ],
        });
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // when
        fireEvent.error(images()[1]);

        // then
        expect(images()[1]).toHaveAttribute("src", "/two-full.png");
        expect(images()[0]).toHaveAttribute("src", "/one-thumb.png");
    });

    it("prefers the cover over the recent pieces when the gallery has both", () => {
        // given
        const gallery = makeGallery({
            cover_thumbnail_url: "/cover-thumb.png",
            cover_image_url: "/cover-full.png",
            preview_images: [{ thumbnail_url: "/one-thumb.png", full_url: "/one-full.png" }],
        });

        // when
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // then
        expect(images()).toHaveLength(1);
        expect(images()[0]).toHaveAttribute("src", "/cover-thumb.png");
    });

    it("calls a gallery with no cover and no pieces empty", () => {
        // given
        const gallery = makeGallery({ preview_images: [] });

        // when
        renderWithProviders(<GalleryCoverPreview gallery={gallery} />);

        // then
        expect(screen.getByText("Empty")).toBeInTheDocument();
        expect(screen.queryByRole("presentation")).not.toBeInTheDocument();
    });
});
