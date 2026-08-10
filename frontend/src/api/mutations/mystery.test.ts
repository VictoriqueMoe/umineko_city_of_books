import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import * as endpoints from "../endpoints";
import { queryKeys } from "../queryKeys";
import {
    useAddMysteryClue,
    useCloseMystery,
    useCreateMystery,
    useCreateMysteryAttempt,
    useCreateMysteryComment,
    useDeleteMystery,
    useDeleteMysteryAttachment,
    useDeleteMysteryAttempt,
    useDeleteMysteryClue,
    useDeleteMysteryComment,
    useDeleteMysteryMedia,
    useLikeMysteryComment,
    useMarkMysterySolved,
    useSetMysteryGmAway,
    useSetMysteryPaused,
    useUnlikeMysteryComment,
    useUpdateMystery,
    useUpdateMysteryClue,
    useUpdateMysteryComment,
    useUploadMysteryAttachment,
    useUploadMysteryAttachmentToAny,
    useUploadMysteryCommentMedia,
    useUploadMysteryMedia,
    useUploadMysteryMediaToAny,
    useVoteMysteryAttempt,
} from "./mystery";

vi.mock("../endpoints", () => ({
    addMysteryClue: vi.fn(),
    closeMystery: vi.fn(),
    createMystery: vi.fn(),
    createMysteryAttempt: vi.fn(),
    createMysteryComment: vi.fn(),
    deleteMystery: vi.fn(),
    deleteMysteryAttachment: vi.fn(),
    deleteMysteryAttempt: vi.fn(),
    deleteMysteryClue: vi.fn(),
    deleteMysteryComment: vi.fn(),
    deleteMysteryMedia: vi.fn(),
    likeMysteryComment: vi.fn(),
    markMysterySolved: vi.fn(),
    setMysteryGmAway: vi.fn(),
    setMysteryPaused: vi.fn(),
    unlikeMysteryComment: vi.fn(),
    updateMystery: vi.fn(),
    updateMysteryClue: vi.fn(),
    updateMysteryComment: vi.fn(),
    uploadMysteryAttachment: vi.fn(),
    uploadMysteryCommentMedia: vi.fn(),
    uploadMysteryMedia: vi.fn(),
    voteMysteryAttempt: vi.fn(),
}));

interface MutationCase {
    name: string;
    useHook: () => { mutateAsync: (variables: never) => Promise<unknown> };
    variables: unknown;
    endpoint: Mock;
    args: unknown[];
}

const mysteryId = "m-1";
const file = new File(["gold"], "clue.png", { type: "image/png" });

const draft = {
    title: "The first twilight",
    body: "Six chosen by the key",
    difficulty: "hard",
    free_for_all: false,
    keep_open_after_solve: true,
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
    clues: [{ body: "The chapel door", truth_type: "red" }],
};

function setup<T>(useHook: () => T) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(useHook, { wrapper: providerWrapper({ queryClient }) });

    return { result, invalidate };
}

beforeEach(() => {
    for (const value of Object.values(endpoints)) {
        if (vi.isMockFunction(value)) {
            value.mockReset();
        }
    }
});

describe("mystery mutations", () => {
    const cases: MutationCase[] = [
        {
            name: "useCreateMystery posts the whole draft",
            useHook: () => useCreateMystery(),
            variables: draft,
            endpoint: vi.mocked(endpoints.createMystery),
            args: [draft],
        },
        {
            name: "useUpdateMystery puts the draft against the id it was built with",
            useHook: () => useUpdateMystery(mysteryId),
            variables: draft,
            endpoint: vi.mocked(endpoints.updateMystery),
            args: [mysteryId, draft],
        },
        {
            name: "useDeleteMystery deletes the id it is handed",
            useHook: () => useDeleteMystery(),
            variables: mysteryId,
            endpoint: vi.mocked(endpoints.deleteMystery),
            args: [mysteryId],
        },
        {
            name: "useCreateMysteryAttempt sends the body and the parent under the mystery it was built with",
            useHook: () => useCreateMysteryAttempt(mysteryId),
            variables: { body: "Kanon did it", parentId: "a-parent" },
            endpoint: vi.mocked(endpoints.createMysteryAttempt),
            args: [mysteryId, "Kanon did it", "a-parent"],
        },
        {
            name: "useDeleteMysteryAttempt addresses the attempt directly and ignores the mystery it was built with",
            useHook: () => useDeleteMysteryAttempt(mysteryId),
            variables: "a-1",
            endpoint: vi.mocked(endpoints.deleteMysteryAttempt),
            args: ["a-1"],
        },
        {
            name: "useVoteMysteryAttempt sends the attempt id with the vote value",
            useHook: () => useVoteMysteryAttempt(mysteryId),
            variables: { id: "a-1", value: -1 },
            endpoint: vi.mocked(endpoints.voteMysteryAttempt),
            args: ["a-1", -1],
        },
        {
            name: "useMarkMysterySolved names the winning attempt on its own mystery",
            useHook: () => useMarkMysterySolved(mysteryId),
            variables: "a-9",
            endpoint: vi.mocked(endpoints.markMysterySolved),
            args: [mysteryId, "a-9"],
        },
        {
            name: "useCloseMystery closes its own mystery without any variables",
            useHook: () => useCloseMystery(mysteryId),
            variables: undefined,
            endpoint: vi.mocked(endpoints.closeMystery),
            args: [mysteryId],
        },
        {
            name: "useSetMysteryPaused forwards the paused flag",
            useHook: () => useSetMysteryPaused(mysteryId),
            variables: true,
            endpoint: vi.mocked(endpoints.setMysteryPaused),
            args: [mysteryId, true],
        },
        {
            name: "useSetMysteryGmAway forwards the away flag",
            useHook: () => useSetMysteryGmAway(mysteryId),
            variables: false,
            endpoint: vi.mocked(endpoints.setMysteryGmAway),
            args: [mysteryId, false],
        },
        {
            name: "useDeleteMysteryClue deletes a numbered clue from its own mystery",
            useHook: () => useDeleteMysteryClue(mysteryId),
            variables: 3,
            endpoint: vi.mocked(endpoints.deleteMysteryClue),
            args: [mysteryId, 3],
        },
        {
            name: "useUpdateMysteryClue rewrites the body of a numbered clue",
            useHook: () => useUpdateMysteryClue(mysteryId),
            variables: { clueId: 4, body: "Without red there is no truth" },
            endpoint: vi.mocked(endpoints.updateMysteryClue),
            args: [mysteryId, 4, "Without red there is no truth"],
        },
        {
            name: "useAddMysteryClue sends the body, the truth type and the addressed player",
            useHook: () => useAddMysteryClue(mysteryId),
            variables: { body: "The door was locked", truthType: "red", playerId: "u-7" },
            endpoint: vi.mocked(endpoints.addMysteryClue),
            args: [mysteryId, "The door was locked", "red", "u-7"],
        },
        {
            name: "useCreateMysteryComment sends the body and the parent under its own mystery",
            useHook: () => useCreateMysteryComment(mysteryId),
            variables: { body: "A fine theory", parentId: "c-parent" },
            endpoint: vi.mocked(endpoints.createMysteryComment),
            args: [mysteryId, "A fine theory", "c-parent"],
        },
        {
            name: "useUpdateMysteryComment addresses the comment directly",
            useHook: () => useUpdateMysteryComment(mysteryId),
            variables: { id: "c-1", body: "Rewritten" },
            endpoint: vi.mocked(endpoints.updateMysteryComment),
            args: ["c-1", "Rewritten"],
        },
        {
            name: "useDeleteMysteryComment addresses the comment directly",
            useHook: () => useDeleteMysteryComment(mysteryId),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.deleteMysteryComment),
            args: ["c-1"],
        },
        {
            name: "useLikeMysteryComment likes the comment id it is handed",
            useHook: () => useLikeMysteryComment(mysteryId),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.likeMysteryComment),
            args: ["c-1"],
        },
        {
            name: "useUnlikeMysteryComment unlikes the comment id it is handed",
            useHook: () => useUnlikeMysteryComment(mysteryId),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.unlikeMysteryComment),
            args: ["c-1"],
        },
        {
            name: "useUploadMysteryCommentMedia attaches the file to the given comment",
            useHook: () => useUploadMysteryCommentMedia(mysteryId),
            variables: { commentId: "c-1", file },
            endpoint: vi.mocked(endpoints.uploadMysteryCommentMedia),
            args: ["c-1", file],
        },
        {
            name: "useUploadMysteryAttachment attaches the file to its own mystery",
            useHook: () => useUploadMysteryAttachment(mysteryId),
            variables: file,
            endpoint: vi.mocked(endpoints.uploadMysteryAttachment),
            args: [mysteryId, file],
        },
        {
            name: "useUploadMysteryAttachmentToAny takes the mystery id from the variables instead",
            useHook: () => useUploadMysteryAttachmentToAny(),
            variables: { mysteryId: "m-other", file },
            endpoint: vi.mocked(endpoints.uploadMysteryAttachment),
            args: ["m-other", file],
        },
        {
            name: "useDeleteMysteryAttachment deletes a numbered attachment from its own mystery",
            useHook: () => useDeleteMysteryAttachment(mysteryId),
            variables: 12,
            endpoint: vi.mocked(endpoints.deleteMysteryAttachment),
            args: [mysteryId, 12],
        },
        {
            name: "useUploadMysteryMedia uploads the file to its own mystery",
            useHook: () => useUploadMysteryMedia(mysteryId),
            variables: file,
            endpoint: vi.mocked(endpoints.uploadMysteryMedia),
            args: [mysteryId, file],
        },
        {
            name: "useUploadMysteryMediaToAny takes the mystery id from the variables instead",
            useHook: () => useUploadMysteryMediaToAny(),
            variables: { mysteryId: "m-other", file },
            endpoint: vi.mocked(endpoints.uploadMysteryMedia),
            args: ["m-other", file],
        },
        {
            name: "useDeleteMysteryMedia deletes a numbered media item from its own mystery",
            useHook: () => useDeleteMysteryMedia(mysteryId),
            variables: 5,
            endpoint: vi.mocked(endpoints.deleteMysteryMedia),
            args: [mysteryId, 5],
        },
    ];

    it.each(cases)("$name", async ({ useHook, variables, endpoint, args }) => {
        // given the hook, its variables and the endpoint call they should produce, from the table row
        const { result, invalidate } = setup(useHook);

        // when
        await act(async () => {
            await result.current.mutateAsync(variables as never);
        });

        // then
        expect(endpoint).toHaveBeenCalledWith(...args);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.mystery.all });
        expect(invalidate).toHaveBeenCalledTimes(1);
    });
});

describe("useCreateMystery", () => {
    it("hands the id of the freshly created mystery back to the caller", async () => {
        // given
        vi.mocked(endpoints.createMystery).mockResolvedValue({ id: "m-new" });
        const { result } = setup(() => useCreateMystery());

        // when
        let created: { id: string } | undefined;
        await act(async () => {
            created = await result.current.mutateAsync(draft);
        });

        // then
        expect(created).toEqual({ id: "m-new" });
    });

    it("leaves the mystery cache alone when the creation is rejected", async () => {
        // given
        vi.mocked(endpoints.createMystery).mockRejectedValue(new Error("not a game master"));
        const { result, invalidate } = setup(() => useCreateMystery());

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync(draft);
        });

        // then
        await expect(attempt).rejects.toThrow("not a game master");
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useCreateMysteryAttempt", () => {
    it("sends an undefined parent when the attempt is not a reply", async () => {
        // given
        const { result } = setup(() => useCreateMysteryAttempt(mysteryId));

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "The window was open" });
        });

        // then
        expect(endpoints.createMysteryAttempt).toHaveBeenCalledWith(mysteryId, "The window was open", undefined);
    });
});

describe("useCreateMysteryComment", () => {
    it("sends an undefined parent when the comment is not a reply", async () => {
        // given
        const { result } = setup(() => useCreateMysteryComment(mysteryId));

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "Nice one" });
        });

        // then
        expect(endpoints.createMysteryComment).toHaveBeenCalledWith(mysteryId, "Nice one", undefined);
    });
});

describe("useAddMysteryClue", () => {
    it("sends an undefined player when the clue is addressed to the whole table", async () => {
        // given
        const { result } = setup(() => useAddMysteryClue(mysteryId));

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "Nobody left the room", truthType: "blue" });
        });

        // then
        expect(endpoints.addMysteryClue).toHaveBeenCalledWith(mysteryId, "Nobody left the room", "blue", undefined);
    });
});
