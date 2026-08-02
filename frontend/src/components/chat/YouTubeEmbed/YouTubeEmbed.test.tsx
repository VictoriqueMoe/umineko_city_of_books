import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { YouTubeEmbed } from "./YouTubeEmbed";

describe("YouTubeEmbed", () => {
    it("renders nothing when there are no videos", () => {
        // given
        const videoIds: string[] = [];

        // when
        const { container } = renderWithProviders(<YouTubeEmbed videoIds={videoIds} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("embeds the video through the no cookie host", () => {
        // given
        const videoIds = ["dQw4w9WgXcQ"];

        // when
        renderWithProviders(<YouTubeEmbed videoIds={videoIds} />);

        // then
        expect(screen.getByTitle("YouTube video")).toHaveAttribute(
            "src",
            "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
        );
    });

    it("renders one frame per video and keeps the given order", () => {
        // given
        const videoIds = ["aaaaaaaaaaa", "bbbbbbbbbbb", "ccccccccccc"];

        // when
        renderWithProviders(<YouTubeEmbed videoIds={videoIds} />);

        // then
        const frames = screen.getAllByTitle("YouTube video");
        expect(frames).toHaveLength(3);
        expect(frames.map(frame => frame.getAttribute("src"))).toEqual(
            videoIds.map(id => `https://www.youtube-nocookie.com/embed/${id}`),
        );
    });

    it("lets the player go fullscreen and loads it lazily", () => {
        // given
        const videoIds = ["dQw4w9WgXcQ"];

        // when
        renderWithProviders(<YouTubeEmbed videoIds={videoIds} />);

        // then
        const frame = screen.getByTitle("YouTube video");
        expect(frame).toHaveAttribute("allowfullscreen");
        expect(frame).toHaveAttribute("loading", "lazy");
    });
});
