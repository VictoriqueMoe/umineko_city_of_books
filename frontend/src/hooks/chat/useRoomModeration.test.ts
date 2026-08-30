import { act, renderHook } from "@testing-library/react";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ChatMessage, ChatRoomMember } from "../../types/api";
import { useRoomModeration } from "./useRoomModeration";

const mocks = vi.hoisted(() => ({
    kick: vi.fn(),
    ban: vi.fn(),
    setNickname: vi.fn(),
    unlockNickname: vi.fn(),
    setMemberTimeout: vi.fn(),
    clearMemberTimeout: vi.fn(),
}));

vi.mock("../mutations/chat", () => ({
    useKickChatRoomMember: () => ({ mutateAsync: mocks.kick }),
    useBanChatRoomMember: () => ({ mutateAsync: mocks.ban }),
    useSetChatRoomMemberNickname: () => ({ mutateAsync: mocks.setNickname }),
    useUnlockChatRoomMemberNickname: () => ({ mutateAsync: mocks.unlockNickname }),
    useSetChatRoomMemberTimeout: () => ({ mutateAsync: mocks.setMemberTimeout }),
    useClearChatRoomMemberTimeout: () => ({ mutateAsync: mocks.clearMemberTimeout }),
}));

function makeRoomMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return {
        user: { id: "u2", username: "battler", display_name: "Battler" },
        role: "member",
        joined_at: "2026-01-01T00:00:00Z",
        nickname: "",
        member_avatar_url: "",
        nickname_locked: false,
        ...overrides,
    };
}

interface ModerationOptions {
    members?: ChatRoomMember[];
    messages?: ChatMessage[];
}

function renderModeration(options: ModerationOptions = {}) {
    const notify = vi.fn();
    const initialMembers = options.members ?? [makeRoomMember()];
    const initialMessages = options.messages ?? [];

    const rendered = renderHook(() => {
        const [members, setMembers] = useState<ChatRoomMember[]>(initialMembers);
        const [messages, setMessages] = useState<ChatMessage[]>(initialMessages);
        const moderation = useRoomModeration({ roomId: "room-1", setMembers, setMessages, notify });

        return { members, messages, moderation };
    });

    return { ...rendered, notify };
}

beforeEach(() => {
    mocks.kick.mockResolvedValue(undefined);
    mocks.ban.mockResolvedValue(undefined);
    mocks.setNickname.mockResolvedValue(makeRoomMember({ nickname: "Beato" }));
    mocks.unlockNickname.mockResolvedValue(makeRoomMember({ nickname_locked: false }));
    mocks.setMemberTimeout.mockResolvedValue(makeRoomMember({ timeout_until: "2099-01-01T00:00:00Z" }));
    mocks.clearMemberTimeout.mockResolvedValue(makeRoomMember({ timeout_until: undefined }));
});

describe("useRoomModeration", () => {
    it("shows nothing at all when there is no timeout to format", () => {
        // given
        const { result } = renderModeration();

        // when
        const formatted = result.current.moderation.formatTimeoutUntil(undefined);

        // then
        expect(formatted).toBe("");
    });

    it("hands back a timestamp it cannot parse untouched", () => {
        // given
        const { result } = renderModeration();

        // when
        const formatted = result.current.moderation.formatTimeoutUntil("not a date");

        // then
        expect(formatted).toBe("not a date");
    });

    it("kicks a member once the viewer confirms and drops them from the list", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const { result } = renderModeration();

        // when
        await act(async () => {
            await result.current.moderation.handleKick("u2");
        });

        // then
        expect(mocks.kick).toHaveBeenCalledWith("u2");
        expect(result.current.members).toEqual([]);
        confirm.mockRestore();
    });

    it("leaves the member alone when the viewer backs out of the kick", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { result } = renderModeration();

        // when
        await act(async () => {
            await result.current.moderation.handleKick("u2");
        });

        // then
        expect(mocks.kick).not.toHaveBeenCalled();
        expect(result.current.members).toHaveLength(1);
        confirm.mockRestore();
    });

    it("bans a member with the reason the viewer typed", async () => {
        // given
        const prompt = vi.spyOn(window, "prompt").mockReturnValue("kept saying the same thing");
        const { result, notify } = renderModeration();

        // when
        await act(async () => {
            await result.current.moderation.handleBan("u2");
        });

        // then
        expect(mocks.ban).toHaveBeenCalledWith({ userId: "u2", reason: "kept saying the same thing" });
        expect(notify).toHaveBeenCalledWith("Member banned from the room.");
        prompt.mockRestore();
    });

    it("leaves the member alone when the viewer cancels the ban prompt", async () => {
        // given
        const prompt = vi.spyOn(window, "prompt").mockReturnValue(null);
        const { result } = renderModeration();

        // when
        await act(async () => {
            await result.current.moderation.handleBan("u2");
        });

        // then
        expect(mocks.ban).not.toHaveBeenCalled();
        expect(result.current.members).toHaveLength(1);
        prompt.mockRestore();
    });

    it("opens the nickname dialogue on the member the viewer picked", () => {
        // given
        const member = makeRoomMember({ nickname: "Battler-kun" });
        const { result } = renderModeration({ members: [member] });
        act(() => {
            result.current.moderation.setOpenMemberMenu("u2");
        });

        // when
        act(() => {
            result.current.moderation.openNicknameDialog(member);
        });

        // then
        expect(result.current.moderation.nicknameDialogTarget?.user.id).toBe("u2");
        expect(result.current.moderation.nicknameDialogValue).toBe("Battler-kun");
        expect(result.current.moderation.openMemberMenu).toBeNull();
    });

    it("saves a trimmed nickname and closes the dialogue", async () => {
        // given
        const member = makeRoomMember();
        const { result } = renderModeration({ members: [member] });
        act(() => {
            result.current.moderation.openNicknameDialog(member);
        });
        act(() => {
            result.current.moderation.setNicknameDialogValue("  Beato  ");
        });

        // when
        await act(async () => {
            await result.current.moderation.handleModSetNickname();
        });

        // then
        expect(mocks.setNickname).toHaveBeenCalledWith({ userId: "u2", nickname: "Beato" });
        expect(result.current.moderation.nicknameDialogTarget).toBeNull();
    });

    it("keeps the nickname dialogue open and explains why the save failed", async () => {
        // given
        mocks.setNickname.mockRejectedValue(new Error("that nickname is taken"));
        const member = makeRoomMember();
        const { result } = renderModeration({ members: [member] });
        act(() => {
            result.current.moderation.openNicknameDialog(member);
        });

        // when
        await act(async () => {
            await result.current.moderation.handleModSetNickname();
        });

        // then
        expect(result.current.moderation.nicknameDialogError).toBe("that nickname is taken");
        expect(result.current.moderation.nicknameDialogTarget?.user.id).toBe("u2");
        expect(result.current.moderation.nicknameDialogSaving).toBe(false);
    });

    it("unlocks a nickname the staff had frozen", async () => {
        // given
        const { result } = renderModeration();

        // when
        await act(async () => {
            await result.current.moderation.handleModUnlockNickname("u2");
        });

        // then
        expect(mocks.unlockNickname).toHaveBeenCalledWith("u2");
        expect(result.current.moderation.busy).toBeNull();
    });

    it("refuses a timeout that is not a whole number of units", async () => {
        // given
        const member = makeRoomMember();
        const { result } = renderModeration({ members: [member] });
        act(() => {
            result.current.moderation.openTimeoutDialog(member);
        });
        act(() => {
            result.current.moderation.setTimeoutDialogAmount("0");
        });

        // when
        await act(async () => {
            await result.current.moderation.handleSetTimeout();
        });

        // then
        expect(result.current.moderation.timeoutDialogError).toBe("Enter a whole number greater than zero");
        expect(mocks.setMemberTimeout).not.toHaveBeenCalled();
    });

    it("times a member out for the amount and unit the viewer chose", async () => {
        // given
        const member = makeRoomMember();
        const { result } = renderModeration({ members: [member] });
        act(() => {
            result.current.moderation.openTimeoutDialog(member);
        });
        act(() => {
            result.current.moderation.setTimeoutDialogAmount("3");
            result.current.moderation.setTimeoutDialogUnit("days");
        });

        // when
        await act(async () => {
            await result.current.moderation.handleSetTimeout();
        });

        // then
        expect(mocks.setMemberTimeout).toHaveBeenCalledWith({ userId: "u2", amount: 3, unit: "days" });
        expect(result.current.members[0].timeout_until).toBe("2099-01-01T00:00:00Z");
        expect(result.current.moderation.timeoutDialogTarget).toBeNull();
    });

    it("explains why a timeout could not be set", async () => {
        // given
        mocks.setMemberTimeout.mockRejectedValue(new Error("you cannot time out a host"));
        const member = makeRoomMember();
        const { result } = renderModeration({ members: [member] });
        act(() => {
            result.current.moderation.openTimeoutDialog(member);
        });

        // when
        await act(async () => {
            await result.current.moderation.handleSetTimeout();
        });

        // then
        expect(result.current.moderation.timeoutDialogError).toBe("you cannot time out a host");
    });

    it("clears a timeout and puts the fresh member back in the list", async () => {
        // given
        const { result } = renderModeration();

        // when
        await act(async () => {
            await result.current.moderation.handleClearTimeout("u2");
        });

        // then
        expect(mocks.clearMemberTimeout).toHaveBeenCalledWith("u2");
        expect(result.current.members[0].timeout_until).toBeUndefined();
        expect(result.current.moderation.busy).toBeNull();
    });
});
