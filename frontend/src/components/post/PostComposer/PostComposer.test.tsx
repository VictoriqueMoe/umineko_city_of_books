import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makePendingUpload } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { emitRealtimeEvent } from "../../../test-utils/ws";
import type { PendingUpload } from "../../../types/api";
import { PostComposer } from "./PostComposer";

const mocks = vi.hoisted(() => ({
    createPost: vi.fn(),
    beginUpload: vi.fn(),
    removeUpload: vi.fn(),
    uploadStatuses: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../../hooks/mutations/post", () => ({
    useCreatePost: () => ({ mutateAsync: mocks.createPost }),
}));

vi.mock("../../../hooks/mutations/uploads", () => ({
    useBeginUpload: () => ({ mutateAsync: mocks.beginUpload }),
    useRemoveUpload: () => ({ mutateAsync: mocks.removeUpload }),
    useUploadStatuses: () => ({ mutateAsync: mocks.uploadStatuses }),
}));

vi.mock("../../chat/GifPicker/GifPicker", () => ({
    GifPicker: ({ onPick, onClose }: { onPick: (gif: { url: string }) => void; onClose: () => void }) => (
        <div>
            <button onClick={() => onPick({ url: "https://media.giphy.com/media/abc123/beato.gif" })}>Pick GIF</button>
            <button onClick={onClose}>Close GIFs</button>
        </div>
    ),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

const newPostId = "22222222-2222-2222-2222-222222222222";

function bodyField(): HTMLElement {
    return screen.getByPlaceholderText("What's on your mind?");
}

function postButton(): HTMLElement {
    return screen.getByRole("button", { name: "Post" });
}

function makeImage(name = "beato.png"): File {
    return new File(["x"], name, { type: "image/png" });
}

function fileInput(container: HTMLElement): HTMLInputElement {
    const input = container.querySelector('input[type="file"]');
    return input as HTMLInputElement;
}

function uploadRef(name: string, isSpoiler = false) {
    return { upload_id: `upload-${name}`, is_spoiler: isSpoiler };
}

async function attach(container: HTMLElement, files: File[]): Promise<void> {
    fireEvent.change(fileInput(container), { target: { files } });
    await waitFor(() => expect(screen.queryByRole("progressbar")).not.toBeInTheDocument());
}

beforeEach(() => {
    mocks.createPost.mockResolvedValue({ id: newPostId });
    mocks.removeUpload.mockResolvedValue(undefined);
    mocks.uploadStatuses.mockResolvedValue([]);
    mocks.beginUpload.mockImplementation(({ file }: { file: File }) =>
        Promise.resolve(makePendingUpload({ id: `upload-${file.name}`, filename: file.name })),
    );
});

describe("PostComposer", () => {
    it("uploads an attachment as soon as it is added and shows its own progress, then processing", async () => {
        // given an upload that reports progress and then waits on the server to encode it
        let report: (sent: number, total: number) => void = () => {};
        let finish: (upload: PendingUpload) => void = () => {};
        mocks.beginUpload.mockImplementation(
            (input: { onProgress: (sent: number, total: number) => void }) =>
                new Promise<PendingUpload>(resolve => {
                    report = input.onProgress;
                    finish = resolve;
                }),
        );
        const { container } = renderWithProviders(<PostComposer />);

        // when
        fireEvent.change(fileInput(container), { target: { files: [makeImage()] } });
        const beforeProgress = screen.getByRole("progressbar", { name: "Upload progress" });
        const startText = beforeProgress.getAttribute("aria-valuetext");
        act(() => {
            report(42, 100);
        });
        const partWay = screen.getByRole("progressbar", { name: "Upload progress" }).getAttribute("aria-valuetext");
        await act(async () => {
            finish(makePendingUpload({ id: "upload-beato.png", status: "processing" }));
        });

        // then
        expect(mocks.beginUpload).toHaveBeenCalledOnce();
        expect(mocks.createPost).not.toHaveBeenCalled();
        expect(startText).toBe("Uploading 0%");
        expect(partWay).toBe("Uploading 42%");
        expect(screen.getByRole("progressbar", { name: "Upload progress" })).toHaveAttribute(
            "aria-valuetext",
            "Processing...",
        );
        expect(postButton()).toBeDisabled();
    });

    it("lets the post go out once the server reports the attachment processed", async () => {
        // given an attachment the server is still encoding
        const user = userEvent.setup();
        mocks.beginUpload.mockResolvedValue(makePendingUpload({ id: "upload-beato.png", status: "processing" }));
        const { container } = renderWithProviders(<PostComposer />);
        fireEvent.change(fileInput(container), { target: { files: [makeImage()] } });
        await waitFor(() =>
            expect(screen.getByRole("progressbar", { name: "Upload progress" })).toHaveAttribute(
                "aria-valuetext",
                "Processing...",
            ),
        );

        // when
        emitRealtimeEvent({ type: "upload_ready", data: makePendingUpload({ id: "upload-beato.png" }) });
        await user.click(postButton());

        // then
        expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
        expect(mocks.createPost).toHaveBeenCalledWith(expect.objectContaining({ media: [uploadRef("beato.png")] }));
    });

    it("marks an attachment the server could not process and keeps the post held", async () => {
        // given
        mocks.beginUpload.mockRejectedValue(new Error("the file is cursed"));
        const { container } = renderWithProviders(<PostComposer />);

        // when
        fireEvent.change(fileInput(container), { target: { files: [makeImage()] } });

        // then
        expect(await screen.findByRole("alert")).toHaveTextContent("Failed");
        expect(screen.getByRole("alert").parentElement).toHaveAttribute("title", "the file is cursed");
        expect(postButton()).toBeDisabled();
    });

    it("shows no upload bar for a post without files", async () => {
        // given a text post whose request is still in flight
        const user = userEvent.setup();
        mocks.createPost.mockReturnValue(new Promise(() => {}));
        renderWithProviders(<PostComposer />);
        await user.type(bodyField(), "the golden truth");

        // when
        await user.click(postButton());

        // then
        expect(screen.getByRole("button", { name: "Posting..." })).toBeDisabled();
        expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    });

    it("keeps the post button disabled until there is something to post", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);
        expect(postButton()).toBeDisabled();

        // when
        await user.type(bodyField(), "the golden truth");

        // then
        expect(postButton()).toBeEnabled();
    });

    it("refuses to post whitespace on its own", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "   ");

        // then
        expect(postButton()).toBeDisabled();
        expect(mocks.createPost).not.toHaveBeenCalled();
    });

    it("posts the trimmed body to the corner it was mounted in", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer corner="umineko" />);

        // when
        await user.type(bodyField(), "  without love it cannot be seen  ");
        await user.click(postButton());

        // then
        expect(mocks.createPost).toHaveBeenCalledWith({
            body: "without love it cannot be seen",
            corner: "umineko",
            poll: undefined,
            media: [],
        });
    });

    it("posts to the general corner when no corner is given", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "hello Rokkenjima");
        await user.click(postButton());

        // then
        expect(mocks.createPost).toHaveBeenCalledWith({
            body: "hello Rokkenjima",
            corner: "general",
            poll: undefined,
            media: [],
        });
    });

    it("lands on the new post and empties the composer once it is created", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "hello Rokkenjima");
        await user.click(postButton());

        // then
        expect(mocks.navigate).toHaveBeenCalledWith(`/game-board/${newPostId}`);
        expect(bodyField()).toHaveValue("");
    });

    it("refuses a poll that has fewer than two filled options", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "who is the culprit?");
        await user.click(screen.getByRole("button", { name: "+ Poll" }));
        await user.type(screen.getByPlaceholderText("Option 1"), "Beatrice");
        await user.click(postButton());

        // then
        expect(screen.getByText("Poll needs at least 2 non-empty options")).toBeInTheDocument();
        expect(mocks.createPost).not.toHaveBeenCalled();
    });

    it("attaches the poll options and the chosen duration to the post", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "who is the culprit?");
        await user.click(screen.getByRole("button", { name: "+ Poll" }));
        await user.type(screen.getByPlaceholderText("Option 1"), "Beatrice");
        await user.type(screen.getByPlaceholderText("Option 2"), "Battler");
        await user.selectOptions(screen.getByRole("combobox"), "3600");
        await user.click(postButton());

        // then
        expect(mocks.createPost).toHaveBeenCalledWith({
            body: "who is the culprit?",
            corner: "general",
            poll: {
                options: [{ label: "Beatrice" }, { label: "Battler" }],
                duration_seconds: 3600,
            },
            media: [],
        });
    });

    it("trims poll labels and drops the options left blank", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "who is the culprit?");
        await user.click(screen.getByRole("button", { name: "+ Poll" }));
        await user.click(screen.getByRole("button", { name: "+ Add Option" }));
        await user.type(screen.getByPlaceholderText("Option 1"), "  Beatrice  ");
        await user.type(screen.getByPlaceholderText("Option 2"), "Battler");
        await user.click(postButton());

        // then
        expect(mocks.createPost).toHaveBeenCalledWith(
            expect.objectContaining({
                poll: { options: [{ label: "Beatrice" }, { label: "Battler" }], duration_seconds: 86400 },
            }),
        );
    });

    it("puts the poll away again when it is removed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);
        await user.click(screen.getByRole("button", { name: "+ Poll" }));

        // when
        await user.click(screen.getByRole("button", { name: "Remove Poll" }));

        // then
        expect(screen.queryByPlaceholderText("Option 1")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "+ Poll" })).toBeInTheDocument();
    });

    it("shows why the post could not be created", async () => {
        // given
        const user = userEvent.setup();
        mocks.createPost.mockRejectedValue(new Error("the witch forbids it"));
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "hello Rokkenjima");
        await user.click(postButton());

        // then
        expect(await screen.findByText("the witch forbids it")).toBeInTheDocument();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("shows a generic failure when the rejection is not an error", async () => {
        // given
        const user = userEvent.setup();
        mocks.createPost.mockRejectedValue("no reason given");
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "hello Rokkenjima");
        await user.click(postButton());

        // then
        expect(await screen.findByText("Failed to create post")).toBeInTheDocument();
    });

    it("sends every finished upload with the post it creates", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<PostComposer />);
        await attach(container, [makeImage()]);

        // when
        await user.type(bodyField(), "look at this");
        await user.click(postButton());

        // then
        expect(mocks.createPost).toHaveBeenCalledWith(expect.objectContaining({ media: [uploadRef("beato.png")] }));
        expect(mocks.navigate).toHaveBeenCalledWith(`/game-board/${newPostId}`);
    });

    it("sends an attachment the author marked as a spoiler with its flag set", async () => {
        // given an attached image the author has marked
        const user = userEvent.setup();
        const { container } = renderWithProviders(<PostComposer />);
        await attach(container, [makeImage()]);

        // when
        await user.click(screen.getByRole("button", { name: "Mark as spoiler" }));
        await user.type(bodyField(), "look at this");
        await user.click(postButton());

        // then
        expect(mocks.createPost).toHaveBeenCalledWith(
            expect.objectContaining({ media: [uploadRef("beato.png", true)] }),
        );
    });

    it("keeps the spoiler flag with its own attachment when they are reordered", async () => {
        // given two attachments where only the second is marked
        const user = userEvent.setup();
        const { container } = renderWithProviders(<PostComposer />);
        await attach(container, [makeImage("first.png"), makeImage("second.png")]);
        await user.click(screen.getAllByRole("button", { name: "Mark as spoiler" })[1]);

        // when the marked one is moved to the front
        await user.click(screen.getAllByRole("button", { name: "Move earlier" })[1]);
        await user.type(bodyField(), "look at this");
        await user.click(postButton());

        // then the flag travelled with its own upload
        expect(mocks.createPost).toHaveBeenCalledWith(
            expect.objectContaining({ media: [uploadRef("second.png", true), uploadRef("first.png")] }),
        );
    });

    it("keeps the whole draft and shows the error instead of leaving when the post and its media are refused", async () => {
        // given
        const user = userEvent.setup();
        mocks.createPost.mockRejectedValue(new Error("the media room is sealed"));
        const { container } = renderWithProviders(<PostComposer />);
        await attach(container, [makeImage("good.png"), makeImage("bad.png")]);
        await user.click(screen.getAllByRole("button", { name: "Mark as spoiler" })[1]);
        await user.click(screen.getByRole("button", { name: "+ Poll" }));
        await user.type(screen.getByPlaceholderText("Option 1"), "Beatrice");
        await user.type(screen.getByPlaceholderText("Option 2"), "Battler");

        // when
        await user.type(bodyField(), "look at these");
        await user.click(postButton());

        // then
        expect(await screen.findByText("the media room is sealed")).toBeInTheDocument();
        expect(mocks.navigate).not.toHaveBeenCalled();
        expect(bodyField()).toHaveValue("look at these");
        expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(2);
        const spoilerToggles = screen.getAllByRole("button", { name: "Mark as spoiler" });
        expect(spoilerToggles[0]).toHaveAttribute("aria-pressed", "false");
        expect(spoilerToggles[1]).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByPlaceholderText("Option 1")).toHaveValue("Beatrice");
        expect(screen.getByPlaceholderText("Option 2")).toHaveValue("Battler");
        expect(postButton()).toBeEnabled();
    });

    it("sends the kept draft again in one request when the refused post is retried", async () => {
        // given
        const user = userEvent.setup();
        mocks.createPost.mockRejectedValueOnce(new Error("storage is unavailable"));
        const { container } = renderWithProviders(<PostComposer />);
        await attach(container, [makeImage()]);
        await user.type(bodyField(), "look at this");
        await user.click(postButton());
        await screen.findByText("storage is unavailable");

        // when
        await user.click(postButton());

        // then
        expect(mocks.createPost).toHaveBeenCalledTimes(2);
        expect(mocks.createPost).toHaveBeenLastCalledWith({
            body: "look at this",
            corner: "general",
            poll: undefined,
            media: [uploadRef("beato.png")],
        });
        expect(mocks.navigate).toHaveBeenCalledWith(`/game-board/${newPostId}`);
    });

    it("lets a post go out with only an attachment and no words", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<PostComposer />);

        // when
        await attach(container, [makeImage()]);

        // then
        expect(postButton()).toBeEnabled();
        await user.click(postButton());
        expect(mocks.createPost).toHaveBeenCalledWith(expect.objectContaining({ body: "", corner: "general" }));
    });

    it("rejects an attachment that is over the site image limit", async () => {
        // given
        const { container } = renderWithProviders(<PostComposer />, { siteInfo: { max_image_size: 0 } });

        // when
        fireEvent.change(fileInput(container), { target: { files: [makeImage()] } });

        // then
        expect(await screen.findByText(/beato\.png is too large/)).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
    });

    it("keeps a pasted image that fits within the site limits", async () => {
        // given
        renderWithProviders(<PostComposer />);

        // when
        fireEvent.paste(bodyField(), { clipboardData: { files: [makeImage()] } });

        // then
        expect(await screen.findByRole("button", { name: "Remove" })).toBeInTheDocument();
    });

    it("rejects a pasted image that is over the site limit", async () => {
        // given
        renderWithProviders(<PostComposer />, { siteInfo: { max_image_size: 0 } });

        // when
        fireEvent.paste(bodyField(), { clipboardData: { files: [makeImage()] } });

        // then
        expect(await screen.findByText(/beato\.png is too large/)).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
    });

    it("posts a picked gif on its own straight away", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer corner="higurashi" />);

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "Pick GIF" }));

        // then
        expect(mocks.createPost).toHaveBeenCalledWith({
            body: "https://media.giphy.com/media/abc123/beato.gif",
            corner: "higurashi",
        });
        expect(mocks.navigate).toHaveBeenCalledWith(`/game-board/${newPostId}`);
    });

    it("shows why a picked gif could not be sent", async () => {
        // given
        const user = userEvent.setup();
        mocks.createPost.mockRejectedValue(new Error("giphy is asleep"));
        renderWithProviders(<PostComposer />);

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "Pick GIF" }));

        // then
        expect(await screen.findByText("giphy is asleep")).toBeInTheDocument();
    });

    it("closes the gif picker without posting when it is dismissed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<PostComposer />);
        await user.click(screen.getByRole("button", { name: "+ GIF" }));

        // when
        await user.click(screen.getByRole("button", { name: "Close GIFs" }));

        // then
        expect(screen.queryByRole("button", { name: "Pick GIF" })).not.toBeInTheDocument();
        expect(mocks.createPost).not.toHaveBeenCalled();
    });

    it("locks the composer while the post is in flight", async () => {
        // given
        const user = userEvent.setup();
        let release: (result: { id: string }) => void = () => {};
        mocks.createPost.mockImplementation(
            () =>
                new Promise<{ id: string }>(resolve => {
                    release = resolve;
                }),
        );
        renderWithProviders(<PostComposer />);

        // when
        await user.type(bodyField(), "hello Rokkenjima");
        await user.click(postButton());

        // then
        expect(screen.getByRole("button", { name: "Posting..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "+ GIF" })).toBeDisabled();
        await act(async () => {
            release({ id: newPostId });
        });
        expect(mocks.createPost).toHaveBeenCalledOnce();
    });
});
