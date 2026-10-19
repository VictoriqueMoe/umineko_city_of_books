import { beforeEach, describe, expect, it, vi } from "vitest";
import { appendMediaParts, createFormData, mediaFormData, postCreate } from "./mediaFormData";
import { lastFormData, postFormDataMock, postMock, resetTransports } from "./testHarness";

vi.mock("../client", async importOriginal => {
    const actual = await importOriginal<typeof import("../client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiFetchText: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

beforeEach(resetTransports);

function image(name: string): File {
    return new File(["x"], name, { type: "image/png" });
}

describe("mediaFormData", () => {
    it("carries one file and flags it only when it is a spoiler", () => {
        // given
        const file = image("one.png");

        // when
        const plain = mediaFormData(file, false);
        const spoiler = mediaFormData(file, true);

        // then
        expect(plain.get("media")).toBe(file);
        expect(plain.has("is_spoiler")).toBe(false);
        expect(spoiler.get("is_spoiler")).toBe("true");
    });
});

describe("appendMediaParts", () => {
    const cases = [
        {
            name: "adds nothing when there are no files",
            files: [] as File[],
            spoilers: [] as boolean[],
            spoilerIndexes: null,
        },
        {
            name: "adds every file and no spoiler indexes when none is a spoiler",
            files: [image("a.png"), image("b.png")],
            spoilers: [false, false],
            spoilerIndexes: null,
        },
        {
            name: "lists the zero based index of every spoilered file",
            files: [image("a.png"), image("b.png"), image("c.png")],
            spoilers: [true, false, true],
            spoilerIndexes: "0,2",
        },
    ];

    it.each(cases)("$name", ({ files, spoilers, spoilerIndexes }) => {
        // given
        const formData = new FormData();

        // when
        appendMediaParts(formData, files, spoilers);

        // then
        expect(formData.getAll("media")).toEqual(files);
        expect(formData.get("spoiler_indexes")).toBe(spoilerIndexes);
    });
});

describe("createFormData", () => {
    it("puts the JSON payload, the media in display order, the spoiler indexes and the attachments in one body", () => {
        // given
        const first = image("first.png");
        const second = image("second.png");
        const notes = new File(["n"], "notes.txt", { type: "text/plain" });
        const payload = { body: "look", parent_id: "c-1" };

        // when
        const formData = createFormData(payload, {
            media: [
                { file: first, isSpoiler: false },
                { file: second, isSpoiler: true },
            ],
            attachments: [notes],
        });

        // then
        expect(JSON.parse(formData.get("payload") as string)).toEqual(payload);
        expect(formData.getAll("media")).toEqual([first, second]);
        expect(formData.get("spoiler_indexes")).toBe("1");
        expect(formData.getAll("attachments")).toEqual([notes]);
    });

    it("leaves fields the payload does not set out of the JSON, as the JSON body would", () => {
        // when
        const formData = createFormData({ body: "", parent_id: undefined }, { media: [] });

        // then
        expect(formData.get("payload")).toBe('{"body":""}');
        expect(formData.has("media")).toBe(false);
        expect(formData.has("attachments")).toBe(false);
    });
});

describe("postCreate", () => {
    it("sends the plain JSON body when there are no files", async () => {
        // given
        const payload = { body: "words only" };

        // when
        await postCreate("/posts/p-1/comments", payload);

        // then
        expect(postMock).toHaveBeenCalledWith("/posts/p-1/comments", payload);
        expect(postFormDataMock).not.toHaveBeenCalled();
    });

    it("still sends the plain JSON body when the upload lists are empty", async () => {
        // given
        const payload = { body: "words only" };

        // when
        await postCreate("/posts/p-1/comments", payload, { media: [], attachments: [] });

        // then
        expect(postMock).toHaveBeenCalledWith("/posts/p-1/comments", payload);
        expect(postFormDataMock).not.toHaveBeenCalled();
    });

    it("sends one multipart request when there is media", async () => {
        // given
        const file = image("one.png");

        // when
        await postCreate("/posts/p-1/comments", { body: "" }, { media: [{ file, isSpoiler: true }] });

        // then
        expect(postMock).not.toHaveBeenCalled();
        expect(postFormDataMock).toHaveBeenCalledOnce();
        expect(postFormDataMock.mock.calls[0][0]).toBe("/posts/p-1/comments");
        expect(lastFormData().get("payload")).toBe('{"body":""}');
        expect(lastFormData().getAll("media")).toEqual([file]);
        expect(lastFormData().get("spoiler_indexes")).toBe("0");
    });

    it("sends one multipart request when there are only attachments", async () => {
        // given
        const notes = new File(["n"], "notes.txt", { type: "text/plain" });

        // when
        await postCreate("/mysteries", { title: "t" }, { attachments: [notes] });

        // then
        expect(postMock).not.toHaveBeenCalled();
        expect(lastFormData().getAll("attachments")).toEqual([notes]);
        expect(lastFormData().has("media")).toBe(false);
    });

    it("hands back what the server returned for the multipart request", async () => {
        // given
        postFormDataMock.mockResolvedValue({ id: "c-9" });

        // when
        const created = await postCreate<{ id: string }, { body: string }>(
            "/posts/p-1/comments",
            { body: "" },
            { media: [{ file: image("one.png"), isSpoiler: false }] },
        );

        // then
        expect(created).toEqual({ id: "c-9" });
    });

    it("lets a refused multipart request reject so nothing is reported as sent", async () => {
        // given
        postFormDataMock.mockRejectedValue(new Error("storage is unavailable"));

        // when
        const attempt = postCreate(
            "/posts/p-1/comments",
            { body: "" },
            { media: [{ file: image("one.png"), isSpoiler: false }] },
        );

        // then
        await expect(attempt).rejects.toThrow("storage is unavailable");
    });
});
