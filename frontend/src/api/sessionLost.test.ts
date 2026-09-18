import { beforeEach, describe, expect, it, vi } from "vitest";
import { clearSessionLost, isSessionLost, markSessionLost, subscribeSessionLost } from "./sessionLost";

beforeEach(() => {
    clearSessionLost();
});

describe("sessionLost", () => {
    it("starts with the session intact", () => {
        // given
        const fresh = isSessionLost;

        // when
        const lost = fresh();

        // then
        expect(lost).toBe(false);
    });

    it("tells its listeners once when the session is lost", () => {
        // given
        const listener = vi.fn();
        subscribeSessionLost(listener);

        // when
        markSessionLost();
        markSessionLost();

        // then
        expect(isSessionLost()).toBe(true);
        expect(listener).toHaveBeenCalledOnce();
    });

    it("tells its listeners when the session is restored", () => {
        // given
        markSessionLost();
        const listener = vi.fn();
        subscribeSessionLost(listener);

        // when
        clearSessionLost();
        clearSessionLost();

        // then
        expect(isSessionLost()).toBe(false);
        expect(listener).toHaveBeenCalledOnce();
    });

    it("stops telling a listener that has unsubscribed", () => {
        // given
        const listener = vi.fn();
        const unsubscribe = subscribeSessionLost(listener);

        // when
        unsubscribe();
        markSessionLost();

        // then
        expect(listener).not.toHaveBeenCalled();
    });
});
