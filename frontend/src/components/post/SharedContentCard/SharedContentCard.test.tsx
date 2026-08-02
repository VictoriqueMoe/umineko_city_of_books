import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { PostMedia, SharedContentPreview, User } from "../../../types/api";
import { SharedContentCard } from "./SharedContentCard";

const author: User = {
    id: "00000000-0000-0000-0000-0000000000aa",
    username: "beatrice",
    display_name: "Beatrice",
};

function makeContent(overrides: Partial<SharedContentPreview> = {}): SharedContentPreview {
    return {
        id: "11111111-1111-1111-1111-111111111111",
        content_type: "post",
        deleted: false,
        url: "/game-board/11111111-1111-1111-1111-111111111111",
        ...overrides,
    };
}

function makeMedia(id: number, overrides: Partial<PostMedia> = {}): PostMedia {
    return {
        id,
        media_url: `https://witch.test/full-${id}.png`,
        media_type: "image",
        sort_order: id,
        ...overrides,
    };
}

describe("SharedContentCard", () => {
    it("replaces deleted content with a notice and no link to follow", () => {
        // given
        const content = makeContent({ deleted: true, body: "the truth is gone" });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("This content is no longer available")).toBeInTheDocument();
        expect(screen.getByText("Post")).toBeInTheDocument();
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
        expect(screen.queryByText("the truth is gone")).not.toBeInTheDocument();
    });

    it("links the whole card to the shared content and names its kind", () => {
        // given
        const content = makeContent({ content_type: "fanfic", url: "/fanfic/7", title: "Rokkenjima Nights" });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByRole("link")).toHaveAttribute("href", "/fanfic/7");
        expect(screen.getByText("Fanfiction")).toBeInTheDocument();
    });

    it("falls back to the raw type for a kind it does not know", () => {
        // given
        const content = makeContent({ content_type: "journal" });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("journal")).toBeInTheDocument();
    });

    it("truncates a long shared post body at two hundred characters", () => {
        // given
        const content = makeContent({ body: "a".repeat(250) });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText(`${"a".repeat(200)}...`)).toBeInTheDocument();
    });

    it("leaves a short shared post body untouched", () => {
        // given
        const content = makeContent({ body: "Without love it cannot be seen" });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("Without love it cannot be seen")).toBeInTheDocument();
    });

    it("abbreviates like and comment counts above a thousand", () => {
        // given
        const content = makeContent({ like_count: 1500, comment_count: 12 });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("1.5k likes")).toBeInTheDocument();
        expect(screen.getByText("12 comments")).toBeInTheDocument();
    });

    it("hides the like and comment counts while they are still zero", () => {
        // given
        const content = makeContent({ like_count: 0, comment_count: 0 });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.queryByText(/likes/)).not.toBeInTheDocument();
        expect(screen.queryByText(/comments/)).not.toBeInTheDocument();
    });

    it("shows at most four media thumbnails from a shared post", () => {
        // given
        const content = makeContent({ media: [1, 2, 3, 4, 5, 6].map(id => makeMedia(id)) });

        // when
        const { container } = renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(container.querySelectorAll("img")).toHaveLength(4);
    });

    it("prefers the thumbnail over the full sized media url", () => {
        // given
        const content = makeContent({
            media: [makeMedia(1, { thumbnail_url: "https://witch.test/thumb-1.png" })],
        });

        // when
        const { container } = renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(container.querySelector("img")).toHaveAttribute("src", "https://witch.test/thumb-1.png");
    });

    it("plays a shared video attachment in a video element rather than an image", () => {
        // given
        const content = makeContent({
            media: [makeMedia(1, { media_type: "video", media_url: "https://witch.test/clip-1.mp4" })],
        });

        // when
        const { container } = renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(container.querySelector("video")).toHaveAttribute("src", "https://witch.test/clip-1.mp4");
        expect(container.querySelectorAll("img")).toHaveLength(0);
    });

    it("renders no media grid when the shared post has no media", () => {
        // given
        const content = makeContent({ media: [], body: "no pictures here" });

        // when
        const { container } = renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(container.querySelectorAll("img")).toHaveLength(0);
    });

    it("labels a shared piece of art with its title as the image alternative text", () => {
        // given
        const content = makeContent({
            content_type: "art",
            title: "Golden Butterflies",
            image_url: "https://witch.test/art.png",
        });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("Art")).toBeInTheDocument();
        expect(screen.getByAltText("Golden Butterflies")).toHaveAttribute("src", "https://witch.test/art.png");
    });

    it("shows the vote score badge of a shared ship", () => {
        // given
        const content = makeContent({ content_type: "ship", title: "Battler and Beatrice", vote_score: 42 });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("Score: 42")).toBeInTheDocument();
    });

    it("marks an unsolved mystery as open alongside its difficulty", () => {
        // given
        const content = makeContent({
            content_type: "mystery",
            title: "The Sealed Room",
            difficulty: "hard",
            solved: false,
        });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("hard")).toBeInTheDocument();
        expect(screen.getByText("Open")).toBeInTheDocument();
    });

    it("marks a solved mystery as solved", () => {
        // given
        const content = makeContent({ content_type: "mystery", title: "The Sealed Room", solved: true });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("Solved")).toBeInTheDocument();
        expect(screen.queryByText("Open")).not.toBeInTheDocument();
    });

    it("shows the series and credibility of a shared theory", () => {
        // given
        const content = makeContent({
            content_type: "theory",
            title: "Kanon is Yasu",
            series: "umineko",
            credibility_score: 88,
        });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("umineko")).toBeInTheDocument();
        expect(screen.getByText("Credibility: 88")).toBeInTheDocument();
    });

    it("abbreviates a large fanfiction word count", () => {
        // given
        const content = makeContent({
            content_type: "fanfic",
            title: "Rokkenjima Nights",
            series: "umineko",
            rating: "teen",
            word_count: 12500,
        });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("12.5k words")).toBeInTheDocument();
        expect(screen.getByText("teen")).toBeInTheDocument();
    });

    it("names the author without nesting a second link inside the card", () => {
        // given
        const content = makeContent({ author, body: "Without love it cannot be seen" });

        // when
        renderWithProviders(<SharedContentCard content={content} />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getAllByRole("link")).toHaveLength(1);
    });
});
