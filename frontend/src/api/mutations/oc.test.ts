import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import * as endpoints from "../endpoints";
import { queryKeys } from "../queryKeys";
import {
    useAddOCGalleryImage,
    useCreateOC,
    useCreateOCComment,
    useDeleteOC,
    useDeleteOCComment,
    useDeleteOCGalleryImage,
    useFavouriteOC,
    useLikeOCComment,
    useUnlikeOCComment,
    useUpdateOC,
    useUpdateOCComment,
    useUploadOCCommentMedia,
    useUploadOCImageById,
    useVoteOC,
} from "./oc";

vi.mock("../endpoints", () => ({
    addOCGalleryImage: vi.fn(),
    createOC: vi.fn(),
    createOCComment: vi.fn(),
    deleteOC: vi.fn(),
    deleteOCComment: vi.fn(),
    deleteOCGalleryImage: vi.fn(),
    favouriteOC: vi.fn(),
    likeOCComment: vi.fn(),
    unlikeOCComment: vi.fn(),
    updateOC: vi.fn(),
    updateOCComment: vi.fn(),
    uploadOCCommentMedia: vi.fn(),
    uploadOCImage: vi.fn(),
    voteOC: vi.fn(),
}));

interface MutationCase {
    name: string;
    useHook: () => { mutateAsync: (variables: never) => Promise<unknown> };
    variables: unknown;
    endpoint: Mock;
    args: unknown[];
}

const ocId = "oc-1";
const file = new File(["ink"], "portrait.png", { type: "image/png" });

const draft = {
    name: "Sakutarou",
    description: "A favourite lion",
    series: "umineko",
    custom_series_name: "",
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

describe("oc mutations", () => {
    const cases: MutationCase[] = [
        {
            name: "useCreateOC posts the whole draft",
            useHook: () => useCreateOC(),
            variables: draft,
            endpoint: vi.mocked(endpoints.createOC),
            args: [draft],
        },
        {
            name: "useUpdateOC puts the draft against the id it was built with",
            useHook: () => useUpdateOC(ocId),
            variables: draft,
            endpoint: vi.mocked(endpoints.updateOC),
            args: [ocId, draft],
        },
        {
            name: "useDeleteOC deletes the id it is handed",
            useHook: () => useDeleteOC(),
            variables: ocId,
            endpoint: vi.mocked(endpoints.deleteOC),
            args: [ocId],
        },
        {
            name: "useUploadOCImageById sends the portrait against the id in its variables",
            useHook: () => useUploadOCImageById(),
            variables: { id: ocId, file },
            endpoint: vi.mocked(endpoints.uploadOCImage),
            args: [ocId, file],
        },
        {
            name: "useAddOCGalleryImage sends the file together with its caption",
            useHook: () => useAddOCGalleryImage(),
            variables: { id: ocId, file, caption: "a favourite pose" },
            endpoint: vi.mocked(endpoints.addOCGalleryImage),
            args: [ocId, file, "a favourite pose"],
        },
        {
            name: "useDeleteOCGalleryImage names both the character and the numbered image",
            useHook: () => useDeleteOCGalleryImage(),
            variables: { ocId, imageId: 8 },
            endpoint: vi.mocked(endpoints.deleteOCGalleryImage),
            args: [ocId, 8],
        },
        {
            name: "useVoteOC sends the vote value against the id it was built with",
            useHook: () => useVoteOC(ocId),
            variables: 1,
            endpoint: vi.mocked(endpoints.voteOC),
            args: [ocId, 1],
        },
        {
            name: "useFavouriteOC toggles the favourite on the id it is handed",
            useHook: () => useFavouriteOC(),
            variables: ocId,
            endpoint: vi.mocked(endpoints.favouriteOC),
            args: [ocId],
        },
        {
            name: "useCreateOCComment sends the body and the parent under the character it was built with",
            useHook: () => useCreateOCComment(ocId),
            variables: { body: "Sakutarou is the best", parentId: "c-parent" },
            endpoint: vi.mocked(endpoints.createOCComment),
            args: [ocId, "Sakutarou is the best", "c-parent"],
        },
        {
            name: "useUpdateOCComment addresses the comment directly",
            useHook: () => useUpdateOCComment(),
            variables: { id: "c-1", body: "Rewritten" },
            endpoint: vi.mocked(endpoints.updateOCComment),
            args: ["c-1", "Rewritten"],
        },
        {
            name: "useDeleteOCComment addresses the comment directly",
            useHook: () => useDeleteOCComment(),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.deleteOCComment),
            args: ["c-1"],
        },
        {
            name: "useLikeOCComment likes the comment id it is handed",
            useHook: () => useLikeOCComment(),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.likeOCComment),
            args: ["c-1"],
        },
        {
            name: "useUnlikeOCComment unlikes the comment id it is handed",
            useHook: () => useUnlikeOCComment(),
            variables: "c-1",
            endpoint: vi.mocked(endpoints.unlikeOCComment),
            args: ["c-1"],
        },
        {
            name: "useUploadOCCommentMedia attaches the file to the given comment",
            useHook: () => useUploadOCCommentMedia(),
            variables: { commentId: "c-1", file },
            endpoint: vi.mocked(endpoints.uploadOCCommentMedia),
            args: ["c-1", file],
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
        expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.oc.all });
        expect(invalidate).toHaveBeenCalledTimes(1);
    });
});

describe("useCreateOC", () => {
    it("hands the id of the freshly created character back to the caller", async () => {
        // given
        vi.mocked(endpoints.createOC).mockResolvedValue({ id: "oc-new" });
        const { result } = setup(() => useCreateOC());

        // when
        let created: { id: string } | undefined;
        await act(async () => {
            created = await result.current.mutateAsync(draft);
        });

        // then
        expect(created).toEqual({ id: "oc-new" });
    });

    it("leaves the character cache alone when the creation is rejected", async () => {
        // given
        vi.mocked(endpoints.createOC).mockRejectedValue(new Error("name already taken"));
        const { result, invalidate } = setup(() => useCreateOC());

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync(draft);
        });

        // then
        await expect(attempt).rejects.toThrow("name already taken");
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useCreateOCComment", () => {
    it("sends an undefined parent when the comment is not a reply", async () => {
        // given
        const { result } = setup(() => useCreateOCComment(ocId));

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "Adorable" });
        });

        // then
        expect(endpoints.createOCComment).toHaveBeenCalledWith(ocId, "Adorable", undefined);
    });
});

describe("useFavouriteOC", () => {
    it("hands the new favourite state back to the caller", async () => {
        // given
        vi.mocked(endpoints.favouriteOC).mockResolvedValue({ favourited: true });
        const { result } = setup(() => useFavouriteOC());

        // when
        let outcome: { favourited: boolean } | undefined;
        await act(async () => {
            outcome = await result.current.mutateAsync(ocId);
        });

        // then
        expect(outcome).toEqual({ favourited: true });
    });
});

describe("useUpdateOC", () => {
    it("keeps using the id it was built with even when the draft carries a different name", async () => {
        // given
        const { result } = setup(() => useUpdateOC("oc-fixed"));

        // when
        await act(async () => {
            await result.current.mutateAsync({ ...draft, name: "Renamed" });
        });

        // then
        expect(endpoints.updateOC).toHaveBeenCalledWith("oc-fixed", { ...draft, name: "Renamed" });
    });
});
