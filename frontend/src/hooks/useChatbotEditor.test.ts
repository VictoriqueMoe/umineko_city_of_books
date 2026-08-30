import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { EMPTY_CHATBOT_FIELDS } from "../domain/chatbots";
import { useChatbotEditor } from "./useChatbotEditor";
import type { Chatbot } from "../types/api";

const mocks = vi.hoisted(() => ({
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    check: vi.fn(),
    creating: false,
    updating: false,
}));

vi.mock("./mutations/admin", () => ({
    useCreateChatbot: () => ({ mutateAsync: mocks.create, isPending: mocks.creating }),
    useUpdateChatbot: () => ({ mutateAsync: mocks.update, isPending: mocks.updating }),
    useDeleteChatbot: () => ({ mutateAsync: mocks.remove, isPending: false }),
    useCheckUsernameAvailable: () => ({ mutateAsync: mocks.check, isPending: false }),
}));

function makeBot(overrides: Partial<Chatbot> = {}): Chatbot {
    return {
        id: "bot-1",
        user_id: "user-1",
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "",
        system_prompt: "You are the Golden Witch.",
        base_prompt_id: null,
        model: "",
        reasoning_effort: "",
        verbosity: "",
        max_output_tokens: 0,
        enabled: true,
        ...overrides,
    };
}

beforeEach(() => {
    mocks.creating = false;
    mocks.updating = false;
    mocks.create.mockResolvedValue(undefined);
    mocks.update.mockResolvedValue(undefined);
    mocks.remove.mockResolvedValue(undefined);
    mocks.check.mockImplementation((username: string) => Promise.resolve({ username, available: true }));
});

describe("useChatbotEditor opening the form", () => {
    it("starts closed with every field blank", () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());

        // when
        const editor = result.current;

        // then
        expect(editor.open).toBe(false);
        expect(editor.editingBot).toBeNull();
        expect(editor.fields).toEqual(EMPTY_CHATBOT_FIELDS);
    });

    it("opens a blank form for a new bot", () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.openEdit(makeBot({ display_name: "Bernkastel" }));
        });

        // when
        act(() => {
            result.current.openCreate();
        });

        // then
        expect(result.current.open).toBe(true);
        expect(result.current.editingBot).toBeNull();
        expect(result.current.fields).toEqual(EMPTY_CHATBOT_FIELDS);
    });

    it("opens the form on the bot it was given", () => {
        // given
        const bot = makeBot({ display_name: "Bernkastel", username: "bern", max_output_tokens: 2048 });
        const { result } = renderHook(() => useChatbotEditor());

        // when
        act(() => {
            result.current.openEdit(bot);
        });

        // then
        expect(result.current.open).toBe(true);
        expect(result.current.editingBot).toBe(bot);
        expect(result.current.fields.username).toBe("bern");
        expect(result.current.fields.maxTokens).toBe("2048");
    });

    it("closes the form and forgets which bot it was editing", () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.openEdit(makeBot());
        });

        // when
        act(() => {
            result.current.close();
        });

        // then
        expect(result.current.open).toBe(false);
        expect(result.current.editingBot).toBeNull();
    });

    it("reports that a save is in flight while either mutation is pending", () => {
        // given
        mocks.creating = true;

        // when
        const { result } = renderHook(() => useChatbotEditor());

        // then
        expect(result.current.saving).toBe(true);
    });
});

describe("useChatbotEditor username check", () => {
    it("asks the server about the trimmed handle and reports it free", async () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.openCreate();
            result.current.setField("username", "  beato  ");
        });

        // when
        await act(async () => {
            result.current.checkUsername();
        });

        // then
        expect(mocks.check).toHaveBeenCalledWith("beato");
        expect(result.current.usernameCheck).toEqual({ state: "available", username: "beato" });
    });

    it("reports a handle the server says is taken", async () => {
        // given
        mocks.check.mockResolvedValue({ username: "beato", available: false });
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.setField("username", "beato");
        });

        // when
        await act(async () => {
            result.current.checkUsername();
        });

        // then
        expect(result.current.usernameCheck).toEqual({ state: "taken", username: "beato" });
        expect(result.current.usernameKnownTaken).toBe(true);
    });

    it("stops treating the handle as taken once it is typed over", async () => {
        // given
        mocks.check.mockResolvedValue({ username: "beato", available: false });
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.setField("username", "beato");
        });
        await act(async () => {
            result.current.checkUsername();
        });

        // when
        act(() => {
            result.current.setField("username", "beatoo");
        });

        // then
        expect(result.current.usernameKnownTaken).toBe(false);
        expect(result.current.usernameCheck).toEqual({ state: "idle" });
    });

    it("records that the check itself failed rather than blocking the save", async () => {
        // given
        mocks.check.mockRejectedValue(new Error("network down"));
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.setField("username", "beato");
        });

        // when
        await act(async () => {
            result.current.checkUsername();
        });

        // then
        expect(result.current.usernameCheck).toEqual({ state: "failed" });
        expect(result.current.usernameKnownTaken).toBe(false);
    });

    it("goes back to idle for a blank handle without asking the server", async () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.setField("username", "   ");
        });

        // when
        await act(async () => {
            result.current.checkUsername();
        });

        // then
        expect(mocks.check).not.toHaveBeenCalled();
        expect(result.current.usernameCheck).toEqual({ state: "idle" });
    });

    it("never checks the handle of a bot that already exists", async () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.openEdit(makeBot({ username: "bern" }));
        });

        // when
        await act(async () => {
            result.current.checkUsername();
        });

        // then
        expect(mocks.check).not.toHaveBeenCalled();
    });
});

describe("useChatbotEditor saving", () => {
    it("creates an enabled bot from the fields that were typed", async () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.openCreate();
            result.current.setField("username", "  beato  ");
            result.current.setField("displayName", "Beato");
        });

        // when
        await act(async () => {
            result.current.save();
        });

        // then
        expect(mocks.create).toHaveBeenCalledWith(
            expect.objectContaining({ username: "beato", display_name: "Beato", enabled: true }),
        );
        expect(result.current.open).toBe(false);
    });

    it("saves an edit against the bot it came from and keeps it switched off", async () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.openEdit(makeBot({ id: "bot-9", enabled: false }));
            result.current.setField("displayName", "Beato");
        });

        // when
        await act(async () => {
            result.current.save();
        });

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "bot-9",
            data: expect.objectContaining({ display_name: "Beato", enabled: false }),
        });
        expect(mocks.create).not.toHaveBeenCalled();
    });

    it("keeps the form open and says why when the save is rejected", async () => {
        // given
        mocks.create.mockRejectedValue(new Error("that username is already taken"));
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.openCreate();
        });

        // when
        await act(async () => {
            result.current.save();
        });

        // then
        expect(result.current.formError).toBe("Could not save the bot: that username is already taken");
        expect(result.current.open).toBe(true);
    });

    it("clears a previous save failure when the form is reopened", async () => {
        // given
        mocks.create.mockRejectedValue(new Error("boom"));
        const { result } = renderHook(() => useChatbotEditor());
        await act(async () => {
            result.current.save();
        });

        // when
        act(() => {
            result.current.openCreate();
        });

        // then
        expect(result.current.formError).toBe("");
    });
});

describe("useChatbotEditor switching a bot on and off", () => {
    it("sends the bot back with only its enabled state changed", async () => {
        // given
        const bot = makeBot({ id: "bot-7", model: "gpt-5", max_output_tokens: 2048 });
        const { result } = renderHook(() => useChatbotEditor());

        // when
        await act(async () => {
            result.current.toggleEnabled(bot, false);
        });

        // then
        expect(mocks.update).toHaveBeenCalledWith({
            id: "bot-7",
            data: expect.objectContaining({ model: "gpt-5", max_output_tokens: 2048, enabled: false }),
        });
        expect(result.current.togglingID).toBeNull();
    });

    it("names the bot and the direction that could not be switched", async () => {
        // given
        mocks.update.mockRejectedValue(new Error("the model is unreachable"));
        const { result } = renderHook(() => useChatbotEditor());

        // when
        await act(async () => {
            result.current.toggleEnabled(makeBot({ display_name: "Beatrice" }), false);
        });

        // then
        expect(result.current.error).toBe("Could not switch Beatrice off: the model is unreachable");
    });

    it("names the other direction when switching a bot back on fails", async () => {
        // given
        mocks.update.mockRejectedValue(new Error("the model is unreachable"));
        const { result } = renderHook(() => useChatbotEditor());

        // when
        await act(async () => {
            result.current.toggleEnabled(makeBot({ display_name: "Beatrice" }), true);
        });

        // then
        expect(result.current.error).toBe("Could not switch Beatrice on: the model is unreachable");
    });
});

describe("useChatbotEditor deleting", () => {
    it("holds the bot awaiting confirmation without deleting anything", () => {
        // given
        const bot = makeBot();
        const { result } = renderHook(() => useChatbotEditor());

        // when
        act(() => {
            result.current.requestDelete(bot);
        });

        // then
        expect(result.current.pendingDelete).toBe(bot);
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("deletes the held bot once the delete is confirmed", async () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.requestDelete(makeBot({ id: "bot-3" }));
        });

        // when
        await act(async () => {
            result.current.confirmDelete();
        });

        // then
        expect(mocks.remove).toHaveBeenCalledWith("bot-3");
        expect(result.current.pendingDelete).toBeNull();
    });

    it("lets go of the bot without deleting it when the delete is cancelled", () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.requestDelete(makeBot({ id: "bot-3" }));
        });

        // when
        act(() => {
            result.current.cancelDelete();
        });

        // then
        expect(result.current.pendingDelete).toBeNull();
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("does nothing at all when a confirmation arrives with no bot held", async () => {
        // given
        const { result } = renderHook(() => useChatbotEditor());

        // when
        await act(async () => {
            result.current.confirmDelete();
        });

        // then
        expect(mocks.remove).not.toHaveBeenCalled();
    });

    it("says which bot could not be deleted", async () => {
        // given
        mocks.remove.mockRejectedValue(new Error("the bot still owns messages"));
        const { result } = renderHook(() => useChatbotEditor());
        act(() => {
            result.current.requestDelete(makeBot({ display_name: "Beatrice" }));
        });

        // when
        await act(async () => {
            result.current.confirmDelete();
        });

        // then
        expect(result.current.error).toBe("Could not delete Beatrice: the bot still owns messages");
    });
});
