import { describe, expect, it } from "vitest";
import type { Gallery, User } from "../../types/api";
import { groupByAuthor } from "./groupByAuthor";

const beatrice: User = { id: "beatrice-id", username: "beatrice", display_name: "Beatrice" };
const ronove: User = { id: "ronove-id", username: "ronove", display_name: "Ronove" };
const ange: User = { id: "ange-id", username: "ange", display_name: "Ange" };

function makeGallery(id: string, author: User, name = id): Gallery {
    return {
        id,
        author,
        name,
        description: "",
        cover_image_url: "",
        cover_thumbnail_url: "",
        art_count: 0,
        created_at: "2026-07-01T10:00:00Z",
    };
}

describe("groupByAuthor", () => {
    it("gathers every gallery an artist made under one entry", () => {
        // given
        const galleries = [makeGallery("g1", beatrice), makeGallery("g2", ronove), makeGallery("g3", beatrice)];

        // when
        const artists = groupByAuthor(galleries);

        // then
        expect(artists).toHaveLength(2);
        expect(artists[0].galleries.map(g => g.id)).toEqual(["g1", "g3"]);
    });

    it("keeps the artist behind the galleries", () => {
        // given
        const galleries = [makeGallery("g1", beatrice)];

        // when
        const artists = groupByAuthor(galleries);

        // then
        expect(artists[0].user).toEqual(beatrice);
    });

    it("orders the artists by their display name", () => {
        // given
        const galleries = [makeGallery("g1", ronove), makeGallery("g2", beatrice), makeGallery("g3", ange)];

        // when
        const artists = groupByAuthor(galleries);

        // then
        expect(artists.map(a => a.user.display_name)).toEqual(["Ange", "Beatrice", "Ronove"]);
    });

    it("keeps each artist's galleries in the order they arrived", () => {
        // given
        const galleries = [
            makeGallery("g1", beatrice, "Sketchbook"),
            makeGallery("g2", beatrice, "Golden Butterflies"),
        ];

        // when
        const artists = groupByAuthor(galleries);

        // then
        expect(artists[0].galleries.map(g => g.name)).toEqual(["Sketchbook", "Golden Butterflies"]);
    });

    it("tells two artists who share a display name apart by their id", () => {
        // given
        const otherBeatrice: User = { id: "beatrice-2", username: "beato", display_name: "Beatrice" };
        const galleries = [makeGallery("g1", beatrice), makeGallery("g2", otherBeatrice)];

        // when
        const artists = groupByAuthor(galleries);

        // then
        expect(artists).toHaveLength(2);
    });

    it("hands back nothing at all when there are no galleries", () => {
        // given
        const galleries: Gallery[] = [];

        // when
        const artists = groupByAuthor(galleries);

        // then
        expect(artists).toEqual([]);
    });
});
