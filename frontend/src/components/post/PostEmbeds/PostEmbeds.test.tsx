import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { PostEmbed } from "../../../types/api";
import { PostEmbeds } from "./PostEmbeds";

function makeEmbed(overrides: Partial<PostEmbed> = {}): PostEmbed {
    return {
        url: "https://witch.test/the-golden-truth",
        type: "link",
        ...overrides,
    };
}

describe("PostEmbeds", () => {
    it("renders nothing when the post carries no embeds", () => {
        // given
        const embeds: PostEmbed[] = [];

        // when
        const { container } = renderWithProviders(<PostEmbeds embeds={embeds} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("frames a youtube embed on the cookieless host", () => {
        // given
        const embeds = [makeEmbed({ type: "youtube", video_id: "beato1986", title: "Golden Land" })];

        // when
        renderWithProviders(<PostEmbeds embeds={embeds} />);

        // then
        const frame = screen.getByTitle("Golden Land");
        expect(frame).toHaveAttribute("src", "https://www.youtube-nocookie.com/embed/beato1986");
        expect(frame).toHaveAttribute("allowfullscreen");
    });

    it("gives a youtube embed a generic frame title when it has none of its own", () => {
        // given
        const embeds = [makeEmbed({ type: "youtube", video_id: "beato1986" })];

        // when
        renderWithProviders(<PostEmbeds embeds={embeds} />);

        // then
        expect(screen.getByTitle("YouTube video")).toBeInTheDocument();
    });

    it("shows the site name, title and description of a link embed", () => {
        // given
        const embeds = [
            makeEmbed({
                site_name: "Rokkenjima Gazette",
                title: "The Witch's Epitaph",
                description: "Nine candles, one truth.",
            }),
        ];

        // when
        renderWithProviders(<PostEmbeds embeds={embeds} />);

        // then
        expect(screen.getByText("Rokkenjima Gazette")).toBeInTheDocument();
        expect(screen.getByText("The Witch's Epitaph")).toBeInTheDocument();
        expect(screen.getByText("Nine candles, one truth.")).toBeInTheDocument();
    });

    it("opens a link embed in a new tab without leaking the referrer", () => {
        // given
        const embeds = [makeEmbed({ title: "The Witch's Epitaph" })];

        // when
        renderWithProviders(<PostEmbeds embeds={embeds} />);

        // then
        const link = screen.getByRole("link");
        expect(link).toHaveAttribute("href", "https://witch.test/the-golden-truth");
        expect(link).toHaveAttribute("target", "_blank");
        expect(link).toHaveAttribute("rel", "noopener noreferrer");
    });

    it("renders no card for a link embed that has nothing worth showing", () => {
        // given
        const embeds = [makeEmbed({ site_name: "Rokkenjima Gazette" })];

        // when
        renderWithProviders(<PostEmbeds embeds={embeds} />);

        // then
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
        expect(screen.queryByText("Rokkenjima Gazette")).not.toBeInTheDocument();
    });

    it("still renders a card when only a preview image is available", () => {
        // given
        const embeds = [makeEmbed({ image: "https://witch.test/butterfly.png" })];

        // when
        const { container } = renderWithProviders(<PostEmbeds embeds={embeds} />);

        // then
        expect(screen.getByRole("link")).toBeInTheDocument();
        const image = container.querySelector("img");
        expect(image).toHaveAttribute("src", "https://witch.test/butterfly.png");
        expect(image).toHaveAttribute("loading", "lazy");
    });

    it("renders every embed it is given, whatever their kinds", () => {
        // given
        const embeds = [
            makeEmbed({ type: "youtube", video_id: "beato1986", title: "Golden Land" }),
            makeEmbed({ url: "https://witch.test/second", title: "Second Twilight" }),
        ];

        // when
        renderWithProviders(<PostEmbeds embeds={embeds} />);

        // then
        expect(screen.getByTitle("Golden Land")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Second Twilight" })).toHaveAttribute(
            "href",
            "https://witch.test/second",
        );
    });
});
