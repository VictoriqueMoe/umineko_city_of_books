import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { CommentComposer } from "./CommentComposer";

const { createComment, uploadMedia, GIF_URL } = vi.hoisted(() => ({
    createComment: vi.fn(),
    uploadMedia: vi.fn(),
    GIF_URL: "https://media.giphy.com/media/abc123/beato.gif",
}));

vi.mock("../../../api/mutations/post", () => ({
    useCreateComment: () => ({ mutateAsync: createComment }),
    useUploadCommentMedia: () => ({ mutateAsync: uploadMedia }),
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
    createCommentFn?: (postId: string, body: string, parentId?: string) => Promise<{ id: string }>;
    uploadMediaFn?: (commentId: string, file: File) => Promise<unknown>;
    maxImageSize?: number;
}

function setup(options: SetupOptions = {}) {
    return renderWithProviders(
        <CommentComposer
            postId="post-1"
            parentId={options.parentId}
            onCreated={options.onCreated ?? (() => {})}
            createCommentFn={options.createCommentFn}
            uploadMediaFn={options.uploadMediaFn}
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
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "the golden truth", undefined);
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
        expect(createCommentFn).toHaveBeenCalledWith("post-1", "not without evidence", "comment-1");
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
        expect(createComment).toHaveBeenCalledWith({ body: "a repeating tragedy", parentId: "comment-1" });
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

    it("uploads every attachment against the comment it just created", async () => {
        // given
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const uploadMediaFn = vi.fn(() => Promise.resolve({}));
        const first = makeImage("one.png");
        const second = makeImage("two.png");
        const user = userEvent.setup();
        const { container } = setup({ createCommentFn, uploadMediaFn });

        // when
        fireEvent.change(fileInput(container), { target: { files: [first, second] } });
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        await waitFor(() => expect(uploadMediaFn).toHaveBeenCalledTimes(2));
        expect(uploadMediaFn).toHaveBeenNthCalledWith(1, "comment-9", first);
        expect(uploadMediaFn).toHaveBeenNthCalledWith(2, "comment-9", second);
    });

    it("reports a failed upload but still finishes posting the comment", async () => {
        // given
        const onCreated = vi.fn();
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const uploadMediaFn = vi.fn(() => Promise.reject(new Error("the image was rejected")));
        const user = userEvent.setup();
        const { container } = setup({ onCreated, createCommentFn, uploadMediaFn });

        // when
        fireEvent.change(fileInput(container), { target: { files: [makeImage("one.png")] } });
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        expect(await screen.findByText("the image was rejected")).toBeInTheDocument();
        expect(onCreated).toHaveBeenCalledOnce();
    });

    it("keeps only the attachments that failed to upload so they are not lost", async () => {
        // given
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const uploadMediaFn = vi.fn((_commentId: string, file: File) =>
            file.name === "bad.png" ? Promise.reject(new Error("the image was rejected")) : Promise.resolve({}),
        );
        const user = userEvent.setup();
        const { container } = setup({ createCommentFn, uploadMediaFn });

        // when
        fireEvent.change(fileInput(container), {
            target: { files: [makeImage("good.png"), makeImage("bad.png")] },
        });
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        expect(await screen.findByText("the image was rejected")).toBeInTheDocument();
        expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(1);
        expect(screen.getByPlaceholderText("Write a comment...")).toHaveValue("");
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

    it("uploads reordered attachments in their new order", async () => {
        // given
        const createCommentFn = vi.fn(() => Promise.resolve({ id: "comment-9" }));
        const uploadMediaFn = vi.fn(() => Promise.resolve({}));
        const first = makeImage("one.png");
        const second = makeImage("two.png");
        const user = userEvent.setup();
        const { container } = setup({ createCommentFn, uploadMediaFn });
        fireEvent.change(fileInput(container), { target: { files: [first, second] } });

        // when
        await user.click(screen.getAllByRole("button", { name: "Move later" })[0]);
        await user.click(screen.getByRole("button", { name: "Comment" }));

        // then
        await waitFor(() => expect(uploadMediaFn).toHaveBeenCalledTimes(2));
        expect(uploadMediaFn).toHaveBeenNthCalledWith(1, "comment-9", second);
        expect(uploadMediaFn).toHaveBeenNthCalledWith(2, "comment-9", first);
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
});
