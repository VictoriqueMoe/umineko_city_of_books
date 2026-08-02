import { act, fireEvent, screen } from "@testing-library/react";
import { useContext } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/render";
import { MentionResolverProvider } from "./MentionResolverContext";
import { MentionResolverContext } from "./mentionResolverContextValue";

const { resolveUsernames } = vi.hoisted(() => ({ resolveUsernames: vi.fn() }));

vi.mock("../api/endpoints", () => ({ resolveUsernames }));

const BATCH_WINDOW_MS = 60;

function Probe({ names }: { names: string[] }) {
    const ctx = useContext(MentionResolverContext);
    if (!ctx) {
        throw new Error("Probe needs a MentionResolverProvider");
    }

    return (
        <div>
            {names.map(name => (
                <button key={name} type="button" onClick={() => ctx.request(name)}>
                    {`request ${name}`}
                </button>
            ))}
            {names.map(name => (
                <p key={name}>{`${name} is ${String(ctx.isKnown(name))}`}</p>
            ))}
        </div>
    );
}

function renderProbe(names: string[]) {
    return renderWithProviders(
        <MentionResolverProvider>
            <Probe names={names} />
        </MentionResolverProvider>,
    );
}

function requestMention(name: string) {
    fireEvent.click(screen.getByRole("button", { name: `request ${name}` }));
}

async function settleBatch() {
    await act(async () => {
        vi.advanceTimersByTime(BATCH_WINDOW_MS);
    });
}

beforeEach(() => {
    vi.useFakeTimers();
    resolveUsernames.mockResolvedValue({ usernames: [] });
});

describe("MentionResolverProvider", () => {
    it("reports a mention as unresolved before any lookup has returned", () => {
        // given
        renderProbe(["beatrice"]);

        // when
        requestMention("beatrice");

        // then
        expect(screen.getByText("beatrice is undefined")).toBeInTheDocument();
        expect(resolveUsernames).not.toHaveBeenCalled();
    });

    it("collects every mention requested inside the batching window into a single lookup", async () => {
        // given
        renderProbe(["beatrice", "battler", "erika"]);

        // when
        requestMention("beatrice");
        requestMention("battler");
        requestMention("erika");
        await settleBatch();

        // then
        expect(resolveUsernames).toHaveBeenCalledOnce();
        expect(resolveUsernames).toHaveBeenCalledWith(["beatrice", "battler", "erika"]);
    });

    it("asks about a username only once however many times it is mentioned", async () => {
        // given
        renderProbe(["Beatrice", "beatrice"]);

        // when
        requestMention("Beatrice");
        requestMention("beatrice");
        requestMention("Beatrice");
        await settleBatch();

        // then
        expect(resolveUsernames).toHaveBeenCalledOnce();
        expect(resolveUsernames).toHaveBeenCalledWith(["beatrice"]);
    });

    it("does not look a username up again once it has been resolved", async () => {
        // given
        resolveUsernames.mockResolvedValue({ usernames: ["beatrice"] });
        renderProbe(["beatrice"]);
        requestMention("beatrice");
        await settleBatch();

        // when
        requestMention("beatrice");
        await settleBatch();

        // then
        expect(resolveUsernames).toHaveBeenCalledOnce();
    });

    it("marks the usernames the server recognised as known and the others as unknown", async () => {
        // given
        resolveUsernames.mockResolvedValue({ usernames: ["Beatrice"] });
        renderProbe(["beatrice", "battler"]);

        // when
        requestMention("beatrice");
        requestMention("battler");
        await settleBatch();

        // then
        expect(screen.getByText("beatrice is true")).toBeInTheDocument();
        expect(screen.getByText("battler is false")).toBeInTheDocument();
    });

    it("matches the server answer regardless of the casing of the mention", async () => {
        // given
        resolveUsernames.mockResolvedValue({ usernames: ["BEATRICE"] });
        renderProbe(["Beatrice"]);

        // when
        requestMention("Beatrice");
        await settleBatch();

        // then
        expect(screen.getByText("Beatrice is true")).toBeInTheDocument();
    });

    it("starts a fresh batch for mentions that arrive after the previous one flushed", async () => {
        // given
        renderProbe(["beatrice", "battler"]);
        requestMention("beatrice");
        await settleBatch();

        // when
        requestMention("battler");
        await settleBatch();

        // then
        expect(resolveUsernames).toHaveBeenCalledTimes(2);
        expect(resolveUsernames).toHaveBeenNthCalledWith(1, ["beatrice"]);
        expect(resolveUsernames).toHaveBeenNthCalledWith(2, ["battler"]);
    });

    it("lets a username be looked up again after the lookup failed", async () => {
        // given
        resolveUsernames.mockRejectedValueOnce(new Error("network is down"));
        renderProbe(["beatrice"]);
        requestMention("beatrice");
        await settleBatch();

        // when
        requestMention("beatrice");
        await settleBatch();

        // then
        expect(resolveUsernames).toHaveBeenCalledTimes(2);
        expect(screen.getByText("beatrice is false")).toBeInTheDocument();
    });

    it("abandons a pending lookup when the provider unmounts", async () => {
        // given
        const { unmount } = renderProbe(["beatrice"]);
        requestMention("beatrice");

        // when
        unmount();
        await settleBatch();

        // then
        expect(resolveUsernames).not.toHaveBeenCalled();
    });
});
