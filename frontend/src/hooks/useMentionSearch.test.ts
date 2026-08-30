import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "../types/api";
import { type MentionSuggestion, useMentionSearch } from "./useMentionSearch";

const { fetchSearchUsers, reportClientError } = vi.hoisted(() => ({
    fetchSearchUsers: vi.fn(),
    reportClientError: vi.fn(),
}));

vi.mock("./queries/user", () => ({ fetchSearchUsers }));

vi.mock("../api/telemetry", () => ({ reportClientError }));

function makeCandidate(overrides: Partial<MentionSuggestion> = {}): MentionSuggestion {
    return {
        id: "u1",
        username: "beatrice",
        display_name: "Beatrice",
        viewer_follows: false,
        follows_viewer: false,
        ...overrides,
    };
}

beforeEach(() => {
    fetchSearchUsers.mockResolvedValue([]);
});

afterEach(() => {
    vi.resetAllMocks();
});

describe("useMentionSearch", () => {
    it("asks the search endpoint for the query it was handed", async () => {
        // given
        const { result } = renderHook(() => useMentionSearch());

        // when
        await result.current("beat");

        // then
        expect(fetchSearchUsers).toHaveBeenCalledWith("beat");
    });

    it("puts the people you both follow at the top", async () => {
        // given
        fetchSearchUsers.mockResolvedValue([
            makeCandidate({ id: "s1", username: "stranger" }),
            makeCandidate({ id: "s2", username: "fan", follows_viewer: true }),
            makeCandidate({ id: "s3", username: "mutual", viewer_follows: true, follows_viewer: true }),
            makeCandidate({ id: "s4", username: "idol", viewer_follows: true }),
        ]);
        const { result } = renderHook(() => useMentionSearch());

        // when
        const ranked = await result.current("s");

        // then
        expect(ranked.map(candidate => candidate.username)).toEqual(["mutual", "idol", "fan", "stranger"]);
    });

    it("leaves an already ordered list alone", async () => {
        // given
        const users: User[] = [makeCandidate({ id: "s1", username: "one" }), makeCandidate({ id: "s2" })];
        fetchSearchUsers.mockResolvedValue(users);
        const { result } = renderHook(() => useMentionSearch());

        // when
        const ranked = await result.current("s");

        // then
        expect(ranked.map(candidate => candidate.id)).toEqual(["s1", "s2"]);
    });

    it("hands back nobody when the search fails", async () => {
        // given
        fetchSearchUsers.mockRejectedValue(new Error("the server is asleep"));
        const { result } = renderHook(() => useMentionSearch());

        // when
        const ranked = await result.current("bea");

        // then
        expect(ranked).toEqual([]);
    });

    it("reports the failure instead of swallowing it", async () => {
        // given
        const thrown = new Error("the server is asleep");
        fetchSearchUsers.mockRejectedValue(thrown);
        const { result } = renderHook(() => useMentionSearch());

        // when
        await result.current("bea");

        // then
        expect(reportClientError).toHaveBeenCalledWith(thrown, { source: "recoverable" });
    });

    it("keeps the same callback across renders so a subscriber does not re-run", () => {
        // given
        const { result, rerender } = renderHook(() => useMentionSearch());
        const first = result.current;

        // when
        rerender();

        // then
        expect(result.current).toBe(first);
    });
});
