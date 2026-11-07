import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { UserProfile } from "../types/api";
import { type EditableResource, useEditable } from "./useEditable";

const mocks = vi.hoisted(() => ({
    useMystery: vi.fn(),
    useTheory: vi.fn(),
    useOC: vi.fn(),
    useFanfic: vi.fn(),
    useJournal: vi.fn(),
}));

vi.mock("./queries/mystery", () => ({ useMystery: mocks.useMystery }));
vi.mock("./queries/theory", () => ({ useTheory: mocks.useTheory }));
vi.mock("./queries/oc", () => ({ useOC: mocks.useOC }));
vi.mock("./queries/fanfic", () => ({ useFanfic: mocks.useFanfic }));
vi.mock("./queries/journal", () => ({ useJournal: mocks.useJournal }));

const author = makeUser({ id: "author-1", username: "beatrice", display_name: "Beatrice" });
const stranger = makeUser({ id: "stranger-1", username: "battler", display_name: "Battler" });
const moderator = makeUser({ id: "mod-1", username: "ronove", display_name: "Ronove", role: "moderator" });

const RESOURCES: EditableResource[] = ["mystery", "theory", "oc", "fanfic", "journal"];

type StubOptions = {
    loading?: boolean;
    exists?: boolean;
};

function stubQueries(options: StubOptions = {}) {
    const loading = options.loading ?? false;
    const item = (options.exists ?? true) ? { author: { id: author.id } } : null;

    mocks.useMystery.mockImplementation((id: string) => ({ mystery: id ? item : null, loading: !!id && loading }));
    mocks.useTheory.mockImplementation((id: string) => ({ theory: id ? item : null, loading: !!id && loading }));
    mocks.useOC.mockImplementation((id: string) => ({ oc: id ? item : null, loading: !!id && loading }));
    mocks.useFanfic.mockImplementation((id: string) => ({ fanfic: id ? item : null, loading: !!id && loading }));
    mocks.useJournal.mockImplementation((id: string) => ({ journal: id ? item : null, loading: !!id && loading }));
}

function renderEditable(resource: EditableResource, user: UserProfile | null) {
    return renderHook(() => useEditable(resource), {
        wrapper: providerWrapper({ user, route: "/thing/item-1/edit", path: "/thing/:id/edit" }),
    });
}

describe("useEditable", () => {
    for (const resource of RESOURCES) {
        it(`lets the author edit their own ${resource}`, () => {
            // given
            stubQueries();

            // when
            const { result } = renderEditable(resource, author);

            // then
            expect(result.current).toEqual({ loading: false, allowed: true });
        });

        it(`refuses a ${resource} to somebody who did not write it`, () => {
            // given
            stubQueries();

            // when
            const { result } = renderEditable(resource, stranger);

            // then
            expect(result.current).toEqual({ loading: false, allowed: false });
        });

        it(`refuses a ${resource} that does not exist, even to a moderator`, () => {
            // given
            stubQueries({ exists: false });

            // when
            const { result } = renderEditable(resource, moderator);

            // then
            expect(result.current).toEqual({ loading: false, allowed: false });
        });

        it(`holds the decision on a ${resource} while it is loading`, () => {
            // given
            stubQueries({ loading: true, exists: false });

            // when
            const { result } = renderEditable(resource, author);

            // then
            expect(result.current).toEqual({ loading: true, allowed: false });
        });
    }

    const moderatorCases: { resource: EditableResource; allowed: boolean }[] = [
        { resource: "mystery", allowed: true },
        { resource: "theory", allowed: true },
        { resource: "fanfic", allowed: true },
        { resource: "oc", allowed: true },
        { resource: "journal", allowed: false },
    ];

    for (const tc of moderatorCases) {
        it(`${tc.allowed ? "lets" : "refuses"} a moderator on somebody else's ${tc.resource}`, () => {
            // given
            stubQueries();

            // when
            const { result } = renderEditable(tc.resource, moderator);

            // then
            expect(result.current.allowed).toBe(tc.allowed);
        });
    }

    it("asks only the named resource's query for the id in the route", () => {
        // given
        stubQueries();

        // when
        renderEditable("journal", author);

        // then
        expect(mocks.useJournal).toHaveBeenCalledWith("item-1");
        expect(mocks.useMystery).toHaveBeenCalledWith("");
        expect(mocks.useTheory).toHaveBeenCalledWith("");
        expect(mocks.useOC).toHaveBeenCalledWith("");
        expect(mocks.useFanfic).toHaveBeenCalledWith("");
        expect(mocks.useMystery).not.toHaveBeenCalledWith("item-1");
    });
});
