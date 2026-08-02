import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useVote } from "./useVote";

interface VoteProps {
    score: number;
    userVote: number;
    voteFn: (value: number) => Promise<void>;
}

function setup(props: VoteProps) {
    return renderHook(p => useVote(p.score, p.userVote, p.voteFn), { initialProps: props });
}

function noopVote() {
    return Promise.resolve();
}

describe("useVote", () => {
    it("starts from the score and vote it was given", () => {
        // given
        const props = { score: 12, userVote: -1, voteFn: vi.fn(noopVote) };

        // when
        const { result } = setup(props);

        // then
        expect(result.current.score).toBe(12);
        expect(result.current.userVote).toBe(-1);
    });

    it("applies an upvote optimistically before the request settles", async () => {
        // given
        let release: () => void = () => {};
        const voteFn = vi.fn(
            () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        );
        const { result } = setup({ score: 5, userVote: 0, voteFn });

        // when
        let pending: Promise<void> = Promise.resolve();
        act(() => {
            pending = result.current.vote(1);
        });

        // then
        expect(result.current.score).toBe(6);
        expect(result.current.userVote).toBe(1);
        expect(voteFn).toHaveBeenCalledWith(1);
        await act(async () => {
            release();
            await pending;
        });
        expect(result.current.score).toBe(6);
    });

    it("sends zero and removes the contribution when the same vote is cast again", async () => {
        // given
        const voteFn = vi.fn(noopVote);
        const { result } = setup({ score: 5, userVote: 1, voteFn });

        // when
        await act(async () => {
            await result.current.vote(1);
        });

        // then
        expect(voteFn).toHaveBeenCalledWith(0);
        expect(result.current.score).toBe(4);
        expect(result.current.userVote).toBe(0);
    });

    it("swings the score by two when a downvote is changed to an upvote", async () => {
        // given
        const voteFn = vi.fn(noopVote);
        const { result } = setup({ score: 3, userVote: -1, voteFn });

        // when
        await act(async () => {
            await result.current.vote(1);
        });

        // then
        expect(voteFn).toHaveBeenCalledWith(1);
        expect(result.current.score).toBe(5);
        expect(result.current.userVote).toBe(1);
    });

    it("subtracts a downvote from an untouched score", async () => {
        // given
        const voteFn = vi.fn(noopVote);
        const { result } = setup({ score: 0, userVote: 0, voteFn });

        // when
        await act(async () => {
            await result.current.vote(-1);
        });

        // then
        expect(result.current.score).toBe(-1);
        expect(result.current.userVote).toBe(-1);
    });

    it("restores the previous score and vote when the request fails", async () => {
        // given
        const voteFn = vi.fn(() => Promise.reject(new Error("the golden truth denies it")));
        const { result } = setup({ score: 7, userVote: -1, voteFn });

        // when
        await act(async () => {
            await result.current.vote(1);
        });

        // then
        expect(result.current.score).toBe(7);
        expect(result.current.userVote).toBe(-1);
    });

    it("keeps the newer vote when an older request fails after it", async () => {
        // given
        let failFirst: (reason?: unknown) => void = () => {};
        const voteFn = vi.fn(noopVote);
        voteFn.mockImplementationOnce(
            () =>
                new Promise<void>((_resolve, reject) => {
                    failFirst = reject;
                }),
        );
        const { result } = setup({ score: 5, userVote: 0, voteFn });

        // when
        let firstPending: Promise<void> = Promise.resolve();
        act(() => {
            firstPending = result.current.vote(1);
        });
        await act(async () => {
            await result.current.vote(-1);
        });
        await act(async () => {
            failFirst(new Error("the golden truth denies it"));
            await firstPending;
        });

        // then
        expect(voteFn).toHaveBeenNthCalledWith(2, -1);
        expect(result.current.score).toBe(4);
        expect(result.current.userVote).toBe(-1);
    });

    it("resyncs when the incoming score or vote changes", () => {
        // given
        const voteFn = vi.fn(noopVote);
        const { result, rerender } = setup({ score: 4, userVote: 0, voteFn });

        // when
        rerender({ score: 9, userVote: 1, voteFn });

        // then
        expect(result.current.score).toBe(9);
        expect(result.current.userVote).toBe(1);
    });

    it("keeps the optimistic state when the incoming props are unchanged", async () => {
        // given
        const voteFn = vi.fn(noopVote);
        const { result, rerender } = setup({ score: 4, userVote: 0, voteFn });
        await act(async () => {
            await result.current.vote(1);
        });

        // when
        rerender({ score: 4, userVote: 0, voteFn });

        // then
        expect(result.current.score).toBe(5);
        expect(result.current.userVote).toBe(1);
    });
});
