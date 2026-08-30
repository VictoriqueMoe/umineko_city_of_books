import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { UserProfile } from "../types/api";
import { useStreamChatJoin } from "./useStreamChatJoin";

const mocks = vi.hoisted(() => ({
    getStreamViewerToken: vi.fn(),
    joinStreamChat: vi.fn(),
    resetStreamCredentials: vi.fn(),
    startStream: vi.fn(),
    stopStream: vi.fn(),
    updateStreamTitle: vi.fn(),
    uploadStreamThumbnail: vi.fn(),
}));

vi.mock("../api/endpoints/stream", () => mocks);

interface JoinProps {
    streamId: string;
    isLive: boolean;
}

function mount(initialProps: JoinProps, user: UserProfile | null = makeUser()) {
    return renderHook(({ streamId, isLive }: JoinProps) => useStreamChatJoin(streamId, isLive), {
        wrapper: providerWrapper({ user }),
        initialProps,
    });
}

beforeEach(() => {
    mocks.joinStreamChat.mockResolvedValue(undefined);
});

describe("useStreamChatJoin", () => {
    it("joins the chat of the stream being watched", async () => {
        // given
        const view = mount({ streamId: "stream-77", isLive: true });

        // when
        await waitFor(() => expect(view.result.current.joined).toBe(true));

        // then
        expect(mocks.joinStreamChat).toHaveBeenCalledWith("stream-77");
        expect(view.result.current.failed).toBe(false);
    });

    it("never joins on behalf of a signed out visitor", () => {
        // given
        const signedOut = null;

        // when
        const view = mount({ streamId: "stream-77", isLive: true }, signedOut);

        // then
        expect(mocks.joinStreamChat).not.toHaveBeenCalled();
        expect(view.result.current.joined).toBe(false);
    });

    it("never joins while the stream is offline", () => {
        // given
        const offline = false;

        // when
        const view = mount({ streamId: "stream-77", isLive: offline });

        // then
        expect(mocks.joinStreamChat).not.toHaveBeenCalled();
        expect(view.result.current.joined).toBe(false);
    });

    it("admits when the chat could not be joined", async () => {
        // given
        mocks.joinStreamChat.mockRejectedValue(new Error("no room at the inn"));

        // when
        const view = mount({ streamId: "stream-77", isLive: true });

        // then
        await waitFor(() => expect(view.result.current.failed).toBe(true));
        expect(view.result.current.joined).toBe(false);
    });

    it("forgets that it joined the moment the viewer switches stream", async () => {
        // given
        const view = mount({ streamId: "stream-1", isLive: true });
        await waitFor(() => expect(view.result.current.joined).toBe(true));

        // when
        view.rerender({ streamId: "stream-2", isLive: true });

        // then
        expect(view.result.current.joined).toBe(false);
        await waitFor(() => expect(view.result.current.joined).toBe(true));
        expect(mocks.joinStreamChat).toHaveBeenLastCalledWith("stream-2");
    });

    it("forgets a failed join the moment the viewer switches stream", async () => {
        // given
        mocks.joinStreamChat.mockRejectedValueOnce(new Error("no room at the inn"));
        const view = mount({ streamId: "stream-1", isLive: true });
        await waitFor(() => expect(view.result.current.failed).toBe(true));

        // when
        view.rerender({ streamId: "stream-2", isLive: true });

        // then
        expect(view.result.current.failed).toBe(false);
        await waitFor(() => expect(view.result.current.joined).toBe(true));
    });

    it("never counts a join that landed for the stream the viewer has left", async () => {
        // given
        let letIn = () => {};
        mocks.joinStreamChat.mockReturnValueOnce(
            new Promise<void>(resolve => {
                letIn = resolve;
            }),
        );
        mocks.joinStreamChat.mockReturnValueOnce(new Promise<void>(() => {}));
        const view = mount({ streamId: "stream-1", isLive: true });

        // when
        view.rerender({ streamId: "stream-2", isLive: true });
        await waitFor(() => expect(mocks.joinStreamChat).toHaveBeenLastCalledWith("stream-2"));
        letIn();

        // then
        expect(view.result.current.joined).toBe(false);
    });
});
