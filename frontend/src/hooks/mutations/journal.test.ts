import { act, renderHook } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../api/queryKeys";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { CreateJournalPayload, JournalDetail, JournalEntryPayload } from "../../types/api";
import {
    useCreateJournal,
    useCreateJournalComment,
    useCreateJournalEntry,
    useDeleteJournal,
    useDeleteJournalComment,
    useDeleteJournalEntry,
    useDeleteJournalEntryMedia,
    useFollowJournal,
    useLikeJournalComment,
    useSetJournalPaused,
    useUnfollowJournal,
    useUnlikeJournalComment,
    useUpdateJournal,
    useUpdateJournalComment,
    useUpdateJournalEntry,
    useUploadJournalCommentMedia,
    useUploadJournalEntryMedia,
} from "./journal";

const mocks = vi.hoisted(() => ({
    createJournal: vi.fn(),
    createJournalComment: vi.fn(),
    createJournalEntry: vi.fn(),
    deleteJournal: vi.fn(),
    deleteJournalEntry: vi.fn(),
    deleteJournalEntryMedia: vi.fn(),
    followJournal: vi.fn(),
    setJournalPaused: vi.fn(),
    unfollowJournal: vi.fn(),
    updateJournal: vi.fn(),
    updateJournalEntry: vi.fn(),
    uploadJournalEntryMedia: vi.fn(),
}));

const commentMocks = vi.hoisted(() => ({
    deleteJournalComment: vi.fn(),
    likeJournalComment: vi.fn(),
    unlikeJournalComment: vi.fn(),
    updateJournalComment: vi.fn(),
    uploadJournalCommentMedia: vi.fn(),
}));

vi.mock("../../api/endpoints/journal", () => mocks);
vi.mock("../../api/endpoints/comments", () => commentMocks);

const journalKey = ["journal"];
const journalId = "11111111-1111-1111-1111-111111111111";
const entryId = "22222222-2222-2222-2222-222222222222";
const commentId = "33333333-3333-3333-3333-333333333333";

const journalPayload: CreateJournalPayload = { title: "the witch's diary", work: "umineko" };
const entryPayload: JournalEntryPayload = { title: "first twilight", body: "six chosen by the key", is_draft: false };

function harness() {
    const queryClient = createTestQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

    return { invalidateQueries, queryClient, wrapper: providerWrapper({ queryClient }) };
}

function makeFile() {
    return new File(["gold"], "butterfly.png", { type: "image/png" });
}

function makeDetail(overrides: Partial<JournalDetail> = {}): JournalDetail {
    return {
        id: journalId,
        title: "the witch's diary",
        work: "umineko",
        author: { id: "author-1", username: "beatrice", display_name: "Beatrice" },
        follower_count: 3,
        is_following: false,
        is_archived: false,
        is_paused: false,
        comment_count: 0,
        entry_count: 0,
        latest_entry_excerpt: "",
        created_at: "2026-01-01T00:00:00Z",
        last_author_activity_at: "2026-01-02T00:00:00Z",
        entries: [],
        comments: [],
        ...overrides,
    };
}

function cacheHarness() {
    const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
    });

    return { queryClient, wrapper: providerWrapper({ queryClient }) };
}

function seedDetail(queryClient: QueryClient, overrides: Partial<JournalDetail> = {}): void {
    queryClient.setQueryData<JournalDetail>(queryKeys.journal.detail(journalId), makeDetail(overrides));
}

function cachedDetail(queryClient: QueryClient): JournalDetail | undefined {
    return queryClient.getQueryData<JournalDetail>(queryKeys.journal.detail(journalId));
}

beforeEach(() => {
    for (const fn of [...Object.values(mocks), ...Object.values(commentMocks)]) {
        fn.mockResolvedValue(undefined);
    }
});

describe("useCreateJournal", () => {
    it("sends the payload untouched and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createJournal.mockResolvedValue({ id: journalId });
        const { result } = renderHook(() => useCreateJournal(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(journalPayload);
        });

        // then
        expect(mocks.createJournal).toHaveBeenCalledWith(journalPayload);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });

    it("hands the created id back to the caller", async () => {
        // given
        const { wrapper } = harness();
        mocks.createJournal.mockResolvedValue({ id: journalId });
        const { result } = renderHook(() => useCreateJournal(), { wrapper });

        // when
        let created: { id: string } | undefined;
        await act(async () => {
            created = await result.current.mutateAsync(journalPayload);
        });

        // then
        expect(created).toEqual({ id: journalId });
    });

    it("leaves the journal cache alone when the creation fails", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createJournal.mockRejectedValue(new Error("title is required"));
        const { result } = renderHook(() => useCreateJournal(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(journalPayload)).rejects.toThrow("title is required");
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useUpdateJournal", () => {
    it("edits the journal the hook was built for and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUpdateJournal(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(journalPayload);
        });

        // then
        expect(mocks.updateJournal).toHaveBeenCalledWith(journalId, journalPayload);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useDeleteJournal", () => {
    it("deletes the journal it was handed and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteJournal(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(journalId);
        });

        // then
        expect(mocks.deleteJournal).toHaveBeenCalledWith(journalId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useSetJournalPaused", () => {
    it("shows the journal as paused before the server has answered", async () => {
        // given
        const { wrapper, queryClient } = cacheHarness();
        seedDetail(queryClient, { is_paused: false });
        const { result } = renderHook(() => useSetJournalPaused(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ id: journalId, paused: true });
        });

        // then
        expect(mocks.setJournalPaused).toHaveBeenCalledWith(journalId, true);
        expect(cachedDetail(queryClient)?.is_paused).toBe(true);
    });

    it("puts the pause state back when the server refuses", async () => {
        // given
        const { wrapper, queryClient } = cacheHarness();
        seedDetail(queryClient, { is_paused: false });
        mocks.setJournalPaused.mockRejectedValue(new Error("nope"));
        const { result } = renderHook(() => useSetJournalPaused(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync({ id: journalId, paused: true })).rejects.toThrow("nope");
        });

        // then
        expect(cachedDetail(queryClient)?.is_paused).toBe(false);
    });
});

describe("useFollowJournal", () => {
    it("follows the journal it was handed and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useFollowJournal(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(journalId);
        });

        // then
        expect(mocks.followJournal).toHaveBeenCalledWith(journalId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });

    it("shows the journal as followed before the server has answered", async () => {
        // given
        const { wrapper, queryClient } = cacheHarness();
        seedDetail(queryClient, { is_following: false });
        const { result } = renderHook(() => useFollowJournal(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(journalId);
        });

        // then
        expect(cachedDetail(queryClient)?.is_following).toBe(true);
    });

    it("puts the follow state back when the server refuses", async () => {
        // given
        const { wrapper, queryClient } = cacheHarness();
        seedDetail(queryClient, { is_following: false });
        mocks.followJournal.mockRejectedValue(new Error("nope"));
        const { result } = renderHook(() => useFollowJournal(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(journalId)).rejects.toThrow("nope");
        });

        // then
        expect(cachedDetail(queryClient)?.is_following).toBe(false);
    });

    it("leaves a journal it has never seen alone", async () => {
        // given
        const { wrapper, queryClient } = cacheHarness();
        const { result } = renderHook(() => useFollowJournal(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(journalId);
        });

        // then
        expect(cachedDetail(queryClient)).toBeUndefined();
    });
});

describe("useUnfollowJournal", () => {
    it("unfollows the journal it was handed and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUnfollowJournal(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(journalId);
        });

        // then
        expect(mocks.unfollowJournal).toHaveBeenCalledWith(journalId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });

    it("drops the follow before the server has answered", async () => {
        // given
        const { wrapper, queryClient } = cacheHarness();
        seedDetail(queryClient, { is_following: true });
        const { result } = renderHook(() => useUnfollowJournal(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(journalId);
        });

        // then
        expect(cachedDetail(queryClient)?.is_following).toBe(false);
    });

    it("puts the follow back when the server refuses", async () => {
        // given
        const { wrapper, queryClient } = cacheHarness();
        seedDetail(queryClient, { is_following: true });
        mocks.unfollowJournal.mockRejectedValue(new Error("nope"));
        const { result } = renderHook(() => useUnfollowJournal(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(journalId)).rejects.toThrow("nope");
        });

        // then
        expect(cachedDetail(queryClient)?.is_following).toBe(true);
    });
});

describe("useCreateJournalComment", () => {
    it("posts a top level comment on the journal with no parent and no entry", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useCreateJournalComment(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "without love it cannot be seen" });
        });

        // then
        expect(mocks.createJournalComment).toHaveBeenCalledWith(
            journalId,
            "without love it cannot be seen",
            undefined,
            undefined,
        );
    });

    it("threads the comment under its parent when one is given", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useCreateJournalComment(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "a reply", parentId: commentId });
        });

        // then
        expect(mocks.createJournalComment).toHaveBeenCalledWith(journalId, "a reply", commentId, undefined);
    });

    it("attaches the comment to an entry when the hook was scoped to one", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useCreateJournalComment(journalId, entryId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "on this entry" });
        });

        // then
        expect(mocks.createJournalComment).toHaveBeenCalledWith(journalId, "on this entry", undefined, entryId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useCreateJournalEntry", () => {
    it("creates the entry under the journal the hook was built for", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createJournalEntry.mockResolvedValue({ id: entryId, entry_number: 1 });
        const { result } = renderHook(() => useCreateJournalEntry(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(entryPayload);
        });

        // then
        expect(mocks.createJournalEntry).toHaveBeenCalledWith(journalId, entryPayload);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });

    it("returns the new entry number to the caller", async () => {
        // given
        const { wrapper } = harness();
        mocks.createJournalEntry.mockResolvedValue({ id: entryId, entry_number: 7 });
        const { result } = renderHook(() => useCreateJournalEntry(journalId), { wrapper });

        // when
        let created: { id: string; entry_number: number } | undefined;
        await act(async () => {
            created = await result.current.mutateAsync(entryPayload);
        });

        // then
        expect(created).toEqual({ id: entryId, entry_number: 7 });
    });
});

describe("useUpdateJournalEntry", () => {
    it("edits the entry by its own id rather than the journal id", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUpdateJournalEntry(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ id: entryId, payload: entryPayload });
        });

        // then
        expect(mocks.updateJournalEntry).toHaveBeenCalledWith(entryId, entryPayload);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useDeleteJournalEntry", () => {
    it("deletes the entry it was handed and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteJournalEntry(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(entryId);
        });

        // then
        expect(mocks.deleteJournalEntry).toHaveBeenCalledWith(entryId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useUpdateJournalComment", () => {
    it("sends the comment id and the new body", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUpdateJournalComment(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ id: commentId, body: "edited" });
        });

        // then
        expect(commentMocks.updateJournalComment).toHaveBeenCalledWith(commentId, "edited");
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useDeleteJournalComment", () => {
    it("deletes the comment it was handed and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteJournalComment(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(commentId);
        });

        // then
        expect(commentMocks.deleteJournalComment).toHaveBeenCalledWith(commentId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useLikeJournalComment", () => {
    it("likes the comment it was handed and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useLikeJournalComment(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(commentId);
        });

        // then
        expect(commentMocks.likeJournalComment).toHaveBeenCalledWith(commentId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useUnlikeJournalComment", () => {
    it("unlikes the comment it was handed and refreshes every journal query", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUnlikeJournalComment(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(commentId);
        });

        // then
        expect(commentMocks.unlikeJournalComment).toHaveBeenCalledWith(commentId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useUploadJournalCommentMedia", () => {
    it("uploads the file against the comment it was given", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const file = makeFile();
        const { result } = renderHook(() => useUploadJournalCommentMedia(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ commentId, file });
        });

        // then
        expect(commentMocks.uploadJournalCommentMedia).toHaveBeenCalledWith(commentId, file);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useUploadJournalEntryMedia", () => {
    it("uploads the file against the entry it was given", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const file = makeFile();
        const { result } = renderHook(() => useUploadJournalEntryMedia(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ entryId, file });
        });

        // then
        expect(mocks.uploadJournalEntryMedia).toHaveBeenCalledWith(entryId, file, false);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });
});

describe("useDeleteJournalEntryMedia", () => {
    it("deletes one numeric media id from the entry it was given", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteJournalEntryMedia(journalId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ entryId, mediaId: 12 });
        });

        // then
        expect(mocks.deleteJournalEntryMedia).toHaveBeenCalledWith(entryId, 12);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: journalKey });
    });

    it("leaves the journal cache alone when the media cannot be deleted", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.deleteJournalEntryMedia.mockRejectedValue(new Error("not found"));
        const { result } = renderHook(() => useDeleteJournalEntryMedia(journalId), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync({ entryId, mediaId: 12 })).rejects.toThrow("not found");
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});
