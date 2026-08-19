import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { LinkPreview } from "../../types/api";
import { LinkPreviews } from "./LinkPreviews";

const { previews } = vi.hoisted(() => ({ previews: { byURL: new Map<string, LinkPreview>() } }));

vi.mock("../../api/queries/linkPreview", () => ({
    useLinkPreview: (url: string) => ({ preview: previews.byURL.get(url), loading: false }),
}));

function givenPreview(url: string, preview: Partial<LinkPreview>) {
    previews.byURL.set(url, { url, type: "link", ...preview } as LinkPreview);
}

describe("LinkPreviews", () => {
    beforeEach(() => {
        previews.byURL.clear();
    });

    it("renders nothing when the body has no links", () => {
        const { container } = renderWithProviders(<LinkPreviews body="no links at all" />);

        expect(container).toBeEmptyDOMElement();
    });

    it("renders a link card with the title, description and site name", () => {
        givenPreview("https://example.com/a", {
            title: "Rokkenjima",
            description: "an island",
            site_name: "Example",
            image: "https://example.com/a.png",
        });

        renderWithProviders(<LinkPreviews body="look at https://example.com/a" />);

        expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        expect(screen.getByText("an island")).toBeInTheDocument();
        expect(screen.getByText("Example")).toBeInTheDocument();
        expect(screen.getByRole("link")).toHaveAttribute("href", "https://example.com/a");
    });

    it("renders nothing for a link with no title, description or image", () => {
        givenPreview("https://example.com/a", {});

        const { container } = renderWithProviders(<LinkPreviews body="https://example.com/a" />);

        expect(container.querySelector("div")).toBeEmptyDOMElement();
    });

    it("renders nothing for a url the server could not preview", () => {
        givenPreview("https://example.com/a", { type: "" });

        const { container } = renderWithProviders(<LinkPreviews body="https://example.com/a" />);

        expect(container.querySelector("div")).toBeEmptyDOMElement();
    });

    it("renders a direct image link as the image itself", () => {
        givenPreview("https://media.tenor.com/x.gif", { type: "image" });

        renderWithProviders(<LinkPreviews body="https://media.tenor.com/x.gif" />);

        expect(screen.getByRole("presentation")).toHaveAttribute("src", "https://media.tenor.com/x.gif");
    });

    it("detects youtube without waiting for the server", () => {
        renderWithProviders(<LinkPreviews body="https://youtu.be/dQw4w9WgXcQ" />);

        expect(screen.getByTitle("YouTube video")).toHaveAttribute(
            "src",
            "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
        );
    });

    it("leaves waifuvault links to the inline renderer", () => {
        givenPreview("https://waifuvault.moe/f/1/cat.png", { type: "image" });

        const { container } = renderWithProviders(<LinkPreviews body="https://waifuvault.moe/f/1/cat.png" />);

        expect(container).toBeEmptyDOMElement();
    });

    it("renders one preview per distinct url", () => {
        givenPreview("https://a.example", { title: "First" });
        givenPreview("https://b.example", { title: "Second" });

        renderWithProviders(<LinkPreviews body="https://a.example and https://b.example and https://a.example" />);

        expect(screen.getByText("First")).toBeInTheDocument();
        expect(screen.getByText("Second")).toBeInTheDocument();
        expect(screen.getAllByRole("link")).toHaveLength(2);
    });

    describe("new account suppression", () => {
        function hoursAgo(hours: number): string {
            return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
        }

        it("renders nothing for an author still inside the restriction window", () => {
            givenPreview("https://example.com/a", { title: "Rokkenjima" });

            const { container } = renderWithProviders(
                <LinkPreviews body="https://example.com/a" authorCreatedAt={hoursAgo(2)} />,
                { siteInfo: { new_account_hours: 24 } },
            );

            expect(container).toBeEmptyDOMElement();
        });

        it("renders normally once the author is past the window", () => {
            givenPreview("https://example.com/a", { title: "Rokkenjima" });

            renderWithProviders(<LinkPreviews body="https://example.com/a" authorCreatedAt={hoursAgo(30)} />, {
                siteInfo: { new_account_hours: 24 },
            });

            expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        });

        it("renders normally when the restriction is disabled", () => {
            givenPreview("https://example.com/a", { title: "Rokkenjima" });

            renderWithProviders(<LinkPreviews body="https://example.com/a" authorCreatedAt={hoursAgo(1)} />, {
                siteInfo: { new_account_hours: 0 },
            });

            expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        });

        it("renders normally when the author has no known signup date", () => {
            givenPreview("https://example.com/a", { title: "Rokkenjima" });

            renderWithProviders(<LinkPreviews body="https://example.com/a" />, {
                siteInfo: { new_account_hours: 24 },
            });

            expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
        });

        it("suppresses a youtube embed too, not just link cards", () => {
            const { container } = renderWithProviders(
                <LinkPreviews body="https://youtu.be/dQw4w9WgXcQ" authorCreatedAt={hoursAgo(2)} />,
                { siteInfo: { new_account_hours: 24 } },
            );

            expect(container).toBeEmptyDOMElement();
        });
    });
});
