import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { FanficChapter, FanficDetail, ShipCharacter, UserProfile } from "../../types/api";
import { FanficEditorPage } from "./FanficEditorPage";

const {
    useFanfic,
    useFanficSeries,
    useFanficLanguages,
    fetched,
    useCreateFanfic,
    useUpdateFanfic,
    useUploadFanficCover,
    useUploadFanficCoverFor,
    useDeleteFanficCover,
    useCreateFanficChapter,
    useUpdateFanficChapter,
    navigate,
} = vi.hoisted(() => ({
    useFanfic: vi.fn(),
    useFanficSeries: vi.fn(),
    useFanficLanguages: vi.fn(),
    fetched: { fanfic: null as unknown, chapter: null as unknown },
    useCreateFanfic: vi.fn(),
    useUpdateFanfic: vi.fn(),
    useUploadFanficCover: vi.fn(),
    useUploadFanficCoverFor: vi.fn(),
    useDeleteFanficCover: vi.fn(),
    useCreateFanficChapter: vi.fn(),
    useUpdateFanficChapter: vi.fn(),
    navigate: vi.fn(),
}));

const { can } = vi.hoisted(() => ({ can: vi.fn() }));

vi.mock("../../api/queries/fanfic", () => ({
    useFanfic,
    useFanficLanguages,
    useFanficSeries,
    fanficQueryFns: {
        fanfic: (id: string) => ({ queryKey: ["fanfic", id], queryFn: () => Promise.resolve(fetched.fanfic) }),
        chapter: (id: string, n: number) => ({
            queryKey: ["fanfic", id, "chapter", n],
            queryFn: () => Promise.resolve(fetched.chapter),
        }),
    },
}));
vi.mock("../../api/mutations/fanfic", () => ({
    useCreateFanfic,
    useCreateFanficChapter,
    useDeleteFanficCover,
    useUpdateFanfic,
    useUpdateFanficChapter,
    useUploadFanficCover,
    useUploadFanficCoverFor,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});
vi.mock("../../utils/permissions", async importOriginal => {
    const actual = await importOriginal<typeof import("../../utils/permissions")>();
    can.mockImplementation(actual.can);
    return { ...actual, can };
});

interface EditorStubProps {
    content: string;
    onChange: (html: string) => void;
    placeholder?: string;
}

vi.mock("../../components/RichTextEditor/RichTextEditor", () => ({
    RichTextEditor: (props: EditorStubProps) => (
        <textarea
            aria-label="story body"
            placeholder={props.placeholder}
            value={props.content}
            onChange={e => props.onChange(e.target.value)}
        />
    ),
}));

interface CharacterPickerStubProps {
    onAdd: (character: ShipCharacter) => void;
    existing: ShipCharacter[];
}

vi.mock("../../components/CharacterPicker/CharacterPicker", () => ({
    CharacterPicker: (props: CharacterPickerStubProps) => (
        <div>
            <button
                type="button"
                onClick={() =>
                    props.onAdd({ series: "umineko", character_id: "c1", character_name: "Kanon", sort_order: 0 })
                }
            >
                add Kanon
            </button>
            <span>{`${props.existing.length} chosen`}</span>
        </div>
    ),
}));

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const stranger = makeUser({ id: "stranger-1", username: "battler", display_name: "Battler" });
const moderator = makeUser({ id: "mod-1", username: "ronove", display_name: "Ronove", role: "moderator" });

const DRAFT_KEY = "fanfic-draft";

function makeFanfic(overrides: Partial<FanficDetail> = {}): FanficDetail {
    return {
        id: "fanfic-1",
        author,
        title: "Golden Land",
        summary: "A closed room on Rokkenjima.",
        series: "Umineko",
        rating: "T",
        language: "English",
        status: "in_progress",
        is_oneshot: false,
        contains_lemons: false,
        genres: ["Mystery"],
        tags: ["closed room"],
        characters: [],
        is_pairing: false,
        word_count: 2500,
        chapter_count: 2,
        favourite_count: 4,
        view_count: 90,
        comment_count: 0,
        user_favourited: false,
        published_at: "2026-01-01T00:00:00Z",
        created_at: "2026-01-01T00:00:00Z",
        chapters: [],
        comments: [],
        reading_progress: 0,
        viewer_blocked: false,
        ...overrides,
    };
}

function makeChapter(overrides: Partial<FanficChapter> = {}): FanficChapter {
    return {
        id: "chapter-1",
        chapter_number: 1,
        title: "",
        body: "<p>Beatrice laughed.</p>",
        word_count: 3,
        has_prev: false,
        has_next: false,
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    fanfic?: FanficDetail | null;
    loading?: boolean;
    series?: string[];
    languages?: string[];
    create?: () => Promise<{ id: string }>;
    update?: () => Promise<unknown>;
    createChapter?: () => Promise<unknown>;
    updateChapter?: () => Promise<unknown>;
}

function stubEditor(options: StubOptions = {}) {
    useFanfic.mockReturnValue({
        fanfic: options.fanfic ?? null,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
    useFanficSeries.mockReturnValue({ series: options.series ?? ["Umineko", "Higurashi", "Rose Guns Days"] });
    useFanficLanguages.mockReturnValue({ languages: options.languages ?? ["English", "Japanese"] });

    const createAsync = vi.fn(options.create ?? (() => Promise.resolve({ id: "fanfic-new" })));
    const updateAsync = vi.fn(options.update ?? (() => Promise.resolve({})));
    const uploadCoverAsync = vi.fn(() => Promise.resolve({}));
    const uploadCoverForAsync = vi.fn(() => Promise.resolve({}));
    const deleteCoverAsync = vi.fn(() => Promise.resolve({}));
    const createChapterAsync = vi.fn(options.createChapter ?? (() => Promise.resolve({})));
    const updateChapterAsync = vi.fn(options.updateChapter ?? (() => Promise.resolve({})));
    useCreateFanfic.mockReturnValue({ mutateAsync: createAsync });
    useUpdateFanfic.mockReturnValue({ mutateAsync: updateAsync });
    useUploadFanficCover.mockReturnValue({ mutateAsync: uploadCoverAsync });
    useUploadFanficCoverFor.mockReturnValue({ mutateAsync: uploadCoverForAsync });
    useDeleteFanficCover.mockReturnValue({ mutateAsync: deleteCoverAsync });
    useCreateFanficChapter.mockReturnValue({ mutateAsync: createChapterAsync });
    useUpdateFanficChapter.mockReturnValue({ mutateAsync: updateChapterAsync });

    return {
        createAsync,
        updateAsync,
        uploadCoverAsync,
        uploadCoverForAsync,
        deleteCoverAsync,
        createChapterAsync,
        updateChapterAsync,
    };
}

function renderNew(user: UserProfile = author) {
    return renderWithProviders(<FanficEditorPage />, { user, route: "/fanfiction/new" });
}

function renderEdit(user: UserProfile = author) {
    return renderWithProviders(<FanficEditorPage />, {
        user,
        route: "/fanfiction/fanfic-1/edit",
        path: "/fanfiction/:id/edit",
    });
}

describe("FanficEditorPage", () => {
    it("starts a brand new fanfic on the details step", () => {
        // given
        stubEditor();

        // when
        renderNew();

        // then
        expect(screen.getByRole("heading", { name: "New Fanfic" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Your fanfic title...")).toHaveValue("");
    });

    it("offers the pinned series alongside the ones the archive already holds", () => {
        // given
        stubEditor({ series: ["Umineko", "Rose Guns Days"] });

        // when
        renderNew();

        // then
        const seriesOptions = screen.getAllByRole("option").map(o => o.textContent);
        expect(seriesOptions).toContain("Umineko");
        expect(seriesOptions).toContain("Higurashi");
        expect(seriesOptions).toContain("Ciconia");
        expect(seriesOptions).toContain("Rose Guns Days");
    });

    it("refuses to move on without a title", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.click(screen.getByRole("button", { name: /^Next:/ }));

        // then
        expect(screen.getByText("Title is required")).toBeInTheDocument();
        expect(screen.queryByLabelText("story body")).not.toBeInTheDocument();
    });

    it("refuses to move on with an empty custom series", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.selectOptions(screen.getAllByRole("combobox")[0], "__other__");
        await user.click(screen.getByRole("button", { name: /^Next:/ }));

        // then
        expect(screen.getByText("Series is required")).toBeInTheDocument();
    });

    it("refuses to move on with an empty custom language", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.selectOptions(screen.getAllByRole("combobox")[2], "__other__");
        await user.click(screen.getByRole("button", { name: /^Next:/ }));

        // then
        expect(screen.getByText("Language is required")).toBeInTheDocument();
    });

    it("calls the second step writing the story for a one-shot", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));

        // then
        expect(screen.getByRole("heading", { name: "Write Your Story" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Write your story here...")).toBeInTheDocument();
    });

    it("calls the second step writing the first chapter for a serial", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.click(screen.getByRole("switch", { name: "One-shot" }));
        await user.click(screen.getByRole("button", { name: "Next: Write Story" }));

        // then
        expect(screen.getByRole("heading", { name: "Write First Chapter" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Write your first chapter here...")).toBeInTheDocument();
    });

    it("publishes a new fanfic and opens it", async () => {
        // given
        const { createAsync } = stubEditor({ create: () => Promise.resolve({ id: "fanfic-new" }) });
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "  Golden Land  ");
        await user.type(screen.getByPlaceholderText("Brief summary of your story..."), "A closed room.");
        await user.selectOptions(screen.getAllByRole("combobox")[1], "M");
        await user.selectOptions(screen.getAllByRole("combobox")[3], "Mystery");
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.type(screen.getByLabelText("story body"), "Beatrice laughed.");
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({
            title: "Golden Land",
            summary: "A closed room.",
            series: "Umineko",
            rating: "M",
            language: "English",
            status: "in_progress",
            is_oneshot: true,
            contains_lemons: false,
            genres: ["Mystery"],
            tags: [],
            characters: [],
            is_pairing: false,
            body: "Beatrice laughed.",
        });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-new");
        });
    });

    it("saves a new fanfic as a draft", async () => {
        // given
        const { createAsync } = stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.click(screen.getByRole("button", { name: "Save as Draft" }));

        // then
        expect(createAsync).toHaveBeenCalledWith(expect.objectContaining({ status: "draft" }));
    });

    it("keeps only the genres that were actually chosen and never twice", async () => {
        // given
        const { createAsync } = stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.selectOptions(screen.getAllByRole("combobox")[3], "Mystery");
        await user.selectOptions(screen.getAllByRole("combobox")[4], "Mystery");
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(createAsync).toHaveBeenCalledWith(expect.objectContaining({ genres: ["Mystery"] }));
    });

    it("sends a typed custom series and language instead of the pinned ones", async () => {
        // given
        const { createAsync } = stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.selectOptions(screen.getAllByRole("combobox")[0], "__other__");
        await user.type(screen.getByPlaceholderText("Enter series name..."), "  Higanbana  ");
        await user.selectOptions(screen.getAllByRole("combobox")[2], "__other__");
        await user.type(screen.getByPlaceholderText("Enter language..."), "  Welsh  ");
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(createAsync).toHaveBeenCalledWith(expect.objectContaining({ series: "Higanbana", language: "Welsh" }));
    });

    it("reports why the fanfic could not be created", async () => {
        // given
        stubEditor({ create: () => Promise.reject(new Error("The witch forbids it")) });
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(await screen.findByText("The witch forbids it")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("adds a tag when the writer presses enter", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Type a tag and press Enter..."), "closed room{Enter}");

        // then
        expect(screen.getByText(/closed room/)).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Type a tag and press Enter...")).toHaveValue("");
    });

    it("refuses to add the same tag twice whatever the casing", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Type a tag and press Enter..."), "closed room{Enter}");
        await user.type(screen.getByPlaceholderText("Type a tag and press Enter..."), "Closed Room{Enter}");

        // then
        expect(screen.getAllByRole("button", { name: "Remove tag" })).toHaveLength(1);
    });

    it("drops a tag the writer changed their mind about", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();
        await user.type(screen.getByPlaceholderText("Type a tag and press Enter..."), "closed room{Enter}");

        // when
        await user.click(screen.getByRole("button", { name: "Remove tag" }));

        // then
        expect(screen.queryByRole("button", { name: "Remove tag" })).not.toBeInTheDocument();
    });

    it("stops the writer at ten tags", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();
        const field = screen.getByPlaceholderText("Type a tag and press Enter...");

        // when
        for (let i = 0; i < 12; i++) {
            await user.type(field, `tag${i}{Enter}`);
        }

        // then
        expect(screen.getAllByRole("button", { name: "Remove tag" })).toHaveLength(10);
    });

    it("adds and drops a character", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.click(screen.getByRole("button", { name: "add Kanon" }));

        // then
        expect(screen.getByText("1 chosen")).toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: "Remove character" }));
        expect(screen.getByText("0 chosen")).toBeInTheDocument();
    });

    it("sends the chosen characters with the new fanfic", async () => {
        // given
        const { createAsync } = stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.click(screen.getByRole("button", { name: "add Kanon" }));
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(createAsync).toHaveBeenCalledWith(
            expect.objectContaining({
                characters: [{ series: "umineko", character_id: "c1", character_name: "Kanon", sort_order: 0 }],
            }),
        );
    });

    it("offers no draft status while the fanfic has never been saved", () => {
        // given
        stubEditor();

        // when
        renderNew();

        // then
        expect(screen.queryByRole("option", { name: "Draft" })).not.toBeInTheDocument();
    });

    it("asks before throwing away unsaved work", async () => {
        // given
        stubEditor();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(confirm).toHaveBeenCalledWith("You have unsaved work. Discard your draft?");
        expect(navigate).not.toHaveBeenCalled();
    });

    it("leaves without asking when nothing has been written", async () => {
        // given
        stubEditor();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderNew();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(confirm).not.toHaveBeenCalled();
        expect(navigate).toHaveBeenCalledWith("/fanfiction");
    });

    it("keeps the work in progress in local storage", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "Golden Land");

        // then
        await waitFor(() => {
            expect(JSON.parse(localStorage.getItem(DRAFT_KEY) ?? "{}").title).toBe("Golden Land");
        });
    });

    it("offers to pick an unfinished draft back up", () => {
        // given
        stubEditor();
        localStorage.setItem(DRAFT_KEY, JSON.stringify({ title: "Golden Land", body: "", step: 1, tags: [] }));

        // when
        renderNew();

        // then
        expect(screen.getByRole("heading", { name: "Unfinished Draft" })).toBeInTheDocument();
        expect(screen.getByText("Golden Land")).toBeInTheDocument();
    });

    it("restores the unfinished draft into the form", async () => {
        // given
        stubEditor();
        localStorage.setItem(
            DRAFT_KEY,
            JSON.stringify({
                title: "Golden Land",
                summary: "A closed room.",
                series: "Umineko",
                customSeries: "",
                rating: "M",
                language: "English",
                customLanguage: "",
                genreA: "Mystery",
                genreB: "",
                tags: ["closed room"],
                status: "in_progress",
                characters: [],
                isPairing: false,
                isOneshot: true,
                containsLemons: false,
                body: "Beatrice laughed.",
                step: 1,
            }),
        );
        const user = userEvent.setup();
        renderNew();

        // when
        await user.click(screen.getByRole("button", { name: "Continue Draft" }));

        // then
        expect(screen.getByPlaceholderText("Your fanfic title...")).toHaveValue("Golden Land");
        expect(screen.getByPlaceholderText("Brief summary of your story...")).toHaveValue("A closed room.");
        expect(screen.getAllByRole("button", { name: "Remove tag" })).toHaveLength(1);
    });

    it("throws the unfinished draft away when the writer starts fresh", async () => {
        // given
        stubEditor();
        localStorage.setItem(DRAFT_KEY, JSON.stringify({ title: "Golden Land", body: "", step: 1, tags: [] }));
        const user = userEvent.setup();
        renderNew();

        // when
        await user.click(screen.getByRole("button", { name: "Start Fresh" }));

        // then
        expect(screen.getByPlaceholderText("Your fanfic title...")).toHaveValue("");
    });

    it("ignores a stored draft that never got a title", () => {
        // given
        stubEditor();
        localStorage.setItem(DRAFT_KEY, JSON.stringify({ title: "", body: "something", step: 1, tags: [] }));

        // when
        renderNew();

        // then
        expect(screen.queryByRole("heading", { name: "Unfinished Draft" })).not.toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "New Fanfic" })).toBeInTheDocument();
    });

    it("waits while the fanfic being edited is loading", () => {
        // given
        stubEditor({ loading: true });

        // when
        renderEdit();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("says so when there is no such fanfic to edit", () => {
        // given
        stubEditor({ fanfic: null });

        // when
        renderEdit();

        // then
        expect(screen.getByText("Fanfic not found.")).toBeInTheDocument();
    });

    it("sends an unrelated reader back to the fanfic instead of the editor", async () => {
        // given
        stubEditor({ fanfic: makeFanfic() });

        // when
        renderEdit(stranger);

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
        });
    });

    it("lets a moderator edit somebody else's fanfic", () => {
        // given
        stubEditor({ fanfic: makeFanfic() });

        // when
        renderEdit(moderator);

        // then
        expect(navigate).not.toHaveBeenCalled();
        expect(screen.getByRole("heading", { name: "Edit Fanfic" })).toBeInTheDocument();
    });

    it("asks for the same post editing permission the fanfic itself gates on", () => {
        // given
        stubEditor({ fanfic: makeFanfic() });

        // when
        renderEdit(moderator);

        // then
        expect(can).toHaveBeenCalledWith(moderator, "edit_any_post");
        expect(can).not.toHaveBeenCalledWith(moderator, "edit_any_theory");
    });

    it("keeps the editor open when the author empties the title", async () => {
        // given
        stubEditor({ fanfic: makeFanfic({ title: "Golden Land" }) });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.clear(screen.getByPlaceholderText("Your fanfic title..."));

        // then
        expect(screen.queryByText("Fanfic not found.")).not.toBeInTheDocument();
        expect(screen.getByPlaceholderText("Your fanfic title...")).toHaveValue("");
        expect(screen.getByPlaceholderText("Brief summary of your story...")).toHaveValue(
            "A closed room on Rokkenjima.",
        );
    });

    it("seeds the editor with the fanfic as it stands", () => {
        // given
        stubEditor({ fanfic: makeFanfic({ title: "Golden Land", summary: "A closed room on Rokkenjima." }) });

        // when
        renderEdit();

        // then
        expect(screen.getByPlaceholderText("Your fanfic title...")).toHaveValue("Golden Land");
        expect(screen.getByPlaceholderText("Brief summary of your story...")).toHaveValue(
            "A closed room on Rokkenjima.",
        );
        expect(screen.getAllByRole("button", { name: "Remove tag" })).toHaveLength(1);
    });

    it("moves a series the archive does not pin into the custom field", () => {
        // given
        stubEditor({ fanfic: makeFanfic({ series: "Higanbana" }) });

        // when
        renderEdit();

        // then
        expect(screen.getByPlaceholderText("Enter series name...")).toHaveValue("Higanbana");
    });

    it("moves a language the archive does not know into the custom field", () => {
        // given
        stubEditor({ fanfic: makeFanfic({ language: "Welsh" }), languages: ["English", "Japanese"] });

        // when
        renderEdit();

        // then
        expect(screen.getByPlaceholderText("Enter language...")).toHaveValue("Welsh");
    });

    it("offers the draft status only once the fanfic exists", () => {
        // given
        stubEditor({ fanfic: makeFanfic() });

        // when
        renderEdit();

        // then
        expect(screen.getByRole("option", { name: "Draft" })).toBeInTheDocument();
    });

    it("locks the one-shot toggle on a story that already has several chapters", async () => {
        // given
        stubEditor({ fanfic: makeFanfic({ chapter_count: 3, is_oneshot: false }) });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("switch", { name: "One-shot" }));

        // then
        expect(screen.getByRole("switch", { name: "One-shot" })).toHaveAttribute("aria-checked", "false");
        expect(
            screen.getByText("Cannot switch to one-shot with 3 chapters. Delete extra chapters first."),
        ).toBeInTheDocument();
    });

    it("saves a serial's details without touching its chapters", async () => {
        // given
        const { updateAsync } = stubEditor({ fanfic: makeFanfic({ is_oneshot: false }) });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith(
            expect.objectContaining({ title: "Golden Land", is_oneshot: false, genres: ["Mystery"] }),
        );
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
        });
    });

    it("loads the existing prose when moving on to edit a one-shot", async () => {
        // given
        const { updateAsync, updateChapterAsync } = stubEditor({ fanfic: makeFanfic({ is_oneshot: true }) });
        fetched.fanfic = makeFanfic({
            is_oneshot: true,
            chapters: [{ id: "chapter-1", chapter_number: 1, title: "", word_count: 3 }],
        });
        fetched.chapter = makeChapter({ id: "chapter-1", body: "<p>Beatrice laughed.</p>" });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith(expect.objectContaining({ is_oneshot: true }));
        expect(await screen.findByLabelText("story body")).toHaveValue("<p>Beatrice laughed.</p>");
        await user.click(screen.getByRole("button", { name: "Save Changes" }));
        await waitFor(() => {
            expect(updateChapterAsync).toHaveBeenCalledWith({
                chapterId: "chapter-1",
                title: "",
                body: "<p>Beatrice laughed.</p>",
            });
        });
    });

    it("writes a first chapter for a one-shot that has none yet", async () => {
        // given
        const { createChapterAsync } = stubEditor({ fanfic: makeFanfic({ is_oneshot: true }) });
        fetched.fanfic = makeFanfic({ is_oneshot: true, chapters: [] });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.type(await screen.findByLabelText("story body"), "Beatrice laughed.");
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        await waitFor(() => {
            expect(createChapterAsync).toHaveBeenCalledWith({ title: "", body: "Beatrice laughed." });
        });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
        });
    });

    it("reports why the details of an edit could not be saved", async () => {
        // given
        stubEditor({
            fanfic: makeFanfic({ is_oneshot: true }),
            update: () => Promise.reject(new Error("The witch forbids it")),
        });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));

        // then
        expect(await screen.findByText("The witch forbids it")).toBeInTheDocument();
    });

    it("goes back to the fanfic rather than the archive from an edit", async () => {
        // given
        stubEditor({ fanfic: makeFanfic() });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByText("← Back to Fanfic"));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
    });

    it("does not save a draft of an edit into local storage", async () => {
        // given
        stubEditor({ fanfic: makeFanfic() });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.type(screen.getByPlaceholderText("Your fanfic title..."), "!");

        // then
        expect(localStorage.getItem(DRAFT_KEY)).toBeNull();
    });
});
