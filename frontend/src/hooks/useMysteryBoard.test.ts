import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { dispatch } from "../api/realtime/bus";
import type { RealtimeEvent } from "../api/realtime/events";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { MysteryAttempt, MysteryDetail, UserProfile } from "../types/api";
import { useMysteryBoard } from "./useMysteryBoard";

const mocked = vi.hoisted(() => ({
    useMystery: vi.fn(),
    navigate: vi.fn(),
    addClue: vi.fn(),
    closeMystery: vi.fn(),
    createAttempt: vi.fn(),
    deleteAttachment: vi.fn(),
    deleteClue: vi.fn(),
    deleteMedia: vi.fn(),
    deleteMystery: vi.fn(),
    setGmAway: vi.fn(),
    setPaused: vi.fn(),
    updateClue: vi.fn(),
    uploadAttachment: vi.fn(),
    uploadMedia: vi.fn(),
    createComment: vi.fn(),
    deleteComment: vi.fn(),
    likeComment: vi.fn(),
    unlikeComment: vi.fn(),
    updateComment: vi.fn(),
    uploadCommentMedia: vi.fn(),
}));

vi.mock("./queries/mystery", () => ({ useMystery: mocked.useMystery }));

vi.mock("./mutations/mystery", () => ({
    useAddMysteryClue: () => ({ mutateAsync: mocked.addClue }),
    useCloseMystery: () => ({ mutateAsync: mocked.closeMystery }),
    useCreateMysteryAttempt: () => ({ mutateAsync: mocked.createAttempt }),
    useDeleteMystery: () => ({ mutateAsync: mocked.deleteMystery }),
    useDeleteMysteryAttachment: () => ({ mutateAsync: mocked.deleteAttachment }),
    useDeleteMysteryClue: () => ({ mutateAsync: mocked.deleteClue }),
    useDeleteMysteryMedia: () => ({ mutateAsync: mocked.deleteMedia }),
    useSetMysteryGmAway: () => ({ mutateAsync: mocked.setGmAway }),
    useSetMysteryPaused: () => ({ mutateAsync: mocked.setPaused }),
    useUpdateMysteryClue: () => ({ mutateAsync: mocked.updateClue }),
    useUploadMysteryAttachment: () => ({ mutateAsync: mocked.uploadAttachment }),
    useUploadMysteryMedia: () => ({ mutateAsync: mocked.uploadMedia }),
    useCreateMysteryComment: () => ({ mutateAsync: mocked.createComment }),
    useDeleteMysteryComment: () => ({ mutateAsync: mocked.deleteComment }),
    useLikeMysteryComment: () => ({ mutateAsync: mocked.likeComment }),
    useUnlikeMysteryComment: () => ({ mutateAsync: mocked.unlikeComment }),
    useUpdateMysteryComment: () => ({ mutateAsync: mocked.updateComment }),
    useUploadMysteryCommentMedia: () => ({ mutateAsync: mocked.uploadCommentMedia }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocked.navigate };
});

const gameMaster = { id: "gm-1", username: "beatrice", display_name: "Beatrice" };
const player = { id: "player-1", username: "battler", display_name: "Battler" };
const otherPlayer = { id: "player-2", username: "ange", display_name: "Ange" };

const gameMasterUser = makeUser({ id: "gm-1", username: "beatrice", display_name: "Beatrice" });
const playerUser = makeUser({ id: "player-1", username: "battler", display_name: "Battler" });

const ALL_MUTATIONS = [
    mocked.addClue,
    mocked.closeMystery,
    mocked.createAttempt,
    mocked.deleteAttachment,
    mocked.deleteClue,
    mocked.deleteMedia,
    mocked.deleteMystery,
    mocked.setGmAway,
    mocked.setPaused,
    mocked.updateClue,
    mocked.uploadAttachment,
    mocked.uploadMedia,
];

function makeAttempt(overrides: Partial<MysteryAttempt> = {}): MysteryAttempt {
    return {
        id: "attempt-1",
        author: player,
        body: "The chain was fixed after the fact.",
        is_winner: false,
        vote_score: 0,
        created_at: "2026-07-01T11:00:00Z",
        ...overrides,
    };
}

function makeMysteryDetail(overrides: Partial<MysteryDetail> = {}): MysteryDetail {
    return {
        id: "mystery-1",
        title: "The sealed guest room",
        body: "Six people died behind a chained door.",
        difficulty: "hard",
        author: gameMaster,
        solved: false,
        paused: false,
        gm_away: false,
        free_for_all: false,
        keep_open_after_solve: false,
        knox_contract: {
            culprit_named_early: true,
            no_supernatural: true,
            passages_declared: true,
            no_unknown_poison: true,
            no_outsider: true,
            no_lucky_accident: true,
            detective_not_culprit: true,
            clues_shown: true,
            narrator_hides_nothing: true,
            no_unannounced_twins: true,
        },
        knox_contract_published: true,
        knox_contract_locked: false,
        solver_count: 0,
        viewer_has_solved: false,
        paused_duration_seconds: 0,
        clues: [],
        attempts: [],
        comments: [],
        player_count: 0,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface BoardOptions {
    mystery?: MysteryDetail | null;
    loading?: boolean;
    viewer?: UserProfile | null;
    mysteryId?: string;
}

function renderBoard(options: BoardOptions = {}) {
    const refresh = vi.fn();
    mocked.useMystery.mockReturnValue({
        mystery: options.mystery === undefined ? makeMysteryDetail() : options.mystery,
        loading: options.loading ?? false,
        refresh,
    });

    const rendered = renderHook(() => useMysteryBoard(options.mysteryId ?? "mystery-1"), {
        wrapper: providerWrapper({ user: options.viewer ?? null }),
    });

    return { ...rendered, refresh };
}

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

function attemptCreated(mysteryId: string, authorId: string): RealtimeEvent {
    return {
        type: "mystery_attempt_created",
        data: {
            mystery_id: mysteryId,
            attempt_id: "attempt-9",
            parent_id: null,
            author_id: authorId,
            author_username: "ange",
            author_display_name: "Ange",
            author_avatar_url: "",
        },
    };
}

beforeEach(() => {
    for (const mutation of ALL_MUTATIONS) {
        mutation.mockResolvedValue({});
    }
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("useMysteryBoard", () => {
    it("hands the page the mystery and its loading state", () => {
        // given
        const mystery = makeMysteryDetail({ title: "The sealed guest room" });

        // when
        const { result } = renderBoard({ mystery });

        // then
        expect(result.current.mystery).toBe(mystery);
        expect(result.current.loading).toBe(false);
    });

    it("works out what the game master may do to their own mystery", () => {
        // given / when
        const { result } = renderBoard({ viewer: gameMasterUser });

        // then
        expect(result.current.permissions).toEqual({
            isAuthor: true,
            canEdit: true,
            canDelete: true,
            canSeeAsGameMaster: true,
        });
    });

    it("grants nothing while the mystery is still on its way", () => {
        // given / when
        const { result } = renderBoard({ mystery: null, loading: true, viewer: gameMasterUser });

        // then
        expect(result.current.permissions).toEqual({
            isAuthor: false,
            canEdit: false,
            canDelete: false,
            canSeeAsGameMaster: false,
        });
    });

    it("groups the attempts, pins the winner and names the crowned authors", () => {
        // given
        const mystery = makeMysteryDetail({
            solved: true,
            attempts: [
                makeAttempt({ id: "a1", is_winner: true }),
                makeAttempt({ id: "a2", author: otherPlayer }),
                makeAttempt({ id: "a3" }),
            ],
        });

        // when
        const { result } = renderBoard({ mystery, viewer: gameMasterUser });

        // then
        expect(result.current.groupedAttempts.map(group => group.author.id)).toEqual(["player-1", "player-2"]);
        expect(result.current.winningAttempt?.id).toBe("a1");
        expect(Array.from(result.current.winners)).toEqual(["player-1"]);
    });

    it("keeps only the red truths the whole board may read", () => {
        // given
        const mystery = makeMysteryDetail({
            clues: [
                { id: 1, body: "The door was chained", truth_type: "red", sort_order: 0 },
                { id: 2, body: "Only Ange may know this", truth_type: "red", sort_order: 1, player_id: "player-2" },
            ],
        });

        // when
        const { result } = renderBoard({ mystery, viewer: playerUser });

        // then
        expect(result.current.sharedClues.map(clue => clue.id)).toEqual([1]);
    });

    it("submits a trimmed blue truth and clears the composer", async () => {
        // given
        const { result } = renderBoard({ viewer: playerUser });
        act(() => {
            result.current.setAttemptBody("  The chain was faked  ");
        });

        // when
        await act(async () => {
            await result.current.submitAttempt();
        });

        // then
        expect(mocked.createAttempt).toHaveBeenCalledWith({ body: "The chain was faked" });
        expect(result.current.attemptBody).toBe("");
    });

    it("refuses to submit an empty blue truth", async () => {
        // given
        const { result } = renderBoard({ viewer: playerUser });

        // when
        await act(async () => {
            await result.current.submitAttempt();
        });

        // then
        expect(mocked.createAttempt).not.toHaveBeenCalled();
    });

    it("leaves the composer alone when the blue truth is rejected", async () => {
        // given
        mocked.createAttempt.mockRejectedValue(new Error("the board is sealed"));
        const { result } = renderBoard({ viewer: playerUser });
        act(() => {
            result.current.setAttemptBody("The chain was faked");
        });

        // when
        await act(async () => {
            await result.current.submitAttempt();
        });

        // then
        expect(result.current.attemptBody).toBe("The chain was faked");
        expect(result.current.submitting).toBe(false);
    });

    it("declares a trimmed global red truth and clears its composer", async () => {
        // given
        const { result } = renderBoard({ viewer: gameMasterUser });
        act(() => {
            result.current.setNewClueBody("  The window was latched  ");
        });

        // when
        await act(async () => {
            await result.current.addGlobalClue();
        });

        // then
        expect(mocked.addClue).toHaveBeenCalledWith({ body: "The window was latched", truthType: "red" });
        expect(result.current.newClueBody).toBe("");
    });

    it("whispers a private red truth to one player", async () => {
        // given
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        await act(async () => {
            await result.current.addPrivateClue("player-1", "Your key is a lie");
        });

        // then
        expect(mocked.addClue).toHaveBeenCalledWith({
            body: "Your key is a lie",
            truthType: "red",
            playerId: "player-1",
        });
    });

    it("asks before deleting a red truth and keeps it when waved away", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        await act(async () => {
            await result.current.removeClue(5);
        });

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this red truth? This cannot be undone.");
        expect(mocked.deleteClue).not.toHaveBeenCalled();
    });

    it("deletes the mystery once confirmed and returns to the list", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        await act(async () => {
            await result.current.removeMystery();
        });

        // then
        expect(mocked.deleteMystery).toHaveBeenCalledWith("mystery-1");
        expect(mocked.navigate).toHaveBeenCalledWith("/mysteries");
    });

    it("stays on the mystery when the deletion is waved away", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        await act(async () => {
            await result.current.removeMystery();
        });

        // then
        expect(mocked.deleteMystery).not.toHaveBeenCalled();
        expect(mocked.navigate).not.toHaveBeenCalled();
    });

    it("flips the pause and the away flag to the opposite of what stands", async () => {
        // given
        const { result } = renderBoard({ mystery: makeMysteryDetail({ paused: true }), viewer: gameMasterUser });

        // when
        await act(async () => {
            await result.current.togglePaused();
            await result.current.toggleGmAway();
        });

        // then
        expect(mocked.setPaused).toHaveBeenCalledWith(false);
        expect(mocked.setGmAway).toHaveBeenCalledWith(true);
    });

    it("asks before marking an ongoing mystery permanently solved", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        await act(async () => {
            await result.current.closeMystery();
        });

        // then
        expect(confirm).toHaveBeenCalled();
        expect(mocked.closeMystery).not.toHaveBeenCalled();
    });

    it("reports why an attachment could not be uploaded", async () => {
        // given
        mocked.uploadAttachment.mockRejectedValue(new Error("the file is cursed"));
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        await act(async () => {
            await result.current.uploadAttachment(new File(["evidence"], "notes.pdf"));
        });

        // then
        expect(result.current.attachmentError).toBe("the file is cursed");
        expect(result.current.uploadingAttachment).toBe(false);
    });

    it("ignores an attachment change that carried no file", async () => {
        // given
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        await act(async () => {
            await result.current.uploadAttachment(undefined);
        });

        // then
        expect(mocked.uploadAttachment).not.toHaveBeenCalled();
    });

    it("uploads every queued image and empties the queue", async () => {
        // given
        const { result } = renderBoard({ viewer: gameMasterUser });
        act(() => {
            result.current.addPendingMedia([new File(["a"], "a.png"), new File(["b"], "b.png")]);
        });

        // when
        await act(async () => {
            await result.current.uploadMedia();
        });

        // then
        expect(mocked.uploadMedia).toHaveBeenCalledTimes(2);
        expect(result.current.pendingMedia).toEqual([]);
    });

    it("keeps the queued images and reports why the upload failed", async () => {
        // given
        mocked.uploadMedia.mockRejectedValue(new Error("that picture is forbidden"));
        const { result } = renderBoard({ viewer: gameMasterUser });
        act(() => {
            result.current.addPendingMedia([new File(["a"], "a.png")]);
        });

        // when
        await act(async () => {
            await result.current.uploadMedia();
        });

        // then
        expect(result.current.mediaError).toBe("that picture is forbidden");
        expect(result.current.pendingMedia).toHaveLength(1);
    });

    it("drops one queued image without disturbing the rest", () => {
        // given
        const { result } = renderBoard({ viewer: gameMasterUser });
        act(() => {
            result.current.addPendingMedia([new File(["a"], "a.png"), new File(["b"], "b.png")]);
        });

        // when
        act(() => {
            result.current.removePendingMedia(0);
        });

        // then
        expect(result.current.pendingMedia.map(file => file.name)).toEqual(["b.png"]);
    });

    it("folds a player's group away and opens it again", () => {
        // given
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        act(() => {
            result.current.togglePlayerCollapse("player-1");
        });

        // then
        expect(result.current.collapsedPlayers.has("player-1")).toBe(true);
        act(() => {
            result.current.togglePlayerCollapse("player-1");
        });
        expect(result.current.collapsedPlayers.has("player-1")).toBe(false);
    });

    it("opens a folded group and marks it read when the pill is followed", () => {
        // given
        localStorage.setItem("mystery-read-cursor-mystery-1", "2026-07-01T10:00:00Z");
        const mystery = makeMysteryDetail({
            attempts: [makeAttempt({ created_at: "2026-07-01T11:00:00Z" })],
        });
        const { result } = renderBoard({ mystery, viewer: gameMasterUser });
        act(() => {
            result.current.togglePlayerCollapse("player-1");
        });
        expect(result.current.unreadPlayers.has("player-1")).toBe(true);

        // when
        act(() => {
            result.current.jumpToPlayer("player-1");
        });

        // then
        expect(result.current.unreadPlayers.has("player-1")).toBe(false);
        expect(result.current.collapsedPlayers.has("player-1")).toBe(false);
    });

    it("lays down a read cursor the first time a game master opens an unsolved mystery", () => {
        // given
        const mystery = makeMysteryDetail({ attempts: [makeAttempt()] });

        // when
        renderBoard({ mystery, viewer: gameMasterUser });

        // then
        expect(localStorage.getItem("mystery-read-cursor-mystery-1")).not.toBeNull();
    });

    it("marks nobody unread on that first visit", () => {
        // given
        const mystery = makeMysteryDetail({ attempts: [makeAttempt()] });

        // when
        const { result } = renderBoard({ mystery, viewer: gameMasterUser });

        // then
        expect(result.current.unreadPlayers.size).toBe(0);
    });

    it("marks the players who moved since the cursor as unread", () => {
        // given
        localStorage.setItem("mystery-read-cursor-mystery-1", "2026-07-01T10:30:00Z");
        const mystery = makeMysteryDetail({
            attempts: [
                makeAttempt({ id: "a1", created_at: "2026-07-01T10:00:00Z" }),
                makeAttempt({ id: "a2", author: otherPlayer, created_at: "2026-07-01T11:00:00Z" }),
            ],
        });

        // when
        const { result } = renderBoard({ mystery, viewer: gameMasterUser });

        // then
        expect(Array.from(result.current.unreadPlayers)).toEqual(["player-2"]);
    });

    it("never works the unread pills out for an ordinary piece", () => {
        // given
        localStorage.setItem("mystery-read-cursor-mystery-1", "2026-07-01T10:30:00Z");
        const mystery = makeMysteryDetail({
            attempts: [makeAttempt({ author: otherPlayer, created_at: "2026-07-01T11:00:00Z" })],
        });

        // when
        const { result } = renderBoard({ mystery, viewer: playerUser });

        // then
        expect(result.current.unreadPlayers.size).toBe(0);
    });

    it("never works the unread pills out once the mystery is solved", () => {
        // given
        localStorage.setItem("mystery-read-cursor-mystery-1", "2026-07-01T10:30:00Z");
        const mystery = makeMysteryDetail({
            solved: true,
            attempts: [makeAttempt({ author: otherPlayer, created_at: "2026-07-01T11:00:00Z" })],
        });

        // when
        const { result } = renderBoard({ mystery, viewer: gameMasterUser });

        // then
        expect(result.current.unreadPlayers.size).toBe(0);
    });

    it("moves the read cursor forward when the board is left", () => {
        // given
        localStorage.setItem("mystery-read-cursor-mystery-1", "2026-07-01T10:30:00Z");
        const { unmount } = renderBoard({ viewer: gameMasterUser });

        // when
        unmount();

        // then
        expect(localStorage.getItem("mystery-read-cursor-mystery-1")).not.toBe("2026-07-01T10:30:00Z");
    });

    it("refetches the board when another piece declares a blue truth", async () => {
        // given
        const { refresh } = renderBoard({ viewer: gameMasterUser });

        // when
        emit(attemptCreated("mystery-1", "player-2"));

        // then
        await waitFor(() => {
            expect(refresh).toHaveBeenCalledTimes(1);
        });
    });

    it("raises an unread pill for the piece who moved", () => {
        // given
        const { result } = renderBoard({ viewer: gameMasterUser });

        // when
        emit(attemptCreated("mystery-1", "player-2"));

        // then
        expect(result.current.unreadPlayers.has("player-2")).toBe(true);
    });

    it("raises the pill again when a piece who was caught up on moves once more", () => {
        // given
        const { result } = renderBoard({ viewer: gameMasterUser });
        emit(attemptCreated("mystery-1", "player-2"));
        act(() => {
            result.current.jumpToPlayer("player-2");
        });
        expect(result.current.unreadPlayers.has("player-2")).toBe(false);

        // when
        emit(attemptCreated("mystery-1", "player-2"));

        // then
        expect(result.current.unreadPlayers.has("player-2")).toBe(true);
    });

    it("keeps a piece read once the pill has been followed, even after the board is refetched", () => {
        // given
        localStorage.setItem("mystery-read-cursor-mystery-1", "2026-07-01T10:00:00Z");
        const mystery = makeMysteryDetail({
            attempts: [makeAttempt({ created_at: "2026-07-01T11:00:00Z" })],
        });
        const { result, rerender } = renderBoard({ mystery, viewer: gameMasterUser });
        act(() => {
            result.current.jumpToPlayer("player-1");
        });

        // when
        rerender();

        // then
        expect(result.current.unreadPlayers.has("player-1")).toBe(false);
    });

    it("never raises an unread pill against the reader's own attempt", () => {
        // given
        const { result } = renderBoard({ viewer: playerUser });

        // when
        emit(attemptCreated("mystery-1", "player-1"));

        // then
        expect(result.current.unreadPlayers.size).toBe(0);
    });

    it("ignores an attempt declared on some other mystery", () => {
        // given
        const { result, refresh } = renderBoard({ viewer: gameMasterUser });

        // when
        emit(attemptCreated("mystery-2", "player-2"));

        // then
        expect(refresh).not.toHaveBeenCalled();
        expect(result.current.unreadPlayers.size).toBe(0);
    });

    it("refetches the board when the mystery is solved and jumps to the winning attempt", async () => {
        // given
        const { refresh } = renderBoard({ viewer: playerUser });
        const frame = vi.spyOn(window, "requestAnimationFrame").mockImplementation(callback => {
            callback(0);
            return 0;
        });
        const scrollIntoView = vi.fn();
        const winner = document.createElement("div");
        winner.id = "attempt-a1";
        winner.scrollIntoView = scrollIntoView;
        document.body.appendChild(winner);

        // when
        emit({ type: "mystery_solved", data: { mystery_id: "mystery-1", attempt_id: "a1" } });

        // then
        await waitFor(() => {
            expect(refresh).toHaveBeenCalledTimes(1);
        });
        expect(scrollIntoView).toHaveBeenCalled();
        frame.mockRestore();
        winner.remove();
    });

    it("refetches the board on every other move the game master makes elsewhere", async () => {
        // given
        const { refresh } = renderBoard({ viewer: playerUser });

        // when
        emit({ type: "mystery_winner_added", data: { mystery_id: "mystery-1", winner_id: "player-2" } });

        // then
        await waitFor(() => {
            expect(refresh).toHaveBeenCalledTimes(1);
        });
        emit({ type: "mystery_clue_added", data: { mystery_id: "mystery-1", truth_type: "red" } });
        emit({ type: "mystery_clue_updated", data: { mystery_id: "mystery-1" } });
        emit({ type: "mystery_paused", data: { mystery_id: "mystery-1", paused: true } });
        emit({ type: "mystery_gm_away", data: { mystery_id: "mystery-1", gm_away: true } });
        await waitFor(() => {
            expect(refresh.mock.calls.length).toBeGreaterThan(1);
        });
    });

    it("leaves the refetching to the mutations rather than asking twice", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result, refresh } = renderBoard({ viewer: gameMasterUser });
        act(() => {
            result.current.setAttemptBody("The chain was faked");
            result.current.setNewClueBody("The window was latched");
        });

        // when
        await act(async () => {
            await result.current.submitAttempt();
            await result.current.addGlobalClue();
            await result.current.togglePaused();
            await result.current.toggleGmAway();
            await result.current.closeMystery();
            await result.current.addPrivateClue("player-1", "Your key is a lie");
            await result.current.saveClue(5, "amended");
            await result.current.removeClue(5);
            await result.current.uploadAttachment(new File(["a"], "a.pdf"));
            await result.current.removeAttachment({
                id: 3,
                file_url: "/files/notes.pdf",
                file_name: "notes.pdf",
                file_size: 512,
            });
            await result.current.removeMedia(11);
        });

        // then
        expect(mocked.createAttempt).toHaveBeenCalled();
        expect(mocked.closeMystery).toHaveBeenCalled();
        expect(refresh).not.toHaveBeenCalled();
    });

    it("hands the comment section an adapter for every operation", () => {
        // given / when
        const { result } = renderBoard({ viewer: playerUser });

        // then
        expect(Object.values(result.current.comments).every(handler => typeof handler === "function")).toBe(true);
    });

    it("routes a new comment through the mystery comment mutation", async () => {
        // given
        mocked.createComment.mockResolvedValue({ id: "comment-1" });
        const { result } = renderBoard({ viewer: playerUser });

        // when
        const created = await result.current.comments.createCommentFn("mystery-1", "A fine game", "parent-1");

        // then
        expect(mocked.createComment).toHaveBeenCalledWith({ body: "A fine game", parentId: "parent-1" });
        expect(created).toEqual({ id: "comment-1" });
    });
});
