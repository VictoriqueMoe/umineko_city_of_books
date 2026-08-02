import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { providerWrapper } from "../test-utils/render";
import { useBlock } from "./useBlock";

const mocks = vi.hoisted(() => ({
    useBlockStatus: vi.fn(),
    block: vi.fn(),
    unblock: vi.fn(),
    refresh: vi.fn(),
}));

vi.mock("../api/queries/misc", () => ({
    useBlockStatus: mocks.useBlockStatus,
}));

vi.mock("../api/mutations/misc", () => ({
    useBlockUser: () => ({ mutateAsync: mocks.block }),
    useUnblockUser: () => ({ mutateAsync: mocks.unblock }),
}));

const userId = "11111111-1111-1111-1111-111111111111";

function statusResult(blocking: boolean, loading = false) {
    return { status: { blocking, blocked_by: false }, loading, refresh: mocks.refresh };
}

function setup(id: string = userId) {
    return renderHook(() => useBlock(id), { wrapper: providerWrapper() });
}

beforeEach(() => {
    mocks.block.mockResolvedValue(undefined);
    mocks.unblock.mockResolvedValue(undefined);
    mocks.refresh.mockResolvedValue(undefined);
    mocks.useBlockStatus.mockReturnValue(statusResult(false));
});

describe("useBlock", () => {
    it("asks for the block status of the user it was given", () => {
        // given
        mocks.useBlockStatus.mockReturnValue(statusResult(true, true));

        // when
        const { result } = setup();

        // then
        expect(mocks.useBlockStatus).toHaveBeenCalledWith(userId);
        expect(result.current.status).toEqual({ blocking: true, blocked_by: false });
        expect(result.current.loading).toBe(true);
    });

    it("blocks the user when they are not blocked yet", async () => {
        // given
        mocks.useBlockStatus.mockReturnValue(statusResult(false));
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.toggleBlock();
        });

        // then
        expect(mocks.block).toHaveBeenCalledWith(userId);
        expect(mocks.unblock).not.toHaveBeenCalled();
    });

    it("unblocks the user when they are already blocked", async () => {
        // given
        mocks.useBlockStatus.mockReturnValue(statusResult(true));
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.toggleBlock();
        });

        // then
        expect(mocks.unblock).toHaveBeenCalledWith(userId);
        expect(mocks.block).not.toHaveBeenCalled();
    });

    it("refreshes the status only after the mutation has settled", async () => {
        // given
        mocks.useBlockStatus.mockReturnValue(statusResult(false));
        let release: () => void = () => {};
        mocks.block.mockImplementation(
            () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        );
        const { result } = setup();

        // when
        let pending: Promise<void> = Promise.resolve();
        act(() => {
            pending = result.current.toggleBlock();
        });

        // then
        expect(mocks.refresh).not.toHaveBeenCalled();
        await act(async () => {
            release();
            await pending;
        });
        expect(mocks.refresh).toHaveBeenCalledOnce();
    });

    it("does nothing when it has no user id to act on", async () => {
        // given
        const { result } = setup("");

        // when
        await act(async () => {
            await result.current.toggleBlock();
        });

        // then
        expect(mocks.block).not.toHaveBeenCalled();
        expect(mocks.unblock).not.toHaveBeenCalled();
        expect(mocks.refresh).not.toHaveBeenCalled();
    });

    it("does nothing when the status hook hands back no status at all", async () => {
        // given
        mocks.useBlockStatus.mockReturnValue({ status: null, loading: true, refresh: mocks.refresh });
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.toggleBlock();
        });

        // then
        expect(mocks.block).not.toHaveBeenCalled();
        expect(mocks.refresh).not.toHaveBeenCalled();
    });

    it("does nothing while the placeholder status is still loading", async () => {
        // given
        mocks.useBlockStatus.mockReturnValue(statusResult(false, true));
        const { result } = setup();

        // when
        await act(async () => {
            await result.current.toggleBlock();
        });

        // then
        expect(mocks.block).not.toHaveBeenCalled();
        expect(mocks.unblock).not.toHaveBeenCalled();
        expect(mocks.refresh).not.toHaveBeenCalled();
    });

    it("swallows a failed block and leaves the status unrefreshed", async () => {
        // given
        mocks.useBlockStatus.mockReturnValue(statusResult(false));
        mocks.block.mockRejectedValue(new Error("the witch refuses"));
        const { result } = setup();

        // when
        await act(async () => {
            await expect(result.current.toggleBlock()).resolves.toBeUndefined();
        });

        // then
        expect(mocks.block).toHaveBeenCalledWith(userId);
        expect(mocks.refresh).not.toHaveBeenCalled();
    });
});
