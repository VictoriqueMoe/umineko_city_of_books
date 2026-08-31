import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeGallery as makeContentGallery, makePublicUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { Gallery } from "../../../types/api";
import { GalleryCard } from "./GalleryCard";

function makeGallery(overrides: Partial<Gallery> = {}): Gallery {
    return makeContentGallery({ author: makePublicUser({ id: "beatrice-id" }), ...overrides });
}

function card(): HTMLElement {
    return screen.getByRole("link");
}

describe("GalleryCard", () => {
    it("links through to the gallery it stands for", () => {
        // given
        const gallery = makeGallery({ id: "gallery-9" });

        // when
        renderWithProviders(<GalleryCard gallery={gallery} />);

        // then
        expect(card()).toHaveAttribute("href", "/gallery/view/gallery-9");
    });

    it("names the gallery and counts what it holds", () => {
        // given
        const gallery = makeGallery({ name: "Witch Portraits", art_count: 4 });

        // when
        renderWithProviders(<GalleryCard gallery={gallery} />);

        // then
        expect(screen.getByText("Witch Portraits")).toBeInTheDocument();
        expect(screen.getByText("4 pieces")).toBeInTheDocument();
    });

    it("lets the name find its own direction", () => {
        // given
        const gallery = makeGallery({ name: "الفراشات" });

        // when
        renderWithProviders(<GalleryCard gallery={gallery} />);

        // then
        expect(screen.getByText("الفراشات")).toHaveAttribute("dir", "auto");
    });

    it("shows the cover inside the card", () => {
        // given
        const gallery = makeGallery({ cover_thumbnail_url: "/cover-thumb.png", cover_image_url: "/cover-full.png" });

        // when
        renderWithProviders(<GalleryCard gallery={gallery} />);

        // then
        expect(within(card()).getByRole("presentation")).toHaveAttribute("src", "/cover-thumb.png");
    });

    it("calls a gallery with nothing in it empty", () => {
        // given
        const gallery = makeGallery({ art_count: 0 });

        // when
        renderWithProviders(<GalleryCard gallery={gallery} />);

        // then
        expect(within(card()).getByText("Empty")).toBeInTheDocument();
    });
});
