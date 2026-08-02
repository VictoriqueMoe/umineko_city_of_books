import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { Art } from "../../../types/api";
import { ArtGrid } from "./ArtGrid";

function makeArt(id: string, title: string): Art {
    return {
        id,
        author: { id: "user-1", username: "beatrice", display_name: "Beatrice" },
        corner: "general",
        art_type: "drawing",
        title,
        description: "",
        image_url: `https://cdn.example.test/${id}.png`,
        thumbnail_url: "",
        tags: [],
        like_count: 0,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        is_spoiler: false,
        created_at: "2026-01-01T00:00:00Z",
    };
}

describe("ArtGrid", () => {
    it("renders one card for every piece it is given", () => {
        // given
        const art = [makeArt("a", "Golden Butterflies"), makeArt("b", "Rokkenjima"), makeArt("c", "Sakutarou")];

        // when
        renderWithProviders(<ArtGrid art={art} />);

        // then
        expect(screen.getAllByRole("link")).toHaveLength(3);
        expect(screen.getByText("Sakutarou")).toBeInTheDocument();
    });

    it("keeps the order it was given", () => {
        // given
        const art = [makeArt("first", "One"), makeArt("second", "Two")];

        // when
        renderWithProviders(<ArtGrid art={art} />);

        // then
        const hrefs = screen.getAllByRole("link").map(link => link.getAttribute("href"));
        expect(hrefs).toEqual(["/gallery/art/first", "/gallery/art/second"]);
    });

    it("renders an empty grid when there is no art", () => {
        // given
        const art: Art[] = [];

        // when
        renderWithProviders(<ArtGrid art={art} />);

        // then
        expect(screen.queryAllByRole("link")).toHaveLength(0);
    });
});
