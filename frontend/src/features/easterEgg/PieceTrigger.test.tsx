import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfoSecret } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { PieceTrigger } from "./PieceTrigger";

const mocks = vi.hoisted(() => ({ unlock: vi.fn() }));

vi.mock("../../api/mutations/secret", () => ({
    useUnlockSecret: () => ({ mutateAsync: mocks.unlock }),
}));

const addSecret = vi.fn();

function makeSecret(overrides: Partial<SiteInfoSecret> = {}): SiteInfoSecret {
    return {
        id: "epitaph",
        title: "Witch's Epitaph",
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
    pieceId?: string;
    ariaLabel?: string;
    secrets?: SiteInfoSecret[];
    collected?: string[];
    signedIn?: boolean;
}

function setup(options: SetupOptions = {}) {
    const collected = new Set(options.collected ?? []);

    return renderWithProviders(<PieceTrigger pieceId={options.pieceId ?? "piece-1"} ariaLabel={options.ariaLabel} />, {
        user: options.signedIn === false ? null : makeUser(),
        siteInfo: { listed_secrets: options.secrets ?? [makeSecret()] },
        theme: { hasSecret: id => collected.has(id), addSecret },
    });
}

beforeEach(() => {
    mocks.unlock.mockResolvedValue(undefined);
});

describe("PieceTrigger", () => {
    it("stays hidden from a signed out visitor", () => {
        // given
        const signedIn = false;

        // when
        const { container } = setup({ signedIn });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays hidden when no listed hunt owns the piece", () => {
        // given
        const pieceId = "piece-of-nothing";

        // when
        const { container } = setup({ pieceId });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays hidden once the hunt has been solved by somebody", () => {
        // given
        const secrets = [makeSecret({ solved: true })];

        // when
        const { container } = setup({ secrets });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays hidden for a piece the visitor already holds", () => {
        // given
        const collected = ["piece-1"];

        // when
        const { container } = setup({ collected });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("describes itself as a curious sparkle by default", () => {
        // given
        const ariaLabel = undefined;

        // when
        setup({ ariaLabel });

        // then
        expect(screen.getByRole("button", { name: "A curious sparkle" })).toBeInTheDocument();
    });

    it("takes the description the caller gives it", () => {
        // given
        const ariaLabel = "A glimmer behind the portrait";

        // when
        setup({ ariaLabel });

        // then
        expect(screen.getByRole("button", { name: ariaLabel })).toBeInTheDocument();
    });

    it("collects the piece and whispers which hunt it belongs to", async () => {
        // given
        const user = userEvent.setup();
        setup();

        // when
        await user.click(screen.getByRole("button", { name: "A curious sparkle" }));

        // then
        expect(mocks.unlock).toHaveBeenCalledWith({ id: "piece-1", phrase: "piece-1_b" });
        expect(addSecret).toHaveBeenCalledWith("piece-1");
        expect(screen.getByRole("status")).toHaveTextContent("Uu~ a piece of the Witch's Epitaph.");
    });

    it("announces the finished hunt when the last piece is collected", async () => {
        // given
        const user = userEvent.setup();
        setup({ collected: ["piece-2"] });

        // when
        await user.click(screen.getByRole("button", { name: "A curious sparkle" }));

        // then
        expect(screen.getByRole("status")).toHaveTextContent(
            "Uu~ all 2 pieces of Witch's Epitaph are yours. Read the riddle again, then open the trophy on your profile to whisper the answer.",
        );
    });

    it("says nothing when the unlock request fails", async () => {
        // given
        mocks.unlock.mockRejectedValue(new Error("the golden land is closed"));
        const user = userEvent.setup();
        setup();

        // when
        await user.click(screen.getByRole("button", { name: "A curious sparkle" }));

        // then
        expect(screen.queryByRole("status")).not.toBeInTheDocument();
        expect(addSecret).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "A curious sparkle" })).toBeEnabled();
    });

    it("stops a second press while the piece is still being collected", async () => {
        // given
        mocks.unlock.mockImplementation(() => new Promise(() => {}));
        const user = userEvent.setup();
        setup();

        // when
        await user.click(screen.getByRole("button", { name: "A curious sparkle" }));

        // then
        expect(screen.getByRole("button", { name: "A curious sparkle" })).toBeDisabled();
        expect(mocks.unlock).toHaveBeenCalledOnce();
    });

    it("lets the whisper fade away on its own", async () => {
        // given
        vi.useFakeTimers();
        setup();
        await act(async () => {
            fireEvent.click(screen.getByRole("button", { name: "A curious sparkle" }));
        });
        expect(screen.getByRole("status")).toBeInTheDocument();

        // when
        act(() => {
            vi.advanceTimersByTime(4200);
        });

        // then
        expect(screen.queryByRole("status")).not.toBeInTheDocument();
    });
});
