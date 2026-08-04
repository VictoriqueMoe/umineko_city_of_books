import { StrictMode, useEffect } from "react";
import { describe, expect, it, vi } from "vitest";
import { render } from "@testing-library/react";

describe("probe", () => {
    it("does StrictMode double-invoke effects here", () => {
        const setup = vi.fn();
        const cleanup = vi.fn();
        function P() {
            useEffect(() => {
                setup();
                return () => cleanup();
            }, []);
            return null;
        }
        render(
            <StrictMode>
                <P />
            </StrictMode>,
        );
        expect({ setup: setup.mock.calls.length, cleanup: cleanup.mock.calls.length }).toEqual({ setup: -1, cleanup: -1 });
    });
});
