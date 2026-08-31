import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeChatMessage } from "../../test-utils/fixtures";
import { providerWrapper } from "../../test-utils/render";
import type { ChatMessage } from "../../types/api";
import { useMessageAnchor } from "./useMessageAnchor";

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({
        sender: { id: "u2", username: "battler", display_name: "Battler" },
        body: "without love it cannot be seen",
        created_at: "2026-08-02T10:00:00Z",
        ...overrides,
    });
}

interface AnchorOptions {
    messages?: ChatMessage[];
    found?: boolean;
}

function renderAnchor(options: AnchorOptions = {}) {
    const loadUntilMessage = vi.fn().mockResolvedValue(options.found ?? false);
    const onError = vi.fn();
    const rendered = renderHook(
        () => useMessageAnchor({ messages: options.messages ?? [], loadUntilMessage, onError }),
        { wrapper: providerWrapper({ route: "/rooms/room-1" }) },
    );

    return { ...rendered, loadUntilMessage, onError };
}

describe("useMessageAnchor", () => {
    it("highlights a message that is already on screen", async () => {
        // given
        const { result } = renderAnchor({ messages: [makeMessage({ id: "m1" })] });
        const element = document.createElement("div");
        element.id = "chat-msg-m1";
        document.body.appendChild(element);

        // when
        await act(async () => {
            await result.current.handleJumpToMessage("m1");
        });

        // then
        expect(result.current.highlightedMsgId).toBe("m1");
        element.remove();
    });

    it("says so when the message cannot be found anywhere in the history", async () => {
        // given
        const { result, onError } = renderAnchor();

        // when
        await act(async () => {
            await result.current.handleJumpToMessage("ghost");
        });

        // then
        expect(onError).toHaveBeenCalledWith("Couldn't locate that message.");
    });
});
