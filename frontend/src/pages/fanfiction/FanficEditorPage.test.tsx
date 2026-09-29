import { screen, waitFor } from "@testing-library/react";
import userEvent, { type UserEvent } from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { FanficChapter, FanficDetail, ShipCharacter, UserProfile } from "../../types/api";
import { FanficEditorPage } from "./FanficEditorPage";

const {
    useFanfic,
    useFanficChapter,
    useFanficSeries,
    useFanficLanguages,
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
    useFanficChapter: vi.fn(),
    useFanficSeries: vi.fn(),
    useFanficLanguages: vi.fn(),
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

vi.mock("../../hooks/queries/fanfic", () => ({
    useFanfic,
    useFanficChapter,
    useFanficLanguages,
    useFanficSeries,
}));
vi.mock("../../hooks/mutations/fanfic", () => ({
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
vi.mock("../../domain/permissions", async importOriginal => {
    const actual = await importOriginal<typeof import("../../domain/permissions")>();
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
const TITLE = "Your fanfic title...";
const SUMMARY = "Brief summary of your story...";
const TAGS = "Type a tag and press Enter...";
const COVER_TOO_LARGE = "The fanfic was saved but its cover image was not: The cover is too large";
const COMBOBOX = { series: 0, rating: 1, language: 2, genreA: 3, genreB: 4 };

const ROUTES = {
    new: { route: "/fanfiction/new" },
    edit: { route: "/fanfiction/fanfic-1/edit", path: "/fanfiction/:id/edit" },
};

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

interface StubOptions {
    fanfic?: FanficDetail | null;
    chapter?: FanficChapter | null;
    loading?: boolean;
    series?: string[];
    languages?: string[];
    create?: () => Promise<{ id: string }>;
    update?: () => Promise<unknown>;
    uploadCover?: () => Promise<unknown>;
    uploadCoverFor?: () => Promise<unknown>;
    deleteCover?: () => Promise<unknown>;
}

function openEditor(mode: keyof typeof ROUTES, options: StubOptions = {}, viewer: UserProfile = author) {
    useFanfic.mockReturnValue({ fanfic: options.fanfic ?? null, loading: options.loading ?? false, refresh: vi.fn() });
    useFanficChapter.mockReturnValue({ chapter: options.chapter ?? null, loading: false, refresh: vi.fn() });
    useFanficSeries.mockReturnValue({ series: options.series ?? ["Umineko", "Higurashi", "Rose Guns Days"] });
    useFanficLanguages.mockReturnValue({ languages: options.languages ?? ["English", "Japanese"] });

    const mutations = {
        createAsync: vi.fn(options.create ?? (() => Promise.resolve({ id: "fanfic-new" }))),
        updateAsync: vi.fn(options.update ?? (() => Promise.resolve({}))),
        uploadCoverAsync: vi.fn(options.uploadCover ?? (() => Promise.resolve({}))),
        uploadCoverForAsync: vi.fn(options.uploadCoverFor ?? (() => Promise.resolve({}))),
        deleteCoverAsync: vi.fn(options.deleteCover ?? (() => Promise.resolve({}))),
        createChapterAsync: vi.fn(() => Promise.resolve({})),
        updateChapterAsync: vi.fn(() => Promise.resolve({})),
    };
    useCreateFanfic.mockReturnValue({ mutateAsync: mutations.createAsync });
    useUpdateFanfic.mockReturnValue({ mutateAsync: mutations.updateAsync });
    useUploadFanficCover.mockReturnValue({ mutateAsync: mutations.uploadCoverAsync });
    useUploadFanficCoverFor.mockReturnValue({ mutateAsync: mutations.uploadCoverForAsync });
    useDeleteFanficCover.mockReturnValue({ mutateAsync: mutations.deleteCoverAsync });
    useCreateFanficChapter.mockReturnValue({ mutateAsync: mutations.createChapterAsync });
    useUpdateFanficChapter.mockReturnValue({ mutateAsync: mutations.updateChapterAsync });

    const user = userEvent.setup();
    const { container } = renderWithProviders(<FanficEditorPage />, { user: viewer, ...ROUTES[mode] });

    return { ...mutations, user, container };
}

async function chooseCover(user: UserEvent, container: HTMLElement) {
    const input = container.querySelector<HTMLInputElement>('input[type="file"]');
    if (!input) {
        throw new Error("the form has no cover input");
    }

    await user.upload(input, new File(["butterflies"], "cover.png", { type: "image/png" }));
}

async function publishNew(user: UserEvent) {
    await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
    await user.click(screen.getByRole("button", { name: "Publish" }));
}

describe("FanficEditorPage", () => {
    const freshStartCases: { name: string; stored: object | null }[] = [
        { name: "starts a brand new fanfic on the details step", stored: null },
        {
            name: "ignores a stored draft that never got a title",
            stored: { title: "", body: "something", step: 1, tags: [] },
        },
    ];

    it.each(freshStartCases)("$name", ({ stored }) => {
        // given
        if (stored) {
            localStorage.setItem(DRAFT_KEY, JSON.stringify(stored));
        }

        // when
        openEditor("new", { series: ["Umineko", "Rose Guns Days"] });

        // then
        expect(screen.queryByRole("heading", { name: "Unfinished Draft" })).not.toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "New Fanfic" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText(TITLE)).toHaveValue("");
        expect(screen.queryByRole("option", { name: "Draft" })).not.toBeInTheDocument();
        const seriesOptions = screen.getAllByRole("option").map(o => o.textContent);
        expect(seriesOptions).toContain("Umineko");
        expect(seriesOptions).toContain("Higurashi");
        expect(seriesOptions).toContain("Ciconia");
        expect(seriesOptions).toContain("Rose Guns Days");
    });

    const validationCases: { name: string; title: string; other: number | null; error: string }[] = [
        { name: "refuses to move on without a title", title: "", other: null, error: "Title is required" },
        {
            name: "refuses to move on with an empty custom series",
            title: "Golden Land",
            other: COMBOBOX.series,
            error: "Series is required",
        },
        {
            name: "refuses to move on with an empty custom language",
            title: "Golden Land",
            other: COMBOBOX.language,
            error: "Language is required",
        },
    ];

    it.each(validationCases)("$name", async ({ title, other, error }) => {
        // given
        const { user } = openEditor("new");

        // when
        if (title) {
            await user.type(screen.getByPlaceholderText(TITLE), title);
        }
        if (other !== null) {
            await user.selectOptions(screen.getAllByRole("combobox")[other], "__other__");
        }
        await user.click(screen.getByRole("button", { name: /^Next:/ }));

        // then
        expect(screen.getByText(error)).toBeInTheDocument();
        expect(screen.queryByLabelText("story body")).not.toBeInTheDocument();
    });

    const storyStepCases = [
        {
            name: "calls the second step writing the story for a one-shot",
            serial: false,
            next: "Next: Edit Story",
            heading: "Write Your Story",
            placeholder: "Write your story here...",
        },
        {
            name: "calls the second step writing the first chapter for a serial",
            serial: true,
            next: "Next: Write Story",
            heading: "Write First Chapter",
            placeholder: "Write your first chapter here...",
        },
    ];

    it.each(storyStepCases)("$name", async ({ serial, next, heading, placeholder }) => {
        // given
        const { user } = openEditor("new");

        // when
        await user.type(screen.getByPlaceholderText(TITLE), "Golden Land");
        if (serial) {
            await user.click(screen.getByRole("switch", { name: "One-shot" }));
        }
        await user.click(screen.getByRole("button", { name: next }));

        // then
        expect(screen.getByRole("heading", { name: heading })).toBeInTheDocument();
        expect(screen.getByPlaceholderText(placeholder)).toBeInTheDocument();
    });

    const createCases = [
        { name: "publishes a new fanfic and opens it", button: "Publish", status: "in_progress", genreB: "" },
        { name: "saves a new fanfic as a draft", button: "Save as Draft", status: "draft", genreB: "Mystery" },
    ];

    it.each(createCases)("$name", async ({ button, status, genreB }) => {
        // given
        const { user, createAsync } = openEditor("new");

        // when
        await user.type(screen.getByPlaceholderText(TITLE), "  Golden Land  ");
        await user.type(screen.getByPlaceholderText(SUMMARY), "A closed room.");
        await user.selectOptions(screen.getAllByRole("combobox")[COMBOBOX.rating], "M");
        await user.selectOptions(screen.getAllByRole("combobox")[COMBOBOX.genreA], "Mystery");
        if (genreB) {
            await user.selectOptions(screen.getAllByRole("combobox")[COMBOBOX.genreB], genreB);
        }
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.type(screen.getByLabelText("story body"), "Beatrice laughed.");
        await user.click(screen.getByRole("button", { name: button }));

        // then
        expect(createAsync).toHaveBeenCalledWith({
            title: "Golden Land",
            summary: "A closed room.",
            series: "Umineko",
            rating: "M",
            language: "English",
            status,
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

    it("sends a typed custom series and language instead of the pinned ones", async () => {
        // given
        const { user, createAsync } = openEditor("new");

        // when
        await user.type(screen.getByPlaceholderText(TITLE), "Golden Land");
        await user.selectOptions(screen.getAllByRole("combobox")[COMBOBOX.series], "__other__");
        await user.type(screen.getByPlaceholderText("Enter series name..."), "  Higanbana  ");
        await user.selectOptions(screen.getAllByRole("combobox")[COMBOBOX.language], "__other__");
        await user.type(screen.getByPlaceholderText("Enter language..."), "  Welsh  ");
        await publishNew(user);

        // then
        expect(createAsync).toHaveBeenCalledWith(expect.objectContaining({ series: "Higanbana", language: "Welsh" }));
    });

    it("reports why the fanfic could not be created", async () => {
        // given
        const { user } = openEditor("new", { create: () => Promise.reject(new Error("The witch forbids it")) });

        // when
        await user.type(screen.getByPlaceholderText(TITLE), "Golden Land");
        await publishNew(user);

        // then
        expect(await screen.findByText("The witch forbids it")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("adds a tag on enter, refuses a repeat whatever the casing, drops one on request and stops at ten", async () => {
        // given
        const { user } = openEditor("new");
        const field = screen.getByPlaceholderText(TAGS);

        // when
        await user.type(field, "closed room{Enter}");

        // then
        expect(screen.getByText(/closed room/)).toBeInTheDocument();
        expect(field).toHaveValue("");

        // when
        await user.type(field, "Closed Room{Enter}");

        // then
        expect(screen.getAllByRole("button", { name: "Remove tag" })).toHaveLength(1);

        // when
        await user.click(screen.getByRole("button", { name: "Remove tag" }));

        // then
        expect(screen.queryByRole("button", { name: "Remove tag" })).not.toBeInTheDocument();

        // when
        for (let i = 0; i < 12; i++) {
            await user.type(field, `tag${i}{Enter}`);
        }

        // then
        expect(screen.getAllByRole("button", { name: "Remove tag" })).toHaveLength(10);
    });

    it("adds and drops a character and sends the chosen ones with the new fanfic", async () => {
        // given
        const { user, createAsync } = openEditor("new");
        await user.type(screen.getByPlaceholderText(TITLE), "Golden Land");

        // when
        await user.click(screen.getByRole("button", { name: "add Kanon" }));

        // then
        expect(screen.getByText("1 chosen")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Remove character" }));

        // then
        expect(screen.getByText("0 chosen")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "add Kanon" }));
        await publishNew(user);

        // then
        expect(createAsync).toHaveBeenCalledWith(
            expect.objectContaining({
                characters: [{ series: "umineko", character_id: "c1", character_name: "Kanon", sort_order: 0 }],
            }),
        );
    });

    it("asks before throwing away unsaved work", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { user } = openEditor("new");

        // when
        await user.type(screen.getByPlaceholderText(TITLE), "Golden Land");
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(confirm).toHaveBeenCalledWith("You have unsaved work. Discard your draft?");
        expect(navigate).not.toHaveBeenCalled();
    });

    it("leaves without asking when nothing has been written", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { user } = openEditor("new");

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(confirm).not.toHaveBeenCalled();
        expect(navigate).toHaveBeenCalledWith("/fanfiction");
    });

    it("keeps the work in progress in local storage", async () => {
        // given
        const { user } = openEditor("new");

        // when
        await user.type(screen.getByPlaceholderText(TITLE), "Golden Land");

        // then
        await waitFor(() => {
            expect(JSON.parse(localStorage.getItem(DRAFT_KEY) ?? "{}").title).toBe("Golden Land");
        });
    });

    const resumeCases = [
        {
            name: "restores the unfinished draft into the form",
            button: "Continue Draft",
            title: "Golden Land",
            summary: "A closed room.",
            tags: 1,
        },
        {
            name: "throws the unfinished draft away when the writer starts fresh",
            button: "Start Fresh",
            title: "",
            summary: "",
            tags: 0,
        },
    ];

    it.each(resumeCases)("$name", async ({ button, title, summary, tags }) => {
        // given
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

        // when
        const { user } = openEditor("new");

        // then
        expect(screen.getByRole("heading", { name: "Unfinished Draft" })).toBeInTheDocument();
        expect(screen.getByText("Golden Land")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: button }));

        // then
        expect(screen.getByPlaceholderText(TITLE)).toHaveValue(title);
        expect(screen.getByPlaceholderText(SUMMARY)).toHaveValue(summary);
        expect(screen.queryAllByRole("button", { name: "Remove tag" })).toHaveLength(tags);
    });

    const unavailableCases: { name: string; stub: StubOptions; shown: string }[] = [
        { name: "waits while the fanfic being edited is loading", stub: { loading: true }, shown: "Loading..." },
        { name: "says so when there is no such fanfic to edit", stub: { fanfic: null }, shown: "Fanfic not found." },
    ];

    it.each(unavailableCases)("$name", ({ stub, shown }) => {
        // given the stubbed fanfic, from the table row

        // when
        openEditor("edit", stub);

        // then
        expect(screen.getByText(shown)).toBeInTheDocument();
    });

    it("sends an unrelated reader back to the fanfic instead of the editor", async () => {
        // given
        const fanfic = makeFanfic();

        // when
        openEditor("edit", { fanfic }, stranger);

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
        });
    });

    it("lets a moderator edit somebody else's fanfic on the post editing permission", () => {
        // given
        const fanfic = makeFanfic();

        // when
        openEditor("edit", { fanfic }, moderator);

        // then
        expect(navigate).not.toHaveBeenCalled();
        expect(screen.getByRole("heading", { name: "Edit Fanfic" })).toBeInTheDocument();
        expect(can).toHaveBeenCalledWith(moderator, "edit_any_post");
        expect(can).not.toHaveBeenCalledWith(moderator, "edit_any_theory");
    });

    it("keeps the editor open when the author empties the title", async () => {
        // given
        const { user } = openEditor("edit", { fanfic: makeFanfic({ title: "Golden Land" }) });

        // when
        await user.clear(screen.getByPlaceholderText(TITLE));

        // then
        expect(screen.queryByText("Fanfic not found.")).not.toBeInTheDocument();
        expect(screen.getByPlaceholderText(TITLE)).toHaveValue("");
        expect(screen.getByPlaceholderText(SUMMARY)).toHaveValue("A closed room on Rokkenjima.");
    });

    it("seeds the editor with the fanfic as it stands", () => {
        // given
        const fanfic = makeFanfic();

        // when
        openEditor("edit", { fanfic });

        // then
        expect(screen.getByPlaceholderText(TITLE)).toHaveValue("Golden Land");
        expect(screen.getByPlaceholderText(SUMMARY)).toHaveValue("A closed room on Rokkenjima.");
        expect(screen.getAllByRole("button", { name: "Remove tag" })).toHaveLength(1);
        expect(screen.getByRole("option", { name: "Draft" })).toBeInTheDocument();
    });

    const customFieldCases: { name: string; fanfic: Partial<FanficDetail>; field: string; value: string }[] = [
        {
            name: "moves a series the archive does not pin into the custom field",
            fanfic: { series: "Higanbana" },
            field: "Enter series name...",
            value: "Higanbana",
        },
        {
            name: "moves a language the archive does not know into the custom field",
            fanfic: { language: "Welsh" },
            field: "Enter language...",
            value: "Welsh",
        },
    ];

    it.each(customFieldCases)("$name", ({ fanfic, field, value }) => {
        // given
        const unlisted = makeFanfic(fanfic);

        // when
        openEditor("edit", { fanfic: unlisted, languages: ["English", "Japanese"] });

        // then
        expect(screen.getByPlaceholderText(field)).toHaveValue(value);
    });

    it("locks the one-shot toggle on a story that already has several chapters", async () => {
        // given
        const { user } = openEditor("edit", { fanfic: makeFanfic({ chapter_count: 3, is_oneshot: false }) });

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
        const { user, updateAsync } = openEditor("edit", { fanfic: makeFanfic({ is_oneshot: false }) });

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
        const { user, updateAsync, updateChapterAsync } = openEditor("edit", {
            fanfic: makeFanfic({
                is_oneshot: true,
                chapters: [{ id: "chapter-1", chapter_number: 1, title: "", word_count: 3 }],
            }),
            chapter: {
                id: "chapter-1",
                chapter_number: 1,
                title: "",
                body: "<p>Beatrice laughed.</p>",
                word_count: 3,
                has_prev: false,
                has_next: false,
                created_at: "2026-01-01T00:00:00Z",
            },
        });

        // when
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));

        // then
        expect(useFanficChapter).toHaveBeenCalledWith("fanfic-1", 1);
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
        const { user, createChapterAsync } = openEditor("edit", {
            fanfic: makeFanfic({ is_oneshot: true, chapters: [] }),
        });

        // when
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));
        await user.type(await screen.findByLabelText("story body"), "Beatrice laughed.");
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        expect(useFanficChapter).toHaveBeenCalledWith("fanfic-1", 0);
        await waitFor(() => {
            expect(createChapterAsync).toHaveBeenCalledWith({ title: "", body: "Beatrice laughed." });
        });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
        });
    });

    it("reports why the details of an edit could not be saved", async () => {
        // given
        const { user } = openEditor("edit", {
            fanfic: makeFanfic({ is_oneshot: true }),
            update: () => Promise.reject(new Error("The witch forbids it")),
        });

        // when
        await user.click(screen.getByRole("button", { name: "Next: Edit Story" }));

        // then
        expect(await screen.findByText("The witch forbids it")).toBeInTheDocument();
    });

    it("goes back to the fanfic rather than the archive from an edit", async () => {
        // given
        const { user } = openEditor("edit", { fanfic: makeFanfic() });

        // when
        await user.click(screen.getByText("← Back to Fanfic"));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
    });

    it("uploads a chosen cover against the fanfic being edited", async () => {
        // given
        const { user, container, uploadCoverAsync } = openEditor("edit", { fanfic: makeFanfic({ is_oneshot: false }) });

        // when
        await chooseCover(user, container);
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        await waitFor(() => {
            expect(uploadCoverAsync).toHaveBeenCalledWith(expect.any(File));
        });
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
    });

    const coverFailureCases: {
        name: string;
        fanfic: Partial<FanficDetail>;
        failure: StubOptions;
        removesCover: boolean;
        button: string;
        message: string;
    }[] = [
        {
            name: "says the cover was not saved rather than opening a fanfic that quietly lost it",
            fanfic: { is_oneshot: false },
            failure: { uploadCover: () => Promise.reject(new Error("The cover is too large")) },
            removesCover: false,
            button: "Save Changes",
            message: COVER_TOO_LARGE,
        },
        {
            name: "says the cover was not saved rather than moving on to the story",
            fanfic: { is_oneshot: true, chapters: [] },
            failure: { uploadCover: () => Promise.reject(new Error("The cover is too large")) },
            removesCover: false,
            button: "Next: Edit Story",
            message: COVER_TOO_LARGE,
        },
        {
            name: "says the cover was not removed rather than opening a fanfic that still has it",
            fanfic: { is_oneshot: false, cover_image_url: "/covers/1.png" },
            failure: { deleteCover: () => Promise.reject(new Error("The witch forbids it")) },
            removesCover: true,
            button: "Save Changes",
            message: "The fanfic was saved but its cover image was not removed: The witch forbids it",
        },
    ];

    it.each(coverFailureCases)("$name", async ({ fanfic, failure, removesCover, button, message }) => {
        // given
        const { user, container } = openEditor("edit", { fanfic: makeFanfic(fanfic), ...failure });

        // when
        if (removesCover) {
            await user.click(screen.getByRole("button", { name: "Remove" }));
        } else {
            await chooseCover(user, container);
        }
        await user.click(screen.getByRole("button", { name: button }));

        // then
        expect(await screen.findByText(message)).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
        expect(screen.queryByLabelText("story body")).not.toBeInTheDocument();
    });

    it("uploads a chosen cover against the fanfic it has just created", async () => {
        // given
        const { user, container, uploadCoverForAsync } = openEditor("new");

        // when
        await user.type(screen.getByPlaceholderText(TITLE), "Golden Land");
        await chooseCover(user, container);
        await publishNew(user);

        // then
        await waitFor(() => {
            expect(uploadCoverForAsync).toHaveBeenCalledWith({ id: "fanfic-new", file: expect.any(File) });
        });
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-new");
    });

    it("says the cover of a new fanfic was not saved and never writes the fanfic twice", async () => {
        // given
        const { user, container, createAsync } = openEditor("new", {
            uploadCoverFor: () => Promise.reject(new Error("The cover is too large")),
        });
        await user.type(screen.getByPlaceholderText(TITLE), "Golden Land");
        await chooseCover(user, container);

        // when
        await publishNew(user);

        // then
        expect(await screen.findByText(COVER_TOO_LARGE)).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();

        // when
        await user.click(screen.getByRole("button", { name: "Publish" }));

        // then
        expect(createAsync).toHaveBeenCalledTimes(1);
    });

    it("does not save a draft of an edit into local storage", async () => {
        // given
        const { user } = openEditor("edit", { fanfic: makeFanfic() });

        // when
        await user.type(screen.getByPlaceholderText(TITLE), "!");

        // then
        expect(localStorage.getItem(DRAFT_KEY)).toBeNull();
    });
});
