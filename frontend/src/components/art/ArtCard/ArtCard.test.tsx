import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { Art } from "../../../types/api";
import { ArtCard } from "./ArtCard";

const fullUrl = "https://cdn.example.test/art/full.png";
const thumbUrl = "https://cdn.example.test/art/thumb.png";

function makeArt(overrides: Partial<Art> = {}): Art {
    return {
        id: "art-1",
        author: { id: "user-1", username: "beatrice", display_name: "Beatrice" },
        corner: "general",
        art_type: "drawing",
        title: "Golden Butterflies",
        description: "",
        image_url: fullUrl,
        thumbnail_url: thumbUrl,
        tags: [],
        like_count: 12,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        is_spoiler: false,
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

describe("ArtCard", () => {
    it("links to the detail page of the piece", () => {
        // given
        const art = makeArt({ id: "art-42" });

        // when
        renderWithProviders(<ArtCard art={art} />);

        // then
        expect(screen.getByRole("link")).toHaveAttribute("href", "/gallery/art/art-42");
    });

    it("shows the thumbnail when the piece has one", () => {
        // given
        const art = makeArt();

        // when
        renderWithProviders(<ArtCard art={art} />);

        // then
        expect(screen.getByRole("img", { name: "Golden Butterflies" })).toHaveAttribute("src", thumbUrl);
    });

    it("falls back to the full image when there is no thumbnail", () => {
        // given
        const art = makeArt({ thumbnail_url: "" });

        // when
        renderWithProviders(<ArtCard art={art} />);

        // then
        expect(screen.getByRole("img", { name: "Golden Butterflies" })).toHaveAttribute("src", fullUrl);
    });

    it("swaps to the full image when the thumbnail fails to load", () => {
        // given
        renderWithProviders(<ArtCard art={makeArt()} />);
        const image = screen.getByRole("img", { name: "Golden Butterflies" });

        // when
        fireEvent.error(image);

        // then
        expect(image).toHaveAttribute("src", fullUrl);
    });

    it("shows the title, the author and the like count", () => {
        // given
        const art = makeArt({ title: "Witch of Miracles", like_count: 7 });

        // when
        renderWithProviders(<ArtCard art={art} />);

        // then
        expect(screen.getByText("Witch of Miracles")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText(/7/)).toBeInTheDocument();
    });

    it("keeps the author name out of the link so the card stays a single target", () => {
        // given
        const art = makeArt();

        // when
        renderWithProviders(<ArtCard art={art} />);

        // then
        expect(screen.getAllByRole("link")).toHaveLength(1);
    });

    it("hides a spoiler piece behind a reveal prompt", () => {
        // given
        const art = makeArt({ is_spoiler: true });

        // when
        renderWithProviders(<ArtCard art={art} />);

        // then
        expect(screen.getByText("Spoiler")).toBeInTheDocument();
        expect(screen.getByText("Click to reveal")).toBeInTheDocument();
    });

    it("reveals a spoiler piece once it is clicked", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ArtCard art={makeArt({ is_spoiler: true })} />);

        // when
        await user.click(screen.getByRole("img", { name: "Golden Butterflies" }));

        // then
        expect(screen.queryByText("Spoiler")).not.toBeInTheDocument();
    });

    it("shows no reveal prompt for a piece that is not marked as a spoiler", () => {
        // given
        const art = makeArt({ is_spoiler: false });

        // when
        renderWithProviders(<ArtCard art={art} />);

        // then
        expect(screen.queryByText("Spoiler")).not.toBeInTheDocument();
    });
});
