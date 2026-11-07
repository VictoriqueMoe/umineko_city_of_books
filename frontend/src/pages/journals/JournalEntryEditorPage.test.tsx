import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makePendingUpload, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { JournalDetail, JournalEntry, PostMedia, UserProfile } from "../../types/api";
import { JournalEntryEditorPage } from "./JournalEntryEditorPage";

const {
    useJournal,
    useJournalEntry,
    useCreateJournalEntry,
    useUpdateJournalEntry,
    useAttachJournalEntryMedia,
    useDeleteJournalEntryMedia,
    uploads,
    navigate,
} = vi.hoisted(() => ({
    useJournal: vi.fn(),
    useJournalEntry: vi.fn(),
    useCreateJournalEntry: vi.fn(),
    useUpdateJournalEntry: vi.fn(),
    useAttachJournalEntryMedia: vi.fn(),
    useDeleteJournalEntryMedia: vi.fn(),
    uploads: { begin: vi.fn(), remove: vi.fn(), statuses: vi.fn() },
    navigate: vi.fn(),
}));

vi.mock("../../hooks/queries/journal", () => ({ useJournal, useJournalEntry }));
vi.mock("../../hooks/mutations/journal", () => ({
    useCreateJournalEntry,
    useDeleteJournalEntryMedia,
    useUpdateJournalEntry,
    useAttachJournalEntryMedia,
}));
vi.mock("../../hooks/mutations/uploads", () => ({
    useBeginUpload: () => ({ mutateAsync: uploads.begin }),
    useRemoveUpload: () => ({ mutateAsync: uploads.remove }),
    useUploadStatuses: () => ({ mutateAsync: uploads.statuses }),
}));

beforeEach(() => {
    uploads.remove.mockResolvedValue(undefined);
    uploads.statuses.mockResolvedValue([]);
    uploads.begin.mockImplementation(({ file }: { file: File }) =>
        Promise.resolve(makePendingUpload({ id: `upload-${file.name}`, filename: file.name })),
    );
});

async function attach(user: ReturnType<typeof userEvent.setup>, container: HTMLElement, files: File[]): Promise<void> {
    await user.upload(container.querySelector<HTMLInputElement>('input[type="file"]')!, files);
    await waitFor(() => expect(screen.queryByRole("progressbar")).not.toBeInTheDocument());
}
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

interface GifPickerStubProps {
    onPick: (gif: { url: string }) => void;
    onClose: () => void;
}

vi.mock("../../components/chat/GifPicker/GifPicker", () => ({
    GifPicker: (props: GifPickerStubProps) => (
        <div aria-label="gif picker">
            <button onClick={() => props.onPick({ url: "https://media1.giphy.com/media/abc123/beato.gif" })}>
                pick a gif
            </button>
            <button onClick={props.onClose}>close the gif picker</button>
        </div>
    ),
}));

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });

function makeMedia(overrides: Partial<PostMedia> = {}): PostMedia {
    return {
        id: 1,
        media_url: "https://cdn.test/beato.png",
        media_type: "image",
        ...overrides,
    } as PostMedia;
}

function makeEntry(overrides: Partial<JournalEntry> = {}): JournalEntry {
    return {
        id: "entry-9",
        journal_id: "journal-1",
        entry_number: 2,
        title: "The Golden Truth",
        body: "The witch smiled.",
        word_count: 3,
        is_draft: false,
        has_prev: true,
        has_next: false,
        created_at: "2026-02-01T10:00:00Z",
        media: [],
        ...overrides,
    };
}

function makeJournal(overrides: Partial<JournalDetail> = {}): JournalDetail {
    return {
        id: "journal-1",
        title: "Rokkenjima Notes",
        work: "umineko",
        author,
        follower_count: 3,
        is_following: false,
        is_archived: false,
        is_paused: false,
        comment_count: 0,
        entry_count: 3,
        latest_entry_excerpt: "",
        created_at: "2026-01-01T00:00:00Z",
        last_author_activity_at: "2026-02-01T11:00:00Z",
        entries: [],
        latest_entry: null,
        comments: [],
        ...overrides,
    };
}

interface StubOptions {
    journal?: JournalDetail | null;
    entry?: JournalEntry | null;
    journalLoading?: boolean;
    entryLoading?: boolean;
    create?: () => Promise<{ id: string; entry_number: number }>;
    update?: () => Promise<unknown>;
    attach?: () => Promise<unknown>;
    remove?: () => Promise<unknown>;
}

function stubEditor(options: StubOptions = {}) {
    useJournal.mockReturnValue({
        journal: options.journal === undefined ? makeJournal() : options.journal,
        loading: options.journalLoading ?? false,
        refresh: vi.fn(),
    });
    useJournalEntry.mockReturnValue({
        entry: options.entry === undefined ? makeEntry() : options.entry,
        comments: [],
        loading: options.entryLoading ?? false,
        refresh: vi.fn(),
    });

    const createAsync = vi.fn(options.create ?? (() => Promise.resolve({ id: "entry-new", entry_number: 4 })));
    const updateAsync = vi.fn(options.update ?? (() => Promise.resolve({})));
    const attachAsync = vi.fn(options.attach ?? (() => Promise.resolve({})));
    const deleteMediaAsync = vi.fn(options.remove ?? (() => Promise.resolve({})));
    useCreateJournalEntry.mockReturnValue({ mutateAsync: createAsync });
    useUpdateJournalEntry.mockReturnValue({ mutateAsync: updateAsync });
    useAttachJournalEntryMedia.mockReturnValue({ mutateAsync: attachAsync });
    useDeleteJournalEntryMedia.mockReturnValue({ mutateAsync: deleteMediaAsync });

    return { createAsync, updateAsync, attachAsync, deleteMediaAsync };
}

function renderNew(user: UserProfile) {
    return renderWithProviders(<JournalEntryEditorPage />, {
        user,
        route: "/journals/journal-1/entry/new",
        path: "/journals/:id/entry/:number",
    });
}

function renderEdit(user: UserProfile) {
    return renderWithProviders(<JournalEntryEditorPage />, {
        user,
        route: "/journals/journal-1/entry/2/edit",
        path: "/journals/:id/entry/:number/edit",
    });
}

describe("JournalEntryEditorPage", () => {
    it("waits while the journal is loading", () => {
        // given
        stubEditor({ journalLoading: true });

        // when
        renderNew(author);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("says so when there is no such journal", () => {
        // given
        stubEditor({ journal: null });

        // when
        renderNew(author);

        // then
        expect(screen.getByText("Journal not found.")).toBeInTheDocument();
    });

    it("says so when the entry being edited does not exist", () => {
        // given
        stubEditor({ entry: null });

        // when
        renderEdit(author);

        // then
        expect(screen.getByText("Entry not found.")).toBeInTheDocument();
    });

    it("does not fetch an entry when starting a new one", () => {
        // given
        stubEditor();

        // when
        renderNew(author);

        // then
        expect(useJournalEntry).toHaveBeenCalledWith("journal-1", 0);
    });

    it("starts a new entry with an empty form", () => {
        // given
        stubEditor();

        // when
        renderNew(author);

        // then
        expect(screen.getByPlaceholderText("Leave empty to use 'Entry N'")).toHaveValue("");
        expect(screen.getByRole("button", { name: "Publish entry" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Save as draft" })).toBeDisabled();
    });

    it("publishes a new entry and opens it", async () => {
        // given
        const { createAsync } = stubEditor({ create: () => Promise.resolve({ id: "entry-new", entry_number: 4 }) });
        const user = userEvent.setup();
        renderNew(author);

        // when
        await user.type(screen.getByPlaceholderText("Leave empty to use 'Entry N'"), "  Arrival  ");
        await user.type(screen.getByPlaceholderText(/Write your entry\./), "  Beato laughed.  ");
        await user.click(screen.getByRole("button", { name: "Publish entry" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({
            payload: { title: "Arrival", body: "Beato laughed.", is_draft: false },
            media: [],
        });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-1/entry/4");
        });
    });

    it("saves a new entry as a draft and returns to the journal", async () => {
        // given
        const { createAsync } = stubEditor();
        const user = userEvent.setup();
        renderNew(author);

        // when
        await user.type(screen.getByPlaceholderText(/Write your entry\./), "Beato laughed.");
        await user.click(screen.getByRole("button", { name: "Save as draft" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({
            payload: { title: "", body: "Beato laughed.", is_draft: true },
            media: [],
        });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-1");
        });
    });

    it("reports why a new entry could not be saved", async () => {
        // given
        stubEditor({ create: () => Promise.reject(new Error("The witch forbids it")) });
        const user = userEvent.setup();
        renderNew(author);

        // when
        await user.type(screen.getByPlaceholderText(/Write your entry\./), "Beato laughed.");
        await user.click(screen.getByRole("button", { name: "Publish entry" }));

        // then
        expect(await screen.findByText("The witch forbids it")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("appends a picked gif to the body", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew(author);

        // when
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "pick a gif" }));

        // then
        expect(screen.getByPlaceholderText(/Write your entry\./)).toHaveValue(
            "https://media1.giphy.com/media/abc123/beato.gif",
        );
        expect(screen.queryByRole("button", { name: "pick a gif" })).not.toBeInTheDocument();
    });

    it("keeps what was already written when a gif is picked", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew(author);

        // when
        await user.type(screen.getByPlaceholderText(/Write your entry\./), "Beato laughed.");
        await user.click(screen.getByRole("button", { name: "+ GIF" }));
        await user.click(screen.getByRole("button", { name: "pick a gif" }));

        // then
        expect(screen.getByPlaceholderText(/Write your entry\./)).toHaveValue(
            "Beato laughed.\nhttps://media1.giphy.com/media/abc123/beato.gif",
        );
    });

    it("seeds the editor with the entry as it stands", () => {
        // given
        stubEditor({ entry: makeEntry({ title: "The Golden Truth", body: "The witch smiled." }) });

        // when
        renderEdit(author);

        // then
        expect(screen.getByRole("heading", { name: "Edit entry" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Leave empty to use 'Entry N'")).toHaveValue("The Golden Truth");
        expect(screen.getByPlaceholderText(/Write your entry\./)).toHaveValue("The witch smiled.");
    });

    it("offers no draft option once an entry has been published", () => {
        // given
        stubEditor({ entry: makeEntry({ is_draft: false }) });

        // when
        renderEdit(author);

        // then
        expect(screen.queryByRole("button", { name: "Save as draft" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Save entry" })).toBeInTheDocument();
    });

    it("explains that publishing a draft is what notifies followers", () => {
        // given
        stubEditor({ entry: makeEntry({ is_draft: true }) });

        // when
        renderEdit(author);

        // then
        expect(screen.getByText(/Save keeps it private; Publish notifies your followers\./)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Publish entry" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Save as draft" })).toBeInTheDocument();
    });

    it("saves an edited entry and reopens it", async () => {
        // given
        const { updateAsync } = stubEditor({ entry: makeEntry({ id: "entry-9", entry_number: 2 }) });
        const user = userEvent.setup();
        renderEdit(author);

        // when
        await user.click(screen.getByRole("button", { name: "Save entry" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith({
            id: "entry-9",
            payload: { title: "The Golden Truth", body: "The witch smiled.", is_draft: false },
        });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-1/entry/2");
        });
    });

    it("keeps an edited draft private and returns to the journal", async () => {
        // given
        const { updateAsync } = stubEditor({ entry: makeEntry({ is_draft: true }) });
        const user = userEvent.setup();
        renderEdit(author);

        // when
        await user.click(screen.getByRole("button", { name: "Save as draft" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith({
            id: "entry-9",
            payload: { title: "The Golden Truth", body: "The witch smiled.", is_draft: true },
        });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-1");
        });
    });

    it("marks an existing attachment for removal without deleting it yet", async () => {
        // given
        const { deleteMediaAsync } = stubEditor({ entry: makeEntry({ media: [makeMedia({ id: 7 })] }) });
        const user = userEvent.setup();
        renderEdit(author);

        // when
        await user.click(screen.getByRole("button", { name: "Remove attachment" }));

        // then
        expect(screen.getByRole("button", { name: "Undo remove" })).toBeInTheDocument();
        expect(deleteMediaAsync).not.toHaveBeenCalled();
    });

    it("changes its mind about removing an attachment", async () => {
        // given
        stubEditor({ entry: makeEntry({ media: [makeMedia({ id: 7 })] }) });
        const user = userEvent.setup();
        renderEdit(author);

        // when
        await user.click(screen.getByRole("button", { name: "Remove attachment" }));
        await user.click(screen.getByRole("button", { name: "Undo remove" }));

        // then
        expect(screen.getByRole("button", { name: "Remove attachment" })).toBeInTheDocument();
    });

    it("removes the marked attachments when the entry is saved", async () => {
        // given
        const { deleteMediaAsync } = stubEditor({ entry: makeEntry({ media: [makeMedia({ id: 7 })] }) });
        const user = userEvent.setup();
        renderEdit(author);

        // when
        await user.click(screen.getByRole("button", { name: "Remove attachment" }));
        await user.click(screen.getByRole("button", { name: "Save entry" }));

        // then
        await waitFor(() => {
            expect(deleteMediaAsync).toHaveBeenCalledWith({ entryId: "entry-9", mediaId: 7 });
        });
    });

    it("sends the finished uploads with the new entry in a single request", async () => {
        // given
        const { createAsync, attachAsync } = stubEditor({
            create: () => Promise.resolve({ id: "entry-new", entry_number: 4 }),
        });
        const user = userEvent.setup();
        const { container } = renderNew(author);

        // when
        await attach(user, container, [new File(["beato"], "beato.png", { type: "image/png" })]);
        await user.click(screen.getByRole("button", { name: "Mark as spoiler" }));
        await user.click(screen.getByRole("button", { name: "Publish entry" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-1/entry/4");
        });
        expect(createAsync).toHaveBeenCalledWith({
            payload: { title: "", body: "", is_draft: false },
            media: [{ upload_id: "upload-beato.png", is_spoiler: true }],
        });
        expect(attachAsync).not.toHaveBeenCalled();
    });

    it("holds the entry until every attachment has finished uploading", async () => {
        // given an upload the server has not answered yet
        stubEditor();
        uploads.begin.mockReturnValue(new Promise(() => {}));
        const user = userEvent.setup();
        const { container } = renderNew(author);

        // when
        await user.type(screen.getByPlaceholderText(/Write your entry\./), "Beato laughed.");
        await user.upload(container.querySelector<HTMLInputElement>('input[type="file"]')!, [
            new File(["beato"], "beato.png", { type: "image/png" }),
        ]);

        // then
        expect(screen.getByRole("progressbar", { name: "Upload progress" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Publish entry" })).toBeDisabled();
    });

    it("keeps the whole form and says so when the new entry and its attachments are refused", async () => {
        // given
        stubEditor({ create: () => Promise.reject(new Error("The upload was refused")) });
        const user = userEvent.setup();
        const { container } = renderNew(author);
        await user.type(screen.getByPlaceholderText("Leave empty to use 'Entry N'"), "Arrival");
        await user.type(screen.getByPlaceholderText(/Write your entry\./), "Beato laughed.");
        await attach(user, container, [new File(["beato"], "beato.png", { type: "image/png" })]);
        await user.click(screen.getByRole("button", { name: "Mark as spoiler" }));

        // when
        await user.click(screen.getByRole("button", { name: "Publish entry" }));

        // then
        expect(await screen.findByText("The upload was refused")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
        expect(screen.getByPlaceholderText("Leave empty to use 'Entry N'")).toHaveValue("Arrival");
        expect(screen.getByPlaceholderText(/Write your entry\./)).toHaveValue("Beato laughed.");
        expect(screen.getByRole("button", { name: "Mark as spoiler" })).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByRole("button", { name: "Publish entry" })).toBeEnabled();
    });

    it("attaches each finished upload to an existing entry when it is edited", async () => {
        // given
        const { attachAsync } = stubEditor({ entry: makeEntry({ id: "entry-9", entry_number: 2 }) });
        const user = userEvent.setup();
        const { container } = renderEdit(author);

        // when
        await attach(user, container, [new File(["beato"], "beato.png", { type: "image/png" })]);
        await user.click(screen.getByRole("button", { name: "Save entry" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-1/entry/2");
        });
        expect(attachAsync).toHaveBeenCalledExactlyOnceWith({
            entryId: "entry-9",
            media: { upload_id: "upload-beato.png", is_spoiler: false },
        });
    });

    it("retries only the attachment that failed when an edited entry is saved again", async () => {
        // given the second of two attachments is refused the first time
        const attachAsync = vi
            .fn()
            .mockResolvedValueOnce({})
            .mockRejectedValueOnce(new Error("the witch keeps that one"))
            .mockResolvedValueOnce({});
        stubEditor({ entry: makeEntry({ id: "entry-9", entry_number: 2 }) });
        useAttachJournalEntryMedia.mockReturnValue({ mutateAsync: attachAsync });
        const user = userEvent.setup();
        const { container } = renderEdit(author);
        await attach(user, container, [
            new File(["one"], "one.png", { type: "image/png" }),
            new File(["two"], "two.png", { type: "image/png" }),
        ]);
        await user.click(screen.getByRole("button", { name: "Save entry" }));
        await screen.findByText("the witch keeps that one");

        // when
        await user.click(screen.getByRole("button", { name: "Save entry" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-1/entry/2");
        });
        expect(attachAsync.mock.calls.map(call => call[0].media.upload_id)).toEqual([
            "upload-one.png",
            "upload-two.png",
            "upload-two.png",
        ]);
    });

    it("stays on the editor and says so when a marked attachment could not be removed", async () => {
        // given
        stubEditor({
            entry: makeEntry({ media: [makeMedia({ id: 7 })] }),
            remove: () => Promise.reject(new Error("The witch keeps her picture")),
        });
        const user = userEvent.setup();
        renderEdit(author);

        // when
        await user.click(screen.getByRole("button", { name: "Remove attachment" }));
        await user.click(screen.getByRole("button", { name: "Save entry" }));

        // then
        expect(await screen.findByText("The witch keeps her picture")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("lets an entry made only of attachments be published", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        const { container } = renderNew(author);

        // when
        await attach(user, container, [new File(["beato"], "beato.png", { type: "image/png" })]);

        // then
        expect(screen.getByRole("button", { name: "Publish entry" })).toBeEnabled();
    });

    it("returns to the journal from the back link", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew(author);

        // when
        await user.click(screen.getByText(/← Back to/));

        // then
        expect(navigate).toHaveBeenCalledWith("/journals/journal-1");
    });
});
