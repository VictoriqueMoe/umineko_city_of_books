import { createElement, type ReactNode } from "react";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MemoryRouter, useLocation } from "react-router";
import { hasNextPage, usePageOffset, useSearchParamPage, type SearchParamPageOptions } from "./usePageOffset";

function routerWrapper(route: string) {
    return function Wrapper({ children }: { children: ReactNode }) {
        return createElement(MemoryRouter, { initialEntries: [route] }, children);
    };
}

function renderSearchParamPage(route: string, options: SearchParamPageOptions) {
    return renderHook(() => ({ pager: useSearchParamPage(options), search: useLocation().search }), {
        wrapper: routerWrapper(route),
    });
}

describe("hasNextPage", () => {
    it("reports no next page while the total is still zero", () => {
        // given
        const page = { offset: 0, limit: 20 };

        // then
        expect(hasNextPage(page, 0)).toBe(false);
    });

    it("reports a next page while the total runs past the current window", () => {
        // given
        const page = { offset: 0, limit: 20 };

        // then
        expect(hasNextPage(page, 45)).toBe(true);
    });

    it("treats an exactly filled last page as the end", () => {
        // given
        const page = { offset: 20, limit: 20 };

        // then
        expect(hasNextPage(page, 40)).toBe(false);
    });

    it("reports no next page once the offset has run past the total", () => {
        // given
        const page = { offset: 60, limit: 20 };

        // then
        expect(hasNextPage(page, 45)).toBe(false);
    });
});

describe("usePageOffset", () => {
    it("starts on the first page with nothing behind it", () => {
        // given
        const { result } = renderHook(() => usePageOffset({ limit: 20 }));

        // then
        expect(result.current.offset).toBe(0);
        expect(result.current.limit).toBe(20);
        expect(result.current.hasPrev).toBe(false);
    });

    it("advances one limit at a time", () => {
        // given
        const { result } = renderHook(() => usePageOffset({ limit: 20 }));

        // when
        act(() => result.current.goNext());
        act(() => result.current.goNext());

        // then
        expect(result.current.offset).toBe(40);
        expect(result.current.hasPrev).toBe(true);
    });

    it("clamps the previous page at zero instead of going negative", () => {
        // given
        const { result } = renderHook(() => usePageOffset({ limit: 30 }));
        act(() => result.current.goNext());

        // when
        act(() => result.current.goPrev());
        act(() => result.current.goPrev());

        // then
        expect(result.current.offset).toBe(0);
        expect(result.current.hasPrev).toBe(false);
    });

    it("walks past the end when asked to, because only the caller knows the total", () => {
        // given
        const { result } = renderHook(() => usePageOffset({ limit: 20 }));

        // when
        act(() => result.current.goNext());
        act(() => result.current.goNext());
        act(() => result.current.goNext());

        // then
        expect(result.current.offset).toBe(60);
        expect(hasNextPage(result.current, 45)).toBe(false);
    });

    it("returns to the first page on reset", () => {
        // given
        const { result } = renderHook(() => usePageOffset({ limit: 20 }));
        act(() => result.current.goNext());
        act(() => result.current.goNext());

        // when
        act(() => result.current.reset());

        // then
        expect(result.current.offset).toBe(0);
    });

    it("shrinks the step when the limit changes", () => {
        // given
        const { result, rerender } = renderHook(({ limit }) => usePageOffset({ limit }), {
            initialProps: { limit: 20 },
        });
        act(() => result.current.goNext());

        // when
        rerender({ limit: 5 });
        act(() => result.current.goNext());

        // then
        expect(result.current.offset).toBe(25);
    });
});

describe("useSearchParamPage", () => {
    const parseCases = [
        { name: "no param at all", route: "/list", expected: 0 },
        { name: "an empty param", route: "/list?page=", expected: 0 },
        { name: "a blank param", route: "/list?page=%20", expected: 0 },
        { name: "the first page itself", route: "/list?page=1", expected: 0 },
        { name: "a later page", route: "/list?page=4", expected: 60 },
        { name: "a fractional page", route: "/list?page=4.9", expected: 60 },
        { name: "a malformed page", route: "/list?page=abc", expected: 0 },
        { name: "a negative page", route: "/list?page=-7", expected: 0 },
        { name: "a zero page in a one-based contract", route: "/list?page=0", expected: 0 },
        { name: "an overflowing page", route: "/list?page=1e999", expected: 0 },
    ];

    it.each(parseCases)("resolves an offset from $name", ({ route, expected }) => {
        // given
        const { result } = renderSearchParamPage(route, { limit: 20 });

        // then
        expect(result.current.pager.offset).toBe(expected);
    });

    it("leaves a malformed param in the url rather than correcting it", () => {
        // given
        const { result } = renderSearchParamPage("/list?page=abc&q=beatrice", { limit: 20 });

        // then
        expect(result.current.pager.offset).toBe(0);
        expect(result.current.search).toBe("?page=abc&q=beatrice");
    });

    it("pages forward from a malformed param as if it were the first page", () => {
        // given
        const { result } = renderSearchParamPage("/list?page=abc", { limit: 20 });

        // when
        act(() => result.current.pager.goNext());

        // then
        expect(result.current.search).toBe("?page=2");
        expect(result.current.pager.offset).toBe(20);
    });

    it("reads a zero-based param when the first page is zero", () => {
        // given
        const { result } = renderSearchParamPage("/search?page=2", { limit: 20, firstPage: 0 });

        // when
        act(() => result.current.pager.goNext());

        // then
        expect(result.current.pager.offset).toBe(60);
        expect(result.current.search).toBe("?page=3");
    });

    it("reads the param name it was given", () => {
        // given
        const { result } = renderSearchParamPage("/list?p=3", { limit: 10, param: "p" });

        // then
        expect(result.current.pager.offset).toBe(20);
    });

    it("keeps every other param when it writes the page", () => {
        // given
        const { result } = renderSearchParamPage("/list?sort=updated&tag=witch", { limit: 25 });

        // when
        act(() => result.current.pager.goNext());

        // then
        expect(result.current.search).toBe("?sort=updated&tag=witch&page=2");
    });

    it("drops the param on the way back to the first page", () => {
        // given
        const { result } = renderSearchParamPage("/list?page=2&tab=everyone", { limit: 20 });

        // when
        act(() => result.current.pager.goPrev());

        // then
        expect(result.current.search).toBe("?tab=everyone");
        expect(result.current.pager.offset).toBe(0);
    });

    it("clamps the previous page at the first page", () => {
        // given
        const { result } = renderSearchParamPage("/list?page=1", { limit: 20 });

        // when
        act(() => result.current.pager.goPrev());

        // then
        expect(result.current.search).toBe("");
        expect(result.current.pager.offset).toBe(0);
    });

    it("walks past the end when asked to, because only the caller knows the total", () => {
        // given
        const { result } = renderSearchParamPage("/list?page=5", { limit: 20 });

        // when
        act(() => result.current.pager.goNext());

        // then
        expect(result.current.search).toBe("?page=6");
        expect(hasNextPage(result.current.pager, 100)).toBe(false);
    });

    it("drops the param on reset and keeps the filters", () => {
        // given
        const { result } = renderSearchParamPage("/list?page=3&q=beatrice", { limit: 20 });

        // when
        act(() => result.current.pager.reset());

        // then
        expect(result.current.search).toBe("?q=beatrice");
        expect(result.current.pager.offset).toBe(0);
    });

    it("lets the last move win when two are made in one tick, because react-router does not queue param updates", () => {
        // given
        const { result } = renderSearchParamPage("/list", { limit: 20 });

        // when
        act(() => {
            result.current.pager.goNext();
            result.current.pager.goNext();
        });

        // then
        expect(result.current.search).toBe("?page=2");
        expect(result.current.pager.offset).toBe(20);
    });
});
