import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { PostComposer } from "./PostComposer";

const mocks = vi.hoisted(() => ({
    createPost: vi.fn(),
    uploadMedia: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../../api/mutations/post", () => ({
    useCreatePost: () => ({ mutateAsync: mocks.createPost }),
    useUploadPostMediaById: () => ({ mutateAsync: mocks.uploadMedia }),
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

beforeEach(() => {
    mocks.createPost.mockResolvedValue({ id: newPostId });
    mocks.uploadMedia.mockResolvedValue({ id: 1 });
});

describe("PostComposer", () => {
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

    it("uploads every attached file against the post it just created", async () => {
        // given
        const user = userEvent.setup();
        const file = makeImage();
        const { container } = renderWithProviders(<PostComposer />);
        fireEvent.change(fileInput(container), { target: { files: [file] } });

        // when
        await user.type(bodyField(), "look at this");
        await user.click(postButton());

        // then
        expect(mocks.uploadMedia).toHaveBeenCalledWith({ id: newPostId, file });
        expect(mocks.navigate).toHaveBeenCalledWith(`/game-board/${newPostId}`);
    });

    it("reports a failed upload instead of leaving for the new post", async () => {
        // given
        const user = userEvent.setup();
        mocks.uploadMedia.mockRejectedValue(new Error("the media room is sealed"));
        const { container } = renderWithProviders(<PostComposer />);
        fireEvent.change(fileInput(container), { target: { files: [makeImage()] } });

        // when
        await user.type(bodyField(), "look at this");
        await user.click(postButton());

        // then
        expect(await screen.findByText("the media room is sealed")).toBeInTheDocument();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("keeps only the attachments that failed to upload so they are not lost", async () => {
        // given
        const user = userEvent.setup();
        const good = makeImage("good.png");
        const bad = makeImage("bad.png");
        mocks.uploadMedia.mockImplementation(({ file }: { id: string; file: File }) =>
            file.name === "bad.png"
                ? Promise.reject(new Error("the media room is sealed"))
                : Promise.resolve({ id: 1 }),
        );
        const { container } = renderWithProviders(<PostComposer />);
        fireEvent.change(fileInput(container), { target: { files: [good, bad] } });

        // when
        await user.type(bodyField(), "look at these");
        await user.click(postButton());

        // then
        expect(await screen.findByText("the media room is sealed")).toBeInTheDocument();
        expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(1);
        expect(bodyField()).toHaveValue("");
    });

    it("lets a post go out with only an attachment and no words", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<PostComposer />);

        // when
        fireEvent.change(fileInput(container), { target: { files: [makeImage()] } });

        // then
        expect(postButton()).toBeEnabled();
        await user.click(postButton());
        expect(mocks.createPost).toHaveBeenCalledWith({ body: "", corner: "general", poll: undefined });
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
