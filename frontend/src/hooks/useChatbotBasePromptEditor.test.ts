import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useChatbotBasePromptEditor } from "./useChatbotBasePromptEditor";
import type { ChatbotBasePrompt } from "../types/api";

const mocks = vi.hoisted(() => ({
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    creating: false,
}));

vi.mock("./mutations/admin", () => ({
    useCreateChatbotBasePrompt: () => ({ mutateAsync: mocks.create, isPending: mocks.creating }),
    useUpdateChatbotBasePrompt: () => ({ mutateAsync: mocks.update, isPending: false }),
    useDeleteChatbotBasePrompt: () => ({ mutateAsync: mocks.remove, isPending: false }),
}));

function makeBasePrompt(overrides: Partial<ChatbotBasePrompt> = {}): ChatbotBasePrompt {
    return {
        id: "base-1",
        name: "game witch",
        prompt: "You are a witch of the game boards.",
        bot_count: 0,
        created_at: "2026-08-11T00:00:00Z",
        updated_at: "2026-08-11T00:00:00Z",
        ...overrides,
    };
}

beforeEach(() => {
    mocks.creating = false;
    mocks.create.mockResolvedValue(undefined);
    mocks.update.mockResolvedValue(undefined);
    mocks.remove.mockResolvedValue(undefined);
});

describe("useChatbotBasePromptEditor opening the form", () => {
    it("starts closed and empty", () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());

        // when
        const editor = result.current;

        // then
        expect(editor.open).toBe(false);
        expect(editor.editing).toBeNull();
        expect(editor.name).toBe("");
        expect(editor.prompt).toBe("");
    });

    it("opens a blank form for a new base prompt", () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.openEdit(makeBasePrompt({ name: "voyager" }));
        });

        // when
        act(() => {
            result.current.openCreate();
        });

        // then
        expect(result.current.open).toBe(true);
        expect(result.current.editing).toBeNull();
        expect(result.current.name).toBe("");
        expect(result.current.prompt).toBe("");
    });

    it("opens the form on the base prompt it was given", () => {
        // given
        const base = makeBasePrompt({ id: "base-9", name: "game witch", prompt: "Obey the hierarchy." });
        const { result } = renderHook(() => useChatbotBasePromptEditor());

        // when
        act(() => {
            result.current.openEdit(base);
        });

        // then
        expect(result.current.open).toBe(true);
        expect(result.current.editing).toBe(base);
        expect(result.current.name).toBe("game witch");
        expect(result.current.prompt).toBe("Obey the hierarchy.");
    });

    it("reports that a save is in flight while a mutation is pending", () => {
        // given
        mocks.creating = true;

        // when
        const { result } = renderHook(() => useChatbotBasePromptEditor());

        // then
        expect(result.current.saving).toBe(true);
    });
});

describe("useChatbotBasePromptEditor saving", () => {
    it("refuses a base prompt with no name or no text", async () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.openCreate();
            result.current.setName("  ");
            result.current.setPrompt("You are a witch of the game boards.");
        });

        // when
        await act(async () => {
            result.current.save();
        });

        // then
        expect(mocks.create).not.toHaveBeenCalled();
        expect(result.current.error).toBe("A base prompt needs a name and some text.");
    });

    it("creates a base prompt from the trimmed name and the text as typed", async () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.openCreate();
            result.current.setName("  game witch  ");
            result.current.setPrompt("  You are a witch of the game boards.  ");
        });

        // when
        await act(async () => {
            result.current.save();
        });

        // then
        expect(mocks.create).toHaveBeenCalledWith({
            name: "game witch",
            prompt: "  You are a witch of the game boards.  ",
        });
        expect(result.current.open).toBe(false);
    });

    it("saves an edit against the base prompt it came from", async () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.openEdit(makeBasePrompt({ id: "base-9" }));
            result.current.setName("voyager");
        });

        // when
        await act(async () => {
            result.current.save();
        });

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "base-9",
            data: { name: "voyager", prompt: "You are a witch of the game boards." },
        });
        expect(mocks.create).not.toHaveBeenCalled();
    });

    it("keeps the form open and says why when the save is rejected", async () => {
        // given
        mocks.create.mockRejectedValue(new Error("that name is already taken"));
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.openCreate();
            result.current.setName("game witch");
            result.current.setPrompt("You are a witch of the game boards.");
        });

        // when
        await act(async () => {
            result.current.save();
        });

        // then
        expect(result.current.error).toBe("that name is already taken");
        expect(result.current.open).toBe(true);
    });
});

describe("useChatbotBasePromptEditor deleting", () => {
    it("refuses to delete a base prompt that bots still extend", () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());

        // when
        act(() => {
            result.current.requestDelete(makeBasePrompt({ name: "game witch", bot_count: 2 }));
        });

        // then
        expect(result.current.pendingDelete).toBeNull();
        expect(result.current.error).toBe("game witch is still used by 2 bot(s). Unassign them first.");
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("holds an unused base prompt awaiting confirmation", () => {
        // given
        const base = makeBasePrompt({ bot_count: 0 });
        const { result } = renderHook(() => useChatbotBasePromptEditor());

        // when
        act(() => {
            result.current.requestDelete(base);
        });

        // then
        expect(result.current.pendingDelete).toBe(base);
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("deletes the held base prompt once the delete is confirmed", async () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.requestDelete(makeBasePrompt({ id: "base-4" }));
        });

        // when
        await act(async () => {
            result.current.confirmDelete();
        });

        // then
        expect(mocks.remove).toHaveBeenCalledWith("base-4");
        expect(result.current.pendingDelete).toBeNull();
    });

    it("lets go of the base prompt without deleting it when the delete is cancelled", () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.requestDelete(makeBasePrompt({ id: "base-4" }));
        });

        // when
        act(() => {
            result.current.cancelDelete();
        });

        // then
        expect(result.current.pendingDelete).toBeNull();
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("does nothing at all when a confirmation arrives with no base prompt held", async () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());

        // when
        await act(async () => {
            result.current.confirmDelete();
        });

        // then
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("says why a base prompt could not be deleted", async () => {
        // given
        mocks.remove.mockRejectedValue(new Error("the base prompt is still referenced"));
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.requestDelete(makeBasePrompt({ id: "base-4" }));
        });

        // when
        await act(async () => {
            result.current.confirmDelete();
        });

        // then
        expect(result.current.error).toBe("the base prompt is still referenced");
    });

    it("clears the refusal once a delete goes through", async () => {
        // given
        const { result } = renderHook(() => useChatbotBasePromptEditor());
        act(() => {
            result.current.requestDelete(makeBasePrompt({ name: "game witch", bot_count: 2 }));
        });
        act(() => {
            result.current.requestDelete(makeBasePrompt({ id: "base-4", bot_count: 0 }));
        });

        // when
        await act(async () => {
            result.current.confirmDelete();
        });

        // then
        expect(result.current.error).toBe("");
    });
});
