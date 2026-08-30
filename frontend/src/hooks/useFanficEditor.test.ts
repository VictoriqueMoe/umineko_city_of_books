import { act, renderHook, waitFor } from "@testing-library/react";
import type { ChangeEvent } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { FanficChapter, FanficDetail, UserProfile } from "../types/api";
import { DRAFT_KEY } from "../domain/fanfic/draft";
import { OTHER_VALUE } from "../domain/fanfic/form";
import { useFanficEditor } from "./useFanficEditor";

const mocks = vi.hoisted(() => ({
    useFanfic: vi.fn(),
    useFanficChapter: vi.fn(),
    useFanficSeries: vi.fn(),
    useFanficLanguages: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    uploadCover: vi.fn(),
    uploadCoverFor: vi.fn(),
    deleteCover: vi.fn(),
    createChapter: vi.fn(),
    updateChapter: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("./queries/fanfic", () => ({
    useFanfic: mocks.useFanfic,
    useFanficChapter: mocks.useFanficChapter,
    useFanficSeries: mocks.useFanficSeries,
    useFanficLanguages: mocks.useFanficLanguages,
}));

vi.mock("./mutations/fanfic", () => ({
    useCreateFanfic: () => ({ mutateAsync: mocks.create }),
    useUpdateFanfic: () => ({ mutateAsync: mocks.update }),
    useUploadFanficCover: () => ({ mutateAsync: mocks.uploadCover }),
    useUploadFanficCoverFor: () => ({ mutateAsync: mocks.uploadCoverFor }),
    useDeleteFanficCover: () => ({ mutateAsync: mocks.deleteCover }),
    useCreateFanficChapter: () => ({ mutateAsync: mocks.createChapter }),
    useUpdateFanficChapter: () => ({ mutateAsync: mocks.updateChapter }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const stranger = makeUser({ id: "stranger-1", username: "battler", display_name: "Battler" });

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

interface Stubs {
    fanfic?: FanficDetail | null;
    chapter?: FanficChapter | null;
    loading?: boolean;
}

function stub(options: Stubs = {}) {
    mocks.useFanfic.mockReturnValue({
        fanfic: options.fanfic ?? null,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
    mocks.useFanficChapter.mockReturnValue({ chapter: options.chapter ?? null, loading: false, refresh: vi.fn() });
    mocks.useFanficSeries.mockReturnValue({ series: ["Umineko", "Rose Guns Days"] });
    mocks.useFanficLanguages.mockReturnValue({ languages: ["English", "Japanese"] });
}

function renderEditor(route: string, path: string, user: UserProfile | null = author) {
    return renderHook(() => useFanficEditor(), { wrapper: providerWrapper({ user, route, path }) });
}

function renderNew(user: UserProfile | null = author) {
    return renderEditor("/fanfiction/new", "/fanfiction/new", user);
}

function renderEdit(user: UserProfile | null = author) {
    return renderEditor("/fanfiction/fanfic-1/edit", "/fanfiction/:id/edit", user);
}

const coverFile = new File(["butterflies"], "cover.png", { type: "image/png" });

beforeEach(() => {
    mocks.create.mockResolvedValue({ id: "fanfic-new" });
    mocks.update.mockResolvedValue(undefined);
    mocks.uploadCover.mockResolvedValue(undefined);
    mocks.uploadCoverFor.mockResolvedValue(undefined);
    mocks.deleteCover.mockResolvedValue(undefined);
    mocks.createChapter.mockResolvedValue(undefined);
    mocks.updateChapter.mockResolvedValue(undefined);
});

describe("useFanficEditor loading the fanfic", () => {
    it("waits while the fanfic being edited is loading", () => {
        // given
        stub({ loading: true });

        // when
        const { result } = renderEdit();

        // then
        expect(result.current.loading).toBe(true);
        expect(result.current.notFound).toBe(false);
    });

    it("reports a fanfic that is not there once the wait is over", () => {
        // given
        stub({ fanfic: null });

        // when
        const { result } = renderEdit();

        // then
        expect(result.current.notFound).toBe(true);
    });

    it("never waits on a fanfic while a new one is being written", () => {
        // given
        stub({ loading: true });

        // when
        const { result } = renderNew();

        // then
        expect(result.current.loading).toBe(false);
        expect(result.current.notFound).toBe(false);
    });

    it("sends a writer who may not edit back to the fanfic", async () => {
        // given
        stub({ fanfic: makeFanfic() });

        // when
        renderEdit(stranger);

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/fanfiction/fanfic-1");
        });
    });
});

describe("useFanficEditor loading the prose", () => {
    it("asks for the first chapter of a one-shot that has one", () => {
        // given
        stub({
            fanfic: makeFanfic({
                is_oneshot: true,
                chapters: [{ id: "chapter-1", chapter_number: 1, title: "", word_count: 3 }],
            }),
            chapter: makeChapter(),
        });

        // when
        const { result } = renderEdit();

        // then
        expect(mocks.useFanficChapter).toHaveBeenCalledWith("fanfic-1", 1);
        expect(result.current.story.body).toBe("<p>Beatrice laughed.</p>");
    });

    it("asks for no chapter of a one-shot that has none yet", () => {
        // given
        stub({ fanfic: makeFanfic({ is_oneshot: true, chapters: [] }) });

        // when
        const { result } = renderEdit();

        // then
        expect(mocks.useFanficChapter).toHaveBeenCalledWith("fanfic-1", 0);
        expect(result.current.story.body).toBe("");
    });

    it("asks for no chapter of a serial, whose chapters have their own editor", () => {
        // given
        stub({
            fanfic: makeFanfic({
                is_oneshot: false,
                chapters: [{ id: "chapter-1", chapter_number: 1, title: "", word_count: 3 }],
            }),
        });

        // when
        renderEdit();

        // then
        expect(mocks.useFanficChapter).toHaveBeenCalledWith("fanfic-1", 0);
    });

    it("asks for no chapter at all while a new fanfic is being written", () => {
        // given
        stub();

        // when
        renderNew();

        // then
        expect(mocks.useFanficChapter).toHaveBeenCalledWith("", 0);
    });

    it("lets the writer's prose win over the chapter it loaded", () => {
        // given
        stub({
            fanfic: makeFanfic({
                is_oneshot: true,
                chapters: [{ id: "chapter-1", chapter_number: 1, title: "", word_count: 3 }],
            }),
            chapter: makeChapter(),
        });
        const { result } = renderEdit();

        // when
        act(() => {
            result.current.story.setBody("<p>Beatrice wept.</p>");
        });

        // then
        expect(result.current.story.body).toBe("<p>Beatrice wept.</p>");
    });

    it("updates the chapter it loaded rather than writing a second one", async () => {
        // given
        stub({
            fanfic: makeFanfic({
                is_oneshot: true,
                chapters: [{ id: "chapter-1", chapter_number: 1, title: "", word_count: 3 }],
            }),
            chapter: makeChapter({ id: "chapter-1" }),
        });
        const { result } = renderEdit();

        // when
        await act(async () => {
            result.current.story.saveChapter();
        });

        // then
        expect(mocks.updateChapter).toHaveBeenCalledWith({
            chapterId: "chapter-1",
            title: "",
            body: "<p>Beatrice laughed.</p>",
        });
        expect(mocks.createChapter).not.toHaveBeenCalled();
    });
});

describe("useFanficEditor moving on to the story", () => {
    it("names the series an edit left empty rather than lumping it with the language", async () => {
        // given
        stub({ fanfic: makeFanfic({ is_oneshot: true }) });
        const { result } = renderEdit();
        act(() => {
            result.current.details.chooseSeries(OTHER_VALUE);
        });

        // when
        await act(async () => {
            result.current.details.next();
        });

        // then
        expect(result.current.error).toBe("Series is required");
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("names the language an edit left empty rather than lumping it with the series", async () => {
        // given
        stub({ fanfic: makeFanfic({ is_oneshot: true }) });
        const { result } = renderEdit();
        act(() => {
            result.current.details.chooseLanguage(OTHER_VALUE);
        });

        // when
        await act(async () => {
            result.current.details.next();
        });

        // then
        expect(result.current.error).toBe("Language is required");
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("saves nothing on the way to the story of a brand new fanfic", async () => {
        // given
        stub();
        const { result } = renderNew();
        act(() => {
            result.current.details.setField("title", "Golden Land");
        });

        // when
        await act(async () => {
            result.current.details.next();
        });

        // then
        expect(result.current.step).toBe(2);
        expect(mocks.update).not.toHaveBeenCalled();
        expect(mocks.create).not.toHaveBeenCalled();
    });
});

describe("useFanficEditor the cover image", () => {
    it("stops at the details when the cover of an edit will not upload", async () => {
        // given
        stub({ fanfic: makeFanfic({ is_oneshot: true }) });
        mocks.uploadCover.mockRejectedValue(new Error("The cover is too large"));
        const { result } = renderEdit();
        act(() => {
            result.current.details.chooseCover({
                target: { files: [coverFile] },
            } as unknown as ChangeEvent<HTMLInputElement>);
        });

        // when
        await act(async () => {
            result.current.details.next();
        });

        // then
        expect(result.current.error).toBe("The fanfic was saved but its cover image was not: The cover is too large");
        expect(result.current.step).toBe(1);
    });

    it("says so in its own words when the upload failed without one", async () => {
        // given
        stub({ fanfic: makeFanfic({ is_oneshot: true }) });
        mocks.uploadCover.mockRejectedValue({});
        const { result } = renderEdit();
        act(() => {
            result.current.details.chooseCover({
                target: { files: [coverFile] },
            } as unknown as ChangeEvent<HTMLInputElement>);
        });

        // when
        await act(async () => {
            result.current.details.next();
        });

        // then
        expect(result.current.error).toBe("The fanfic was saved but its cover image was not");
    });

    it("stays on the editor when the cover of an edit will not be removed", async () => {
        // given
        stub({ fanfic: makeFanfic({ is_oneshot: false, cover_image_url: "/covers/1.png" }) });
        mocks.deleteCover.mockRejectedValue(new Error("The witch forbids it"));
        const { result } = renderEdit();
        act(() => {
            result.current.details.removeCover();
        });

        // when
        await act(async () => {
            result.current.details.save();
        });

        // then
        expect(result.current.error).toBe(
            "The fanfic was saved but its cover image was not removed: The witch forbids it",
        );
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("never asks to remove a cover a brand new fanfic never had", async () => {
        // given
        stub();
        const { result } = renderNew();
        act(() => {
            result.current.details.setField("title", "Golden Land");
            result.current.details.removeCover();
        });

        // when
        await act(async () => {
            result.current.story.publish();
        });

        // then
        expect(mocks.deleteCover).not.toHaveBeenCalled();
        expect(mocks.navigate).toHaveBeenCalledWith("/fanfiction/fanfic-new");
    });

    it("keeps the fanfic it already created when only the cover failed", async () => {
        // given
        stub();
        mocks.uploadCoverFor.mockRejectedValue(new Error("The cover is too large"));
        const { result } = renderNew();
        act(() => {
            result.current.details.setField("title", "Golden Land");
            result.current.details.chooseCover({
                target: { files: [coverFile] },
            } as unknown as ChangeEvent<HTMLInputElement>);
        });
        await act(async () => {
            result.current.story.publish();
        });

        // when
        await act(async () => {
            result.current.story.publish();
        });

        // then
        expect(mocks.create).toHaveBeenCalledTimes(1);
        expect(mocks.uploadCoverFor).toHaveBeenCalledTimes(2);
        expect(mocks.navigate).not.toHaveBeenCalled();
    });
});

describe("useFanficEditor the unfinished draft", () => {
    it("hides the form until the writer answers the unfinished draft prompt", () => {
        // given
        localStorage.setItem(DRAFT_KEY, JSON.stringify({ title: "Golden Land", body: "", step: 1, tags: [] }));
        stub();

        // when
        const { result } = renderNew();

        // then
        expect(result.current.draftPrompt?.title).toBe("Golden Land");
        expect(result.current.hidden).toBe(true);
    });

    it("never offers a stored draft while a fanfic is being edited", () => {
        // given
        localStorage.setItem(DRAFT_KEY, JSON.stringify({ title: "Golden Land", body: "", step: 1, tags: [] }));
        stub({ fanfic: makeFanfic() });

        // when
        const { result } = renderEdit();

        // then
        expect(result.current.draftPrompt).toBeNull();
        expect(result.current.hidden).toBe(false);
    });

    it("puts the writer back where the draft left off", () => {
        // given
        localStorage.setItem(
            DRAFT_KEY,
            JSON.stringify({ title: "Golden Land", body: "Beatrice laughed.", step: 2, tags: ["closed room"] }),
        );
        stub();
        const { result } = renderNew();

        // when
        act(() => {
            result.current.resumeDraft();
        });

        // then
        expect(result.current.details.fields.title).toBe("Golden Land");
        expect(result.current.details.fields.tags).toEqual(["closed room"]);
        expect(result.current.story.body).toBe("Beatrice laughed.");
        expect(result.current.step).toBe(2);
    });
});
