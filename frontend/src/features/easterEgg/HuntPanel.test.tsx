import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfoSecret } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { HuntPanel } from "./HuntPanel";

const mocks = vi.hoisted(() => ({ unlock: vi.fn() }));

vi.mock("../../api/mutations/secret", () => ({
    useUnlockSecret: () => ({ mutateAsync: mocks.unlock }),
}));

const addSecret = vi.fn();

const secretId = "epitaph";

function makeSecret(overrides: Partial<SiteInfoSecret> = {}): SiteInfoSecret {
    return {
        id: secretId,
        title: "The Witch's Epitaph",
        description: "Seek the key that opens the golden land.",
        pointer: "At the sweetfish river, the key is sleeping.",
        pending_hint: "The sparkles hide where the story is quiet.",
        ready_placeholder: "Whisper the name of the witch...",
        solved_message: "The golden land opens for you.",
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
    isOpen?: boolean;
    id?: string;
    onClose?: () => void;
}

function setup(options: SetupOptions = {}) {
    const secret = options.secret ?? makeSecret();
    const collected = new Set(options.collected ?? ["piece-1"]);

    return renderWithProviders(
        <HuntPanel
            secretId={options.id ?? secretId}
            isOpen={options.isOpen ?? true}
            onClose={options.onClose ?? noop}
        />,
        {
            user: makeUser(),
            siteInfo: { listed_secrets: [secret] },
            theme: { hasSecret: id => collected.has(id), addSecret },
        },
    );
}

function noop() {}

beforeEach(() => {
    mocks.unlock.mockResolvedValue(undefined);
});

describe("HuntPanel", () => {
    it("renders nothing when the site does not list the hunt", () => {
        // given
        const id = "no-such-hunt";

        // when
        const { container } = setup({ id });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("renders nothing while the panel is closed", () => {
        // given
        const isOpen = false;

        // when
        const { container } = setup({ isOpen });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("reveals the letter of every found tile and leaves the rest blank", () => {
        // given
        const collected = ["piece-1", "piece-3"];

        // when
        setup({ collected });

        // then
        expect(screen.getByLabelText("Tile 1: B")).toHaveTextContent("B");
        expect(screen.getByLabelText("Tile 2: empty")).toHaveTextContent("·");
        expect(screen.getByLabelText("Tile 3: A")).toHaveTextContent("A");
    });

    it("keeps the letters of a hunt whose tiles start above one", () => {
        // given
        const secret = makeSecret({
            pieces: [
                { id: "piece-2", letter: "E", tile: 2 },
                { id: "piece-3", letter: "A", tile: 3 },
            ],
        });

        // when
        setup({ secret, collected: ["piece-2", "piece-3"] });

        // then
        expect(screen.getByLabelText("Tile 2: E")).toHaveTextContent("E");
        expect(screen.getByLabelText("Tile 3: A")).toHaveTextContent("A");
    });

    it("numbers the tiles in reading order when the hunt numbers none of its pieces", () => {
        // given
        const secret = makeSecret({
            pieces: [
                { id: "piece-1", letter: "B" },
                { id: "piece-2", letter: "E" },
            ],
        });

        // when
        setup({ secret, collected: ["piece-1"] });

        // then
        expect(screen.getByLabelText("Tile 1: B")).toHaveTextContent("B");
        expect(screen.getByLabelText("Tile 2: empty")).toHaveTextContent("·");
    });

    it("counts the missing pieces in the field and blocks the whisper", () => {
        // given
        const collected = ["piece-1"];

        // when
        setup({ collected });

        // then
        expect(screen.getByPlaceholderText("1 / 3 pieces found")).toBeDisabled();
        expect(screen.getByRole("button", { name: "Declare" })).toBeDisabled();
    });

    it("shows the pointer and the pending hint while the hunt is unfinished", () => {
        // given
        const collected = ["piece-1"];

        // when
        setup({ collected });

        // then
        expect(screen.getByText("At the sweetfish river, the key is sleeping.")).toBeInTheDocument();
        expect(screen.getByText("The sparkles hide where the story is quiet.")).toBeInTheDocument();
    });

    it("invites the answer once every piece has been found", () => {
        // given
        const collected = ["piece-1", "piece-2", "piece-3"];

        // when
        setup({ collected });

        // then
        expect(screen.getByPlaceholderText("Whisper the name of the witch...")).toBeEnabled();
        expect(screen.queryByText("The sparkles hide where the story is quiet.")).not.toBeInTheDocument();
        expect(screen.getByText("At the sweetfish river, the key is sleeping.")).toBeInTheDocument();
    });

    it("falls back to a generic invitation when the hunt names no placeholder", () => {
        // given
        const secret = makeSecret({ ready_placeholder: undefined });

        // when
        setup({ secret, collected: ["piece-1", "piece-2", "piece-3"] });

        // then
        expect(screen.getByPlaceholderText("Whisper the answer...")).toBeInTheDocument();
    });

    it("keeps the declare control disabled for an answer of only whitespace", async () => {
        // given
        const user = userEvent.setup();
        setup({ collected: ["piece-1", "piece-2", "piece-3"] });

        // when
        await user.type(screen.getByPlaceholderText("Whisper the name of the witch..."), "   ");

        // then
        expect(screen.getByRole("button", { name: "Declare" })).toBeDisabled();
    });

    it("whispers the trimmed lowercased answer and clears the field when it is right", async () => {
        // given
        const user = userEvent.setup();
        setup({ collected: ["piece-1", "piece-2", "piece-3"] });
        const field = screen.getByPlaceholderText("Whisper the name of the witch...");

        // when
        await user.type(field, "  BeaTRICE  ");
        await user.click(screen.getByRole("button", { name: "Declare" }));

        // then
        expect(mocks.unlock).toHaveBeenCalledWith({ id: secretId, phrase: "beatrice" });
        expect(addSecret).toHaveBeenCalledWith(secretId);
        await waitFor(() => expect(field).toHaveValue(""));
    });

    it("nudges the reader and keeps the answer when the whisper is wrong", async () => {
        // given
        mocks.unlock.mockRejectedValue(new Error("not the golden truth"));
        const user = userEvent.setup();
        setup({ collected: ["piece-1", "piece-2", "piece-3"] });
        const field = screen.getByPlaceholderText("Whisper the name of the witch...");

        // when
        await user.type(field, "kinzo");
        await user.click(screen.getByRole("button", { name: "Declare" }));

        // then
        expect(await screen.findByText("Not quite. Read the riddle again.")).toBeInTheDocument();
        expect(field).toHaveValue("kinzo");
        expect(addSecret).not.toHaveBeenCalled();
    });

    it("locks the form while a whisper is still in flight", async () => {
        // given
        mocks.unlock.mockImplementation(() => new Promise(() => {}));
        const user = userEvent.setup();
        setup({ collected: ["piece-1", "piece-2", "piece-3"] });
        const field = screen.getByPlaceholderText("Whisper the name of the witch...");
        await user.type(field, "beatrice");

        // when
        await user.click(screen.getByRole("button", { name: "Declare" }));

        // then
        expect(field).toBeDisabled();
        expect(screen.getByRole("button", { name: "Declare" })).toBeDisabled();
        expect(mocks.unlock).toHaveBeenCalledOnce();
    });

    it("celebrates with the hunt's own message once the reward is held", () => {
        // given
        const collected = ["piece-1", "piece-2", "piece-3", secretId];

        // when
        setup({ collected });

        // then
        expect(screen.getByText("The golden land opens for you.")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Declare" })).not.toBeInTheDocument();
    });

    it("falls back to a generic celebration when the hunt has no solved message", () => {
        // given
        const secret = makeSecret({ solved_message: "" });

        // when
        setup({ secret, collected: [secretId] });

        // then
        expect(screen.getByText("You solved the hunt. The reward has been added to your profile.")).toBeInTheDocument();
    });

    it("explains that the hunt is closed when somebody else answered first", () => {
        // given
        const secret = makeSecret({ solved: true });

        // when
        setup({ secret, collected: ["piece-1", "piece-2"] });

        // then
        expect(
            screen.getByText(/Someone else whispered the answer before you.*2 \/ 3 pieces stay with you/),
        ).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Declare" })).not.toBeInTheDocument();
    });

    it("still shows the gathered tiles when the hunt is closed", () => {
        // given
        const secret = makeSecret({ solved: true });

        // when
        setup({ secret, collected: ["piece-2"] });

        // then
        expect(screen.getByLabelText("Tile 2: E")).toBeInTheDocument();
        expect(screen.queryByText("At the sweetfish river, the key is sleeping.")).not.toBeInTheDocument();
    });

    it("hands the dismissal back to the caller", async () => {
        // given
        const onClose = vi.fn();
        const user = userEvent.setup();
        setup({ onClose });

        // when
        await user.click(screen.getByRole("button", { name: "✕" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });
});
