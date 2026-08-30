import { beforeEach, describe, expect, it, vi } from "vitest";
import { destroyHyperbeamEmbed, mountHyperbeamEmbed, setHyperbeamInputEnabled, type HyperbeamHandle } from "./embed";

const mocks = vi.hoisted(() => ({ hyperbeam: vi.fn() }));

vi.mock("@hyperbeam/web", () => ({ default: mocks.hyperbeam, getRegionInfo: vi.fn() }));

function fakeHandle(): HyperbeamHandle & { destroy: ReturnType<typeof vi.fn> } {
    return { destroy: vi.fn(), disableInput: false, userId: "hb-user" } as unknown as HyperbeamHandle & {
        destroy: ReturnType<typeof vi.fn>;
    };
}

beforeEach(() => {
    mocks.hyperbeam.mockReset();
    mocks.hyperbeam.mockResolvedValue(fakeHandle());
});

describe("mountHyperbeamEmbed", () => {
    it("mounts the virtual browser into the container it is given and delegates the keyboard", async () => {
        // given
        const container = document.createElement("div");

        // when
        await mountHyperbeamEmbed({ container, embedURL: "https://hb.test/embed", inputEnabled: true });

        // then
        expect(mocks.hyperbeam).toHaveBeenCalledExactlyOnceWith(container, "https://hb.test/embed", {
            delegateKeyboard: true,
            disableInput: false,
        });
    });

    it("mounts with input disabled for a participant who does not hold control", async () => {
        // given
        const container = document.createElement("div");

        // when
        await mountHyperbeamEmbed({ container, embedURL: "https://hb.test/embed", inputEnabled: false });

        // then
        expect(mocks.hyperbeam.mock.calls[0][2]).toEqual({ delegateKeyboard: true, disableInput: true });
    });

    it("answers with the handle so the caller can identify the participant", async () => {
        // given
        const handle = fakeHandle();
        mocks.hyperbeam.mockResolvedValue(handle);

        // when
        const mounted = await mountHyperbeamEmbed({
            container: document.createElement("div"),
            embedURL: "https://hb.test/embed",
            inputEnabled: true,
        });

        // then
        expect(mounted).toBe(handle);
        expect(mounted.userId).toBe("hb-user");
    });

    it("lets a mount failure reach the caller, because the VM may have expired", async () => {
        // given
        mocks.hyperbeam.mockRejectedValue(new Error("err_no_available_vm"));

        // when
        const attempt = mountHyperbeamEmbed({
            container: document.createElement("div"),
            embedURL: "https://hb.test/embed",
            inputEnabled: true,
        });

        // then
        await expect(attempt).rejects.toThrow("err_no_available_vm");
    });
});

describe("destroyHyperbeamEmbed", () => {
    it("tears the virtual browser down", () => {
        // given
        const handle = fakeHandle();

        // when
        destroyHyperbeamEmbed(handle);

        // then
        expect(handle.destroy).toHaveBeenCalledOnce();
    });

    it("does nothing when the mount never completed", () => {
        // then
        expect(() => destroyHyperbeamEmbed(null)).not.toThrow();
        expect(() => destroyHyperbeamEmbed(undefined)).not.toThrow();
    });
});

describe("setHyperbeamInputEnabled", () => {
    it("hands control to the viewer without remounting the VM", () => {
        // given
        const handle = fakeHandle();
        handle.disableInput = true;

        // when
        setHyperbeamInputEnabled(handle, true);

        // then
        expect(handle.disableInput).toBe(false);
    });

    it("takes control away from the viewer without remounting the VM", () => {
        // given
        const handle = fakeHandle();

        // when
        setHyperbeamInputEnabled(handle, false);

        // then
        expect(handle.disableInput).toBe(true);
    });

    it("does nothing when the mount never completed", () => {
        // then
        expect(() => setHyperbeamInputEnabled(null, true)).not.toThrow();
        expect(() => setHyperbeamInputEnabled(undefined, false)).not.toThrow();
    });
});
