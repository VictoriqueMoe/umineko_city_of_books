import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useHyperbeamEmbed, type UseHyperbeamEmbedOptions } from "./useHyperbeamEmbed";

const mocks = vi.hoisted(() => ({ hyperbeam: vi.fn() }));

vi.mock("@hyperbeam/web", () => ({ default: mocks.hyperbeam }));

function Harness(props: UseHyperbeamEmbedOptions) {
    const { wrapRef, mountError } = useHyperbeamEmbed(props);

    return (
        <div>
            <div data-testid="wrap" ref={wrapRef} />
            {mountError && <p data-testid="mount-error">{mountError}</p>}
        </div>
    );
}

interface HarnessOptions {
    embedURL?: string;
    isOpen?: boolean;
    hasControl?: boolean;
}

function makeHandle(userId = "hb-user-1") {
    return { destroy: vi.fn(), disableInput: true, userId };
}

function renderEmbed(options: HarnessOptions = {}) {
    const onIdentify = vi.fn();

    const props = (overrides: HarnessOptions = {}): UseHyperbeamEmbedOptions => ({
        embedURL: overrides.embedURL ?? options.embedURL ?? "https://hb.test/embed",
        isOpen: overrides.isOpen ?? options.isOpen ?? true,
        hasControl: overrides.hasControl ?? options.hasControl ?? false,
        onIdentify,
    });

    const view = render(<Harness {...props()} />);

    return {
        ...view,
        onIdentify,
        update: (overrides: HarnessOptions) => view.rerender(<Harness {...props(overrides)} />),
    };
}

beforeEach(() => {
    mocks.hyperbeam.mockReset();
    mocks.hyperbeam.mockResolvedValue(makeHandle());
    vi.spyOn(console, "error").mockImplementation(() => {});
});

describe("useHyperbeamEmbed mounting", () => {
    it("mounts the virtual browser at the embed url with input locked for a passenger", async () => {
        // given
        const options = { embedURL: "https://hb.test/embed-9", hasControl: false };

        // when
        renderEmbed(options);

        // then
        await waitFor(() => {
            expect(mocks.hyperbeam).toHaveBeenCalledOnce();
        });
        expect(mocks.hyperbeam.mock.calls[0][1]).toBe("https://hb.test/embed-9");
        expect(mocks.hyperbeam.mock.calls[0][2]).toEqual({ delegateKeyboard: true, disableInput: true });
    });

    it("unlocks input straight away for the viewer who already holds control", async () => {
        // given
        const options = { hasControl: true };

        // when
        renderEmbed(options);

        // then
        await waitFor(() => {
            expect(mocks.hyperbeam).toHaveBeenCalledOnce();
        });
        expect(mocks.hyperbeam.mock.calls[0][2]).toEqual({ delegateKeyboard: true, disableInput: false });
    });

    it("mounts the embed into a container of its own inside the wrapper", async () => {
        // given
        renderEmbed();

        // when
        await waitFor(() => {
            expect(mocks.hyperbeam).toHaveBeenCalledOnce();
        });

        // then
        const container = mocks.hyperbeam.mock.calls[0][0] as HTMLElement;
        expect(container.parentElement).toBe(screen.getByTestId("wrap"));
        expect(container.style.position).toBe("absolute");
    });

    it("waits without mounting anything while the embed url is still missing", () => {
        // given
        const options = { embedURL: "" };

        // when
        renderEmbed(options);

        // then
        expect(mocks.hyperbeam).not.toHaveBeenCalled();
    });

    it("mounts nothing while the party window is closed", () => {
        // given
        const options = { isOpen: false };

        // when
        renderEmbed(options);

        // then
        expect(mocks.hyperbeam).not.toHaveBeenCalled();
    });

    it("tells the caller which seat the virtual browser handed the viewer", async () => {
        // given
        mocks.hyperbeam.mockResolvedValue(makeHandle("hb-user-7"));

        // when
        const { onIdentify } = renderEmbed();

        // then
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalledWith("hb-user-7");
        });
    });

    it("identifies nobody when the virtual browser hands back no seat", async () => {
        // given
        mocks.hyperbeam.mockResolvedValue(makeHandle(""));

        // when
        const { onIdentify } = renderEmbed();

        // then
        await waitFor(() => {
            expect(mocks.hyperbeam).toHaveBeenCalledOnce();
        });
        expect(onIdentify).not.toHaveBeenCalled();
    });

    it("explains a virtual browser that refused to connect", async () => {
        // given
        mocks.hyperbeam.mockRejectedValue(new Error("the VM has expired"));

        // when
        renderEmbed();

        // then
        expect(await screen.findByTestId("mount-error")).toHaveTextContent("the VM has expired");
    });

    it("falls back to a plain message when the refusal carries none", async () => {
        // given
        mocks.hyperbeam.mockRejectedValue(new Error(""));

        // when
        renderEmbed();

        // then
        expect(await screen.findByTestId("mount-error")).toHaveTextContent("Failed to connect to virtual browser");
    });

    it("tears the virtual browser down and removes its container when the window goes away", async () => {
        // given
        const handle = makeHandle();
        mocks.hyperbeam.mockResolvedValue(handle);
        const { unmount, onIdentify } = renderEmbed();
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalled();
        });
        const container = mocks.hyperbeam.mock.calls[0][0] as HTMLElement;

        // when
        unmount();

        // then
        expect(handle.destroy).toHaveBeenCalledOnce();
        expect(container.parentElement).toBeNull();
    });

    it("builds a fresh virtual browser when the embed url changes", async () => {
        // given
        const first = makeHandle();
        mocks.hyperbeam.mockResolvedValue(first);
        const { update, onIdentify } = renderEmbed({ embedURL: "https://hb.test/embed-1" });
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalled();
        });

        // when
        update({ embedURL: "https://hb.test/embed-2" });

        // then
        await waitFor(() => {
            expect(mocks.hyperbeam).toHaveBeenCalledTimes(2);
        });
        expect(first.destroy).toHaveBeenCalledOnce();
        expect(mocks.hyperbeam.mock.calls[1][1]).toBe("https://hb.test/embed-2");
    });
});

describe("useHyperbeamEmbed control changes", () => {
    it("never remounts the virtual machine when control changes hands", async () => {
        // given
        const handle = makeHandle();
        mocks.hyperbeam.mockResolvedValue(handle);
        const { update, onIdentify } = renderEmbed({ hasControl: false });
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalled();
        });

        // when
        update({ hasControl: true });
        update({ hasControl: false });
        update({ hasControl: true });

        // then
        expect(mocks.hyperbeam).toHaveBeenCalledOnce();
        expect(handle.destroy).not.toHaveBeenCalled();
    });

    it("unlocks the running virtual browser when control is handed to the viewer", async () => {
        // given
        const handle = makeHandle();
        mocks.hyperbeam.mockResolvedValue(handle);
        const { update, onIdentify } = renderEmbed({ hasControl: false });
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalled();
        });

        // when
        update({ hasControl: true });

        // then
        expect(handle.disableInput).toBe(false);
    });

    it("locks the running virtual browser again when control is taken away", async () => {
        // given
        const handle = makeHandle();
        mocks.hyperbeam.mockResolvedValue(handle);
        const { update, onIdentify } = renderEmbed({ hasControl: true });
        await waitFor(() => {
            expect(onIdentify).toHaveBeenCalled();
        });

        // when
        update({ hasControl: false });

        // then
        expect(handle.disableInput).toBe(true);
    });
});
