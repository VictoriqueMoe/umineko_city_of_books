import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { GifEmbed } from "./GifEmbed";

const MEDIA_GIF = "https://media.giphy.com/media/abc123/giphy.gif";
const SHORT_GIF = "https://i.giphy.com/xyz789.gif";

interface RejectionWatcher {
    on(event: "unhandledRejection", listener: (reason: unknown) => void): void;
    off(event: "unhandledRejection", listener: (reason: unknown) => void): void;
}

const rejectionWatcher = (globalThis as unknown as { process: RejectionWatcher }).process;

describe("GifEmbed", () => {
    it("shows the gif with the alt text it was given", () => {
        // given
        const alt = "dancing golden butterfly";

        // when
        renderWithProviders(<GifEmbed src={MEDIA_GIF} alt={alt} />);

        // then
        const img = screen.getByAltText(alt);
        expect(img).toHaveAttribute("src", MEDIA_GIF);
        expect(img).toHaveAttribute("loading", "lazy");
    });

    it("falls back to a generic alt text", () => {
        // given
        const src = MEDIA_GIF;

        // when
        renderWithProviders(<GifEmbed src={src} />);

        // then
        expect(screen.getByAltText("GIF")).toBeInTheDocument();
    });

    it("hides the favourite control from signed out visitors", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<GifEmbed src={MEDIA_GIF} />, { user });

        // then
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("hides the favourite control when the url is not a giphy gif", () => {
        // given
        const src = "https://example.com/cats/beatrice.gif";

        // when
        renderWithProviders(<GifEmbed src={src} />, { user: makeUser() });

        // then
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("offers to add an unfavourited gif to favourites", () => {
        // given
        const isFavourite = vi.fn(() => false);

        // when
        renderWithProviders(<GifEmbed src={MEDIA_GIF} />, { user: makeUser(), gifFavourites: { isFavourite } });

        // then
        expect(screen.getByRole("button", { name: "Add to favourites" })).toHaveTextContent("☆");
        expect(isFavourite).toHaveBeenCalledWith("abc123");
    });

    it("offers to remove a gif that is already favourited", () => {
        // given
        const isFavourite = (giphyID: string) => giphyID === "abc123";

        // when
        renderWithProviders(<GifEmbed src={MEDIA_GIF} />, { user: makeUser(), gifFavourites: { isFavourite } });

        // then
        expect(screen.getByRole("button", { name: "Remove from favourites" })).toHaveTextContent("★");
    });

    it("sends the giphy id and the title when the star is pressed", async () => {
        // given
        const toggle = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderWithProviders(<GifEmbed src={MEDIA_GIF} alt="golden butterfly" />, {
            user: makeUser(),
            gifFavourites: { toggle },
        });

        // when
        await user.click(screen.getByRole("button", { name: "Add to favourites" }));

        // then
        expect(toggle).toHaveBeenCalledWith({
            giphy_id: "abc123",
            url: MEDIA_GIF,
            title: "golden butterfly",
            preview_url: MEDIA_GIF,
            width: 0,
            height: 0,
        });
    });

    it("stores an empty title when the alt text is only the fallback", async () => {
        // given
        const toggle = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderWithProviders(<GifEmbed src={MEDIA_GIF} />, { user: makeUser(), gifFavourites: { toggle } });

        // when
        await user.click(screen.getByRole("button", { name: "Add to favourites" }));

        // then
        expect(toggle).toHaveBeenCalledWith(expect.objectContaining({ title: "" }));
    });

    it("reads the id out of the short i.giphy.com form", async () => {
        // given
        const toggle = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderWithProviders(<GifEmbed src={SHORT_GIF} />, { user: makeUser(), gifFavourites: { toggle } });

        // when
        await user.click(screen.getByRole("button", { name: "Add to favourites" }));

        // then
        expect(toggle).toHaveBeenCalledWith(expect.objectContaining({ giphy_id: "xyz789" }));
    });

    it("swallows a failed favourite instead of leaving the rejection unhandled", async () => {
        // given
        const toggle = vi.fn(() => Promise.reject(new Error("giphy is unreachable")));
        const unhandled: unknown[] = [];
        const record = (reason: unknown) => unhandled.push(reason);
        rejectionWatcher.on("unhandledRejection", record);
        const user = userEvent.setup();
        renderWithProviders(<GifEmbed src={MEDIA_GIF} />, { user: makeUser(), gifFavourites: { toggle } });

        // when
        await user.click(screen.getByRole("button", { name: "Add to favourites" }));
        await new Promise(resolve => setTimeout(resolve, 0));
        rejectionWatcher.off("unhandledRejection", record);

        // then
        expect(toggle).toHaveBeenCalledOnce();
        expect(unhandled).toEqual([]);
    });

    it("keeps the star press away from the surrounding element", async () => {
        // given
        const onSurroundingClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <div onClick={onSurroundingClick}>
                <GifEmbed src={MEDIA_GIF} />
            </div>,
            { user: makeUser() },
        );

        // when
        await user.click(screen.getByRole("button", { name: "Add to favourites" }));

        // then
        expect(onSurroundingClick).not.toHaveBeenCalled();
    });

    it("calls onClick when the gif itself is clicked", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<GifEmbed src={MEDIA_GIF} onClick={onClick} />, { user: makeUser() });

        // when
        await user.click(screen.getByAltText("GIF"));

        // then
        expect(onClick).toHaveBeenCalledOnce();
    });
});
