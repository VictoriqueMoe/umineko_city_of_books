import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { SiteInfoSecret } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { HuntIcon } from "./HuntIcon";

const mocks = vi.hoisted(() => ({ unlock: vi.fn() }));

vi.mock("../../api/mutations/secret", () => ({
    useUnlockSecret: () => ({ mutateAsync: mocks.unlock }),
}));

const ownerId = "11111111-1111-1111-1111-111111111111";
const secretId = "epitaph";

function makeSecret(overrides: Partial<SiteInfoSecret> = {}): SiteInfoSecret {
    return {
        id: secretId,
        title: "The Witch's Epitaph",
        description: "Seek the key that opens the golden land.",
        icon: "\u{1f52e}",
        solved: false,
        pieces: [
            { id: "piece-1", letter: "B", tile: 1 },
            { id: "piece-2", letter: "E", tile: 2 },
            { id: "piece-3", letter: "A", tile: 3 },
        ],
        ...overrides,
    };
}

interface SetupOptions {
    secret?: SiteInfoSecret;
    collected?: string[];
    profileUserId?: string;
    signedIn?: boolean;
}

function setup(options: SetupOptions = {}) {
    const secret = options.secret ?? makeSecret();
    const collected = new Set(options.collected ?? ["piece-1"]);

    return renderWithProviders(<HuntIcon profileUserId={options.profileUserId ?? ownerId} secret={secret} />, {
        user: options.signedIn === false ? null : makeUser({ id: ownerId }),
        siteInfo: { listed_secrets: [secret] },
        theme: { hasSecret: id => collected.has(id) },
    });
}

describe("HuntIcon", () => {
    it("stays hidden on somebody else's profile", () => {
        // given
        const profileUserId = "22222222-2222-2222-2222-222222222222";

        // when
        const { container } = setup({ profileUserId });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays hidden for a signed out visitor", () => {
        // given
        const signedIn = false;

        // when
        const { container } = setup({ signedIn });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays hidden once the hunt has been solved", () => {
        // given
        const collected = ["piece-1", "piece-2", "piece-3", secretId];

        // when
        const { container } = setup({ collected });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays hidden until the first piece is found", () => {
        // given
        const collected: string[] = [];

        // when
        const { container } = setup({ collected });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("announces how much of the hunt has been gathered", () => {
        // given
        const collected = ["piece-1", "piece-2"];

        // when
        setup({ collected });

        // then
        const icon = screen.getByRole("button", { name: "The Witch's Epitaph: 2 of 3 pieces" });
        expect(icon).toHaveAttribute("title", "The Witch's Epitaph - 2 / 3");
    });

    it("badges the icon with the count while pieces are still missing", () => {
        // given
        const collected = ["piece-1", "piece-2"];

        // when
        setup({ collected });

        // then
        expect(screen.getByRole("button").textContent).toBe("\u{1f52e}2");
    });

    it("drops the badge once every piece has been found", () => {
        // given
        const collected = ["piece-1", "piece-2", "piece-3"];

        // when
        setup({ collected });

        // then
        expect(screen.getByRole("button", { name: "The Witch's Epitaph: 3 of 3 pieces" }).textContent).toBe(
            "\u{1f52e}",
        );
    });

    it("falls back to a star when the hunt has no icon of its own", () => {
        // given
        const secret = makeSecret({ icon: "" });

        // when
        setup({ secret });

        // then
        expect(screen.getByRole("button").textContent).toBe("★1");
    });

    it("opens the hunt panel when the icon is pressed", async () => {
        // given
        const user = userEvent.setup();
        setup();
        expect(screen.queryByRole("heading", { name: "The Witch's Epitaph" })).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "The Witch's Epitaph: 1 of 3 pieces" }));

        // then
        expect(screen.getByRole("heading", { name: "The Witch's Epitaph" })).toBeInTheDocument();
    });

    it("closes the hunt panel again when the panel is dismissed", async () => {
        // given
        const user = userEvent.setup();
        setup();
        await user.click(screen.getByRole("button", { name: "The Witch's Epitaph: 1 of 3 pieces" }));

        // when
        await user.click(screen.getByRole("button", { name: "✕" }));

        // then
        expect(screen.queryByRole("heading", { name: "The Witch's Epitaph" })).not.toBeInTheDocument();
    });
});
