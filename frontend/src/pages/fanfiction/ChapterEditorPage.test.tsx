import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { FanficChapter } from "../../types/api";
import { ChapterEditorPage } from "./ChapterEditorPage";

const { useFanficChapter, useCreateFanficChapter, useUpdateFanficChapter, navigate } = vi.hoisted(() => ({
    useFanficChapter: vi.fn(),
    useCreateFanficChapter: vi.fn(),
    useUpdateFanficChapter: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/fanfic", () => ({ useFanficChapter }));
vi.mock("../../api/mutations/fanfic", () => ({ useCreateFanficChapter, useUpdateFanficChapter }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

interface EditorStubProps {
    content: string;
    onChange: (html: string) => void;
    placeholder?: string;
}

vi.mock("../../components/RichTextEditor/RichTextEditor", () => ({
    RichTextEditor: (props: EditorStubProps) => (
        <textarea
            aria-label="chapter body"
            placeholder={props.placeholder}
            value={props.content}
            onChange={e => props.onChange(e.target.value)}
        />
    ),
}));

function makeChapter(overrides: Partial<FanficChapter> = {}): FanficChapter {
    return {
        id: "chapter-2",
        chapter_number: 2,
        title: "The Witch",
        body: "<p>Beatrice laughed.</p>",
        word_count: 1500,
        has_prev: true,
        has_next: false,
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    chapter?: FanficChapter | null;
    loading?: boolean;
    create?: () => Promise<unknown>;
    update?: () => Promise<unknown>;
}

function stubEditor(options: StubOptions = {}) {
    useFanficChapter.mockReturnValue({
        chapter: options.chapter === undefined ? makeChapter() : options.chapter,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
    const createAsync = vi.fn(options.create ?? (() => Promise.resolve({ id: "chapter-new" })));
    const updateAsync = vi.fn(options.update ?? (() => Promise.resolve({})));
    useCreateFanficChapter.mockReturnValue({ mutateAsync: createAsync });
    useUpdateFanficChapter.mockReturnValue({ mutateAsync: updateAsync });

    return { createAsync, updateAsync };
}

function renderNew() {
    return renderWithProviders(<ChapterEditorPage />, {
        route: "/fanfiction/fanfic-1/chapter/new",
        path: "/fanfiction/:id/chapter/:number",
    });
}

function renderEdit() {
    return renderWithProviders(<ChapterEditorPage />, {
        route: "/fanfiction/fanfic-1/chapter/2/edit",
        path: "/fanfiction/:id/chapter/:number/edit",
    });
}

describe("ChapterEditorPage", () => {
    it("does not fetch a chapter when adding a new one", () => {
        // given
        stubEditor({ chapter: null });

        // when
        renderNew();

        // then
        expect(useFanficChapter).toHaveBeenCalledWith("", 0);
        expect(screen.getByRole("heading", { name: "Add Chapter" })).toBeInTheDocument();
    });

    it("waits while an existing chapter is loading", () => {
        // given
        stubEditor({ loading: true });

        // when
        renderEdit();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("does not wait on a chapter it is not fetching", () => {
        // given
        stubEditor({ chapter: null, loading: true });

        // when
        renderNew();

        // then
        expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
    });

    it("says so when the chapter being edited does not exist", () => {
        // given
        stubEditor({ chapter: null });

        // when
        renderEdit();

        // then
        expect(screen.getByText("Chapter not found.")).toBeInTheDocument();
    });

    it("starts a new chapter with an empty form", () => {
        // given
        stubEditor({ chapter: null });

        // when
        renderNew();

        // then
        expect(screen.getByPlaceholderText("Chapter title...")).toHaveValue("");
        expect(screen.getByLabelText("chapter body")).toHaveValue("");
        expect(screen.getByRole("button", { name: "Save Chapter" })).toBeDisabled();
    });

    it("seeds the editor with the chapter as it stands", () => {
        // given
        stubEditor({ chapter: makeChapter({ title: "The Witch", body: "<p>Beatrice laughed.</p>" }) });

        // when
        renderEdit();

        // then
        expect(screen.getByRole("heading", { name: "Edit Chapter 2" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Chapter title...")).toHaveValue("The Witch");
        expect(screen.getByLabelText("chapter body")).toHaveValue("<p>Beatrice laughed.</p>");
    });

    it("creates a new chapter and returns to the story", async () => {
        // given
        const { createAsync } = stubEditor({ chapter: null });
        const user = userEvent.setup();
        renderNew();

        // when
        await user.type(screen.getByPlaceholderText("Chapter title..."), "  The Arrival  ");
        await user.type(screen.getByLabelText("chapter body"), "Beatrice laughed.");
        await user.click(screen.getByRole("button", { name: "Save Chapter" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({ title: "The Arrival", body: "Beatrice laughed." });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
        });
    });

    it("saves an edited chapter and reopens it", async () => {
        // given
        const { updateAsync } = stubEditor({ chapter: makeChapter({ id: "chapter-2", title: "The Witch" }) });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Save Chapter" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith({
            chapterId: "chapter-2",
            title: "The Witch",
            body: "<p>Beatrice laughed.</p>",
        });
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/chapter/2");
        });
    });

    it("reports why the chapter could not be saved", async () => {
        // given
        stubEditor({ update: () => Promise.reject(new Error("The witch forbids it")) });
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByRole("button", { name: "Save Chapter" }));

        // then
        expect(await screen.findByText("The witch forbids it")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("refuses to save a chapter whose body was emptied", async () => {
        // given
        const { updateAsync } = stubEditor();
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.clear(screen.getByLabelText("chapter body"));

        // then
        expect(screen.getByRole("button", { name: "Save Chapter" })).toBeDisabled();
        expect(updateAsync).not.toHaveBeenCalled();
    });

    it("abandons a new chapter back to the story", async () => {
        // given
        stubEditor({ chapter: null });
        const user = userEvent.setup();
        renderNew();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
        expect(screen.getByText("← Back to Fanfic")).toBeInTheDocument();
    });

    it("abandons an edit back to the chapter it started from", async () => {
        // given
        stubEditor();
        const user = userEvent.setup();
        renderEdit();

        // when
        await user.click(screen.getByText("← Back to Chapter"));

        // then
        expect(navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1/chapter/2");
    });
});
