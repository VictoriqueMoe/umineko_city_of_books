import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { CreateCommentHandler } from "../../../hooks/useCommentHandlers";
import { CommentComposer } from "./CommentComposer";

const { createComment, GIF_URL } = vi.hoisted(() => ({
    createComment: vi.fn(),
    GIF_URL: "https://media.giphy.com/media/abc123/beato.gif",
}));

vi.mock("../../../hooks/mutations/post", () => ({
    useCreateComment: () => ({ mutateAsync: createComment }),
}));

vi.mock("../../chat/GifPicker/GifPicker", () => ({
    GifPicker: ({ onPick, onClose }: { onPick: (gif: { url: string }) => void; onClose: () => void }) => (
        <div>
            <button type="button" onClick={() => onPick({ url: GIF_URL })}>
                pick a gif
            </button>
            <button type="button" onClick={onClose}>
                close the gif picker
            </button>
        </div>
    ),
}));

interface SetupOptions {
    parentId?: string;
    onCreated?: () => void;
    createCommentFn?: CreateCommentHandler;
    maxImageSize?: number;
}

function setup(options: SetupOptions = {}) {
    return renderWithProviders(
        <CommentComposer
            postId="post-1"
            parentId={options.parentId}
            onCreated={options.onCreated ?? (() => {})}
            createCommentFn={options.createCommentFn}
        />,
        { siteInfo: options.maxImageSize === undefined ? undefined : { max_image_size: options.maxImageSize } },
    );
}

function makeImage(name: string, bytes = 4): File {
    return new File(["x".repeat(bytes)], name, { type: "image/png" });
}

function fileInput(container: HTMLElement): HTMLInputElement {
    const input = container.querySelector('input[type="file"]');
    if (!input) {
        throw new Error("the composer has no file input");
    }
    return input as HTMLInputElement;
}

describe("CommentComposer", () => {
    it("presents itself as a comment box at the top level", () => {
        // given
        const parentId = undefined;

        // when
        setup({ parentId });

        // then
        expect(screen.getByPlaceholderText("Write a comment...")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Comment" })).toBeInTheDocument();
    });

    it("presents itself as a reply box when it hangs off a parent comment", () => {
        // given
        const parentId = "comment-1";

        // when
        setup({ parentId });

        // then
        expect(screen.getByPlaceholderText("Write a reply...")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Reply" })).toBeInTheDocument();
    });

    it("keeps submission locked until there is something to send", async () => {
        // given
        const user = userEvent.setup();
        setup();
        expect(screen.getByRole("button", { name: "Comment" })).toBeDisabled();

        // when
        await user.type(screen.getByPlaceholderText("Write a comment..."), "Beato did it");

        // then
        expect(screen.getByRole("button", { name: "Comment" })).toBeEnabled();
    });

    it("treats a body of only whitespace as empty", async () => {
        // given
        const user = userEvent.setup();
        setup();

        // when
        await user.type(screen.getByPlaceholderText("Write a comment..."), "   ");

        // then
        expect(screen.getByRole("button", { name: "Comment" })).toBeDisabled();
    });

    it("sends the trimmed body and clears itself afterwards", async () => {
        // given
        const onCreated = vi.fn();
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const user = userEvent.setup();
        setup({ onCreated, createCommentFn });

        // when
        await user.type(screen.getByPlaceholderText("Write a comment..."), "  the golden truth  ");
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "the golden truth", undefined, []);
        await waitFor(() => expect(screen.getByPlaceholderText("Write a comment...")).toHaveValue(""));
        expect(onCreated).toHaveBeenCalledOnce();
    });

    it("passes the parent comment along when replying", async () => {
        // given
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const user = userEvent.setup();
        setup({ parentId: "comment-1", createCommentFn });

        // when
        await user.type(screen.getByPlaceholderText("Write a reply..."), "not without evidence");
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // then
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "not without evidence", "comment-1", []);
    });

    it("falls back to the create comment mutation when no handler is injected", async () => {
        // given
        createComment.mockResolvedValue({ id: "comment-9" });
        const user = userEvent.setup();
        setup({ parentId: "comment-1" });

        // when
        await user.type(screen.getByPlaceholderText("Write a reply..."), "a repeating tragedy");
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // then
        expect(createComment).toHaveBeenCalledWith({ body: "a repeating tragedy", parentId: "comment-1", media: [] });
    });

    it("hands the staged media and spoiler flags to the create comment mutation when no handler is injected", async () => {
        // given
        createComment.mockResolvedValue({ id: "comment-9" });
        const image = makeImage("one.png");
        const user = userEvent.setup();
        const { container } = setup();
        fireEvent.change(fileInput(container), { target: { files: [image] } });
        await user.click(screen.getByRole("button", { name: "Mark as spoiler" }));

        // when
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        expect(createComment).toHaveBeenCalledWith({
            body: "",
            parentId: undefined,
            media: [{ file: image, isSpoiler: true }],
        });
    });

    it("surfaces the reason a comment could not be posted", async () => {
        // given
        const onCreated = vi.fn();
        const createCommentFn = vi.fn(() => Promise.reject(new Error("the witch forbids it")));
        const user = userEvent.setup();
        setup({ onCreated, createCommentFn });

        // when
        await user.type(screen.getByPlaceholderText("Write a comment..."), "let me speak");
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        expect(await screen.findByText("the witch forbids it")).toBeInTheDocument();
        expect(onCreated).not.toHaveBeenCalled();
    });

    it("sends every attachment with the comment in a single request", async () => {
        // given
        const createCommentFn = vi.fn<CreateCommentHandler>(() => Promise.resolve({ id: "comment-9" }));
        const first = makeImage("one.png");
        const second = makeImage("two.png");
        const user = userEvent.setup();
        const { container } = setup({ createCommentFn });

        // when
        fireEvent.change(fileInput(container), { target: { files: [first, second] } });
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        await waitFor(() => expect(createCommentFn).toHaveBeenCalledOnce());
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "", undefined, [
            { file: first, isSpoiler: false },
            { file: second, isSpoiler: false },
        ]);
    });

    it("clears the text and every attachment once the comment and its media are stored", async () => {
        // given
        const onCreated = vi.fn();
        const createCommentFn = vi.fn<CreateCommentHandler>(() => Promise.resolve({ id: "comment-9" }));
        const user = userEvent.setup();
        const { container } = setup({ onCreated, createCommentFn });
        await user.type(screen.getByPlaceholderText("Write a comment..."), "look at this");
        fireEvent.change(fileInput(container), { target: { files: [makeImage("one.png")] } });

        // when
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        await waitFor(() => expect(onCreated).toHaveBeenCalledOnce());
        expect(screen.getByPlaceholderText("Write a comment...")).toHaveValue("");
        expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
    });

    it("keeps the text, every attachment and its spoiler flag, and shows the error when the comment is refused", async () => {
        // given
        const onCreated = vi.fn();
        const createCommentFn = vi.fn<CreateCommentHandler>(() => Promise.reject(new Error("the image was rejected")));
        const user = userEvent.setup();
        const { container } = setup({ onCreated, createCommentFn });
        await user.type(screen.getByPlaceholderText("Write a comment..."), "look at this");
        fireEvent.change(fileInput(container), {
            target: { files: [makeImage("good.png"), makeImage("bad.png")] },
        });
        await user.click(screen.getAllByRole("button", { name: "Mark as spoiler" })[1]);

        // when
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        expect(await screen.findByText("the image was rejected")).toBeInTheDocument();
        expect(onCreated).not.toHaveBeenCalled();
        expect(screen.getByPlaceholderText("Write a comment...")).toHaveValue("look at this");
        expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(2);
        const spoilerToggles = screen.getAllByRole("button", { name: "Mark as spoiler" });
        expect(spoilerToggles[0]).toHaveAttribute("aria-pressed", "false");
        expect(spoilerToggles[1]).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByRole("button", { name: "Comment" })).toBeEnabled();
    });

    it("sends the kept draft again unchanged when the refused comment is retried", async () => {
        // given
        const good = makeImage("good.png");
        const createCommentFn = vi
            .fn<CreateCommentHandler>()
            .mockRejectedValueOnce(new Error("storage is unavailable"))
            .mockResolvedValueOnce({ id: "comment-9" });
        const user = userEvent.setup();
        const { container } = setup({ createCommentFn });
        await user.type(screen.getByPlaceholderText("Write a comment..."), "second try");
        fireEvent.change(fileInput(container), { target: { files: [good] } });
        await user.click(screen.getByRole("button", { name: "Comment" }));
        await screen.findByText("storage is unavailable");

        // when
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        await waitFor(() => expect(createCommentFn).toHaveBeenCalledTimes(2));
        expect(createCommentFn).toHaveBeenLastCalledWith("post-1", "second try", undefined, [
            { file: good, isSpoiler: false },
        ]);
    });

    it("refuses an attachment that is over the site image limit", async () => {
        // given
        const { container } = setup({ maxImageSize: 8 });

        // when
        fireEvent.change(fileInput(container), { target: { files: [makeImage("huge.png", 64)] } });

        // then
        expect(await screen.findByText(/huge\.png is too large/)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Comment" })).toBeDisabled();
    });

    it("keeps the pasted images that fit and complains about the ones that do not", async () => {
        // given
        setup({ maxImageSize: 8 });

        // when
        fireEvent.paste(screen.getByPlaceholderText("Write a comment..."), {
            clipboardData: { files: [makeImage("small.png", 4), makeImage("huge.png", 64)] },
        });

        // then
        expect(await screen.findByText(/huge\.png is too large/)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Comment" })).toBeEnabled();
    });

    it("lets an attachment be taken back off again", async () => {
        // given
        const user = userEvent.setup();
        const { container } = setup();
        fireEvent.change(fileInput(container), { target: { files: [makeImage("one.png")] } });
        expect(screen.getByRole("button", { name: "Comment" })).toBeEnabled();

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Comment" })).toBeDisabled();
    });

    it("sends reordered attachments in their new order", async () => {
        // given
        const createCommentFn = vi.fn<CreateCommentHandler>(() => Promise.resolve({ id: "comment-9" }));
        const first = makeImage("one.png");
        const second = makeImage("two.png");
        const user = userEvent.setup();
        const { container } = setup({ createCommentFn });
        fireEvent.change(fileInput(container), { target: { files: [first, second] } });

        // when
        await user.click(screen.getAllByRole("button", { name: "Move later" })[0]);
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        await waitFor(() => expect(createCommentFn).toHaveBeenCalledOnce());
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "", undefined, [
            { file: second, isSpoiler: false },
            { file: first, isSpoiler: false },
        ]);
    });

    it("posts a picked GIF as a comment of its own and closes the picker", async () => {
        // given
        const onCreated = vi.fn();
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const user = userEvent.setup();
        setup({ onCreated, createCommentFn });

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "pick a gif" }));

        // then
        expect(createCommentFn).toHaveBeenCalledWith("post-1", GIF_URL, undefined);
        await waitFor(() => expect(screen.queryByRole("button", { name: "pick a gif" })).not.toBeInTheDocument());
        expect(onCreated).toHaveBeenCalledOnce();
    });

    it("sends nothing when the GIF picker is dismissed", async () => {
        // given
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const user = userEvent.setup();
        setup({ createCommentFn });

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "close the gif picker" }));

        // then
        expect(screen.queryByRole("button", { name: "pick a gif" })).not.toBeInTheDocument();
        expect(createCommentFn).not.toHaveBeenCalled();
    });

    it("explains why a GIF could not be sent", async () => {
        // given
        const createCommentFn = vi.fn(() => Promise.reject(new Error("giphy is unreachable")));
        const user = userEvent.setup();
        setup({ createCommentFn });

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "pick a gif" }));

        // then
        expect(await screen.findByText("giphy is unreachable")).toBeInTheDocument();
    });

    it("drops an unsent draft when the composer is reused for another post", async () => {
        // given an unsent draft on one post
        const user = userEvent.setup();
        const { rerender } = setup();
        await user.type(screen.getByPlaceholderText("Write a comment..."), "Beato did it");

        // when the same instance is handed a different post, as it is when the route id changes
        rerender(<CommentComposer postId="post-2" onCreated={() => {}} />);

        // then the draft does not follow the reader to the other post
        expect(screen.getByPlaceholderText("Write a comment...")).toHaveValue("");
    });
});
