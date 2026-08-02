import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfoSecret } from "../api/endpoints";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import type { UserProfile } from "../types/api";
import { useHuntState } from "./useHuntState";

const mocks = vi.hoisted(() => ({ unlock: vi.fn() }));

vi.mock("../api/mutations/secret", () => ({
    useUnlockSecret: () => ({ mutateAsync: mocks.unlock }),
}));

const addSecret = vi.fn();

const secretId = "epitaph";

function makeSecret(overrides: Partial<SiteInfoSecret> = {}): SiteInfoSecret {
    return {
        id: secretId,
        title: "The Witch's Epitaph",
        description: "Seek the key that opens the golden land.",
        solved: false,
        pieces: [
            { id: "piece-1", letter: "B", tile: 1 },
            { id: "piece-2", letter: "E", tile: 2 },
        ],
        ...overrides,
    };
}

interface SetupOptions {
    id?: string;
    secrets?: SiteInfoSecret[];
    collected?: string[];
    user?: UserProfile | null;
}

function setup(options: SetupOptions = {}) {
    const collected = new Set(options.collected ?? []);

    return renderHook(() => useHuntState(options.id ?? secretId), {
        wrapper: providerWrapper({
            user: options.user === undefined ? makeUser() : options.user,
            siteInfo: { listed_secrets: options.secrets ?? [makeSecret()] },
            theme: { hasSecret: id => collected.has(id), addSecret },
        }),
    });
}

beforeEach(() => {
    mocks.unlock.mockResolvedValue(undefined);
});

describe("useHuntState derived state", () => {
    it("reports an empty hunt when the site does not list the secret", () => {
        // given
        const id = "no-such-hunt";

        // when
        const { result } = setup({ id });

        // then
        expect(result.current.secret).toBeNull();
        expect(result.current.totalPieces).toBe(0);
        expect(result.current.collectedCount).toBe(0);
        expect(result.current.allPiecesCollected).toBe(false);
    });

    it("counts only the pieces of this hunt that the visitor already holds", () => {
        // given
        const collected = ["piece-1", "a-piece-of-another-hunt"];

        // when
        const { result } = setup({ collected });

        // then
        expect(result.current.collectedCount).toBe(1);
        expect(result.current.collectedPieces.has("piece-1")).toBe(true);
        expect(result.current.collectedPieces.has("a-piece-of-another-hunt")).toBe(false);
        expect(result.current.totalPieces).toBe(2);
        expect(result.current.allPiecesCollected).toBe(false);
    });

    it("declares the hunt ready once every piece has been collected", () => {
        // given
        const collected = ["piece-1", "piece-2"];

        // when
        const { result } = setup({ collected });

        // then
        expect(result.current.allPiecesCollected).toBe(true);
        expect(result.current.solved).toBe(false);
        expect(result.current.closed).toBe(false);
    });

    it("never declares a pieceless hunt ready", () => {
        // given
        const secrets = [makeSecret({ pieces: [] })];

        // when
        const { result } = setup({ secrets });

        // then
        expect(result.current.totalPieces).toBe(0);
        expect(result.current.allPiecesCollected).toBe(false);
    });

    it("treats the hunt as solved when the visitor holds the reward itself", () => {
        // given
        const collected = [secretId];

        // when
        const { result } = setup({ collected });

        // then
        expect(result.current.solved).toBe(true);
        expect(result.current.closed).toBe(false);
    });

    it("treats the hunt as closed when somebody else answered first", () => {
        // given
        const secrets = [makeSecret({ solved: true })];

        // when
        const { result } = setup({ secrets });

        // then
        expect(result.current.solved).toBe(false);
        expect(result.current.closed).toBe(true);
    });

    it("keeps a hunt the visitor solved out of the closed state", () => {
        // given
        const secrets = [makeSecret({ solved: true })];

        // when
        const { result } = setup({ secrets, collected: [secretId] });

        // then
        expect(result.current.solved).toBe(true);
        expect(result.current.closed).toBe(false);
    });
});

describe("useHuntState collectPiece", () => {
    it("unlocks a new piece with the phrase derived from its letter and stores it", async () => {
        // given
        const { result } = setup();

        // when
        const outcome = await result.current.collectPiece("piece-1");

        // then
        expect(outcome).toBe("new");
        expect(mocks.unlock).toHaveBeenCalledWith({ id: "piece-1", phrase: "piece-1_b" });
        expect(addSecret).toHaveBeenCalledWith("piece-1");
    });

    it("ignores a piece the visitor has already collected", async () => {
        // given
        const { result } = setup({ collected: ["piece-1"] });

        // when
        const outcome = await result.current.collectPiece("piece-1");

        // then
        expect(outcome).toBe("already");
        expect(mocks.unlock).not.toHaveBeenCalled();
        expect(addSecret).not.toHaveBeenCalled();
    });

    it("refuses to collect a piece once the hunt has been closed", async () => {
        // given
        const { result } = setup({ secrets: [makeSecret({ solved: true })] });

        // when
        const outcome = await result.current.collectPiece("piece-1");

        // then
        expect(outcome).toBe("closed");
        expect(mocks.unlock).not.toHaveBeenCalled();
    });

    it("refuses to collect anything for a signed out visitor", async () => {
        // given
        const { result } = setup({ user: null });

        // when
        const outcome = await result.current.collectPiece("piece-1");

        // then
        expect(outcome).toBe("error");
        expect(mocks.unlock).not.toHaveBeenCalled();
    });

    it("refuses to collect for a hunt the site does not list", async () => {
        // given
        const { result } = setup({ id: "no-such-hunt" });

        // when
        const outcome = await result.current.collectPiece("piece-1");

        // then
        expect(outcome).toBe("error");
        expect(mocks.unlock).not.toHaveBeenCalled();
    });

    it("refuses a piece that does not belong to the hunt", async () => {
        // given
        const { result } = setup();

        // when
        const outcome = await result.current.collectPiece("piece-of-nothing");

        // then
        expect(outcome).toBe("error");
        expect(mocks.unlock).not.toHaveBeenCalled();
    });

    it("refuses a piece that carries no letter", async () => {
        // given
        const secrets = [makeSecret({ pieces: [{ id: "piece-1", tile: 1 }] })];
        const { result } = setup({ secrets });

        // when
        const outcome = await result.current.collectPiece("piece-1");

        // then
        expect(outcome).toBe("error");
        expect(mocks.unlock).not.toHaveBeenCalled();
    });

    it("reports an error and stores nothing when the unlock request fails", async () => {
        // given
        mocks.unlock.mockRejectedValue(new Error("the golden land is closed"));
        const { result } = setup();

        // when
        const outcome = await result.current.collectPiece("piece-1");

        // then
        expect(outcome).toBe("error");
        expect(addSecret).not.toHaveBeenCalled();
    });
});

describe("useHuntState attemptAnswer", () => {
    it("whispers the phrase against the hunt itself and stores the reward", async () => {
        // given
        const { result } = setup();

        // when
        const accepted = await result.current.attemptAnswer("golden truth");

        // then
        expect(accepted).toBe(true);
        expect(mocks.unlock).toHaveBeenCalledWith({ id: secretId, phrase: "golden truth" });
        expect(addSecret).toHaveBeenCalledWith(secretId);
    });

    it("rejects the answer and stores nothing when the request fails", async () => {
        // given
        mocks.unlock.mockRejectedValue(new Error("not quite"));
        const { result } = setup();

        // when
        const accepted = await result.current.attemptAnswer("wrong");

        // then
        expect(accepted).toBe(false);
        expect(addSecret).not.toHaveBeenCalled();
    });

    it("refuses an answer for a hunt somebody else has already solved", async () => {
        // given
        const { result } = setup({ secrets: [makeSecret({ solved: true })] });

        // when
        const accepted = await result.current.attemptAnswer("golden truth");

        // then
        expect(accepted).toBe(false);
        expect(mocks.unlock).not.toHaveBeenCalled();
    });

    it("refuses an answer for a hunt the site does not list", async () => {
        // given
        const { result } = setup({ id: "no-such-hunt" });

        // when
        const accepted = await result.current.attemptAnswer("golden truth");

        // then
        expect(accepted).toBe(false);
        expect(mocks.unlock).not.toHaveBeenCalled();
    });
});
