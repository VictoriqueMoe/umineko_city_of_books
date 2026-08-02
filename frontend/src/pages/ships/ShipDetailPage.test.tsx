import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { ShipCharacter, ShipDetail, UserProfile } from "../../types/api";
import { ShipDetailPage } from "./ShipDetailPage";

const mocks = vi.hoisted(() => ({
    useShip: vi.fn(),
    vote: vi.fn(),
    deleteShip: vi.fn(),
    updateShip: vi.fn(),
    navigate: vi.fn(),
    noop: vi.fn(),
}));

vi.mock("../../api/queries/ship", () => ({ useShip: mocks.useShip }));

vi.mock("../../api/mutations/ship", () => ({
    useVoteShip: () => ({ mutateAsync: mocks.vote }),
    useDeleteShip: () => ({ mutateAsync: mocks.deleteShip }),
    useUpdateShip: () => ({ mutateAsync: mocks.updateShip }),
    useLikeShipComment: () => ({ mutateAsync: mocks.noop }),
    useUnlikeShipComment: () => ({ mutateAsync: mocks.noop }),
    useDeleteShipComment: () => ({ mutateAsync: mocks.noop }),
    useUpdateShipComment: () => ({ mutateAsync: mocks.noop }),
    useCreateShipComment: () => ({ mutateAsync: mocks.noop }),
    useUploadShipCommentMedia: () => ({ mutateAsync: mocks.noop }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: ({ targetId, viewerBlocked }: { targetId: string; viewerBlocked?: boolean }) => (
        <div data-testid="comments" data-target={targetId} data-blocked={String(!!viewerBlocked)} />
    ),
}));

vi.mock("../../components/ShareButton/ShareButton", () => ({
    ShareButton: ({ contentId }: { contentId: string }) => <div data-testid="share" data-content={contentId} />,
}));

vi.mock("../../components/CharacterPicker/CharacterPicker", () => ({
    CharacterPicker: ({ onAdd }: { onAdd: (c: ShipCharacter) => void }) => (
        <button onClick={() => onAdd({ series: "oc", character_name: "Featherine Junior", sort_order: 0 })}>
            pick an OC
        </button>
    ),
}));

vi.mock("../../components/MentionTextArea/MentionTextArea", () => ({
    MentionTextArea: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
        <textarea aria-label="Why do you ship it" value={value} onChange={e => onChange(e.target.value)} />
    ),
}));

const author = { id: "author-1", username: "ronove", display_name: "Ronove" };

function makeShip(overrides: Partial<ShipDetail> = {}): ShipDetail {
    return {
        id: "ship-1",
        author,
        title: "Battler and Beatrice",
        description: "the golden witch and her opponent",
        characters: [
            { series: "umineko", character_id: "battler", character_name: "Battler", sort_order: 0 },
            { series: "umineko", character_id: "beatrice", character_name: "Beatrice", sort_order: 1 },
        ],
        vote_score: 5,
        comment_count: 0,
        is_crackship: false,
        created_at: "2026-07-01T10:00:00Z",
        comments: [],
        viewer_blocked: false,
        ...overrides,
    };
}

interface ShipState {
    ship?: ShipDetail | null;
    loading?: boolean;
}

function stubShip(state: ShipState = {}) {
    const refresh = vi.fn(() => Promise.resolve());
    mocks.useShip.mockReturnValue({
        ship: state.ship === undefined ? makeShip() : state.ship,
        loading: state.loading ?? false,
        refresh,
    });
    return { refresh };
}

function renderPage(viewer: UserProfile | null = makeUser({ id: "author-1" }), route = "/ships/ship-1") {
    return renderWithProviders(<ShipDetailPage />, { user: viewer, route, path: "/ships/:id" });
}

beforeEach(() => {
    mocks.vote.mockResolvedValue({});
    mocks.deleteShip.mockResolvedValue({});
    mocks.updateShip.mockResolvedValue({});
});

describe("ShipDetailPage loading and missing states", () => {
    it("says it is loading while the ship is on its way", () => {
        // given
        stubShip({ ship: null, loading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading ship...")).toBeInTheDocument();
    });

    it("says the ship could not be found once the fetch has settled", () => {
        // given
        stubShip({ ship: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Ship not found.")).toBeInTheDocument();
    });
});

describe("ShipDetailPage content", () => {
    it("shows the title, the pairing and the reasoning", () => {
        // given
        stubShip();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Battler and Beatrice" })).toBeInTheDocument();
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("the golden witch and her opponent")).toBeInTheDocument();
    });

    it("marks a positive score with a plus sign", () => {
        // given
        stubShip({ ship: makeShip({ vote_score: 5 }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("+5")).toBeInTheDocument();
    });

    it("brands a low scoring pairing as a crackship", () => {
        // given
        stubShip({ ship: makeShip({ is_crackship: true }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("Crackship")).toBeInTheDocument();
    });

    it("hands the ship's own comments to the comments section", () => {
        // given
        stubShip({ ship: makeShip({ id: "ship-7", viewer_blocked: true }) });

        // when
        renderPage();

        // then
        expect(screen.getByTestId("comments")).toHaveAttribute("data-target", "ship-7");
        expect(screen.getByTestId("comments")).toHaveAttribute("data-blocked", "true");
    });

    it("shows no image at all when the ship has none", () => {
        // given
        stubShip();

        // when
        renderPage();

        // then
        expect(screen.queryByRole("img", { name: "Battler and Beatrice" })).not.toBeInTheDocument();
    });

    it("opens the lightbox when the ship image is clicked", async () => {
        // given
        stubShip({ ship: makeShip({ image_url: "/full.png" }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("img", { name: "Battler and Beatrice" }));

        // then
        expect(screen.getByRole("dialog")).toBeInTheDocument();
    });

    it("returns to the ship list from the back link", async () => {
        // given
        stubShip();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("← All Ships"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/ships");
    });
});

describe("ShipDetailPage voting", () => {
    it("locks the vote buttons for a signed out visitor", () => {
        // given
        stubShip();

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("button", { name: "△" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "▽" })).toBeDisabled();
    });

    it("casts an upvote for a member who has not voted yet", async () => {
        // given
        stubShip();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "△" }));

        // then
        expect(mocks.vote).toHaveBeenCalledWith(1);
    });

    it("casts a downvote for a member who has not voted yet", async () => {
        // given
        stubShip();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "▽" }));

        // then
        expect(mocks.vote).toHaveBeenCalledWith(-1);
    });

    it("clears an existing upvote when the same arrow is pressed again", async () => {
        // given
        stubShip({ ship: makeShip({ user_vote: 1 }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "▲" }));

        // then
        expect(mocks.vote).toHaveBeenCalledWith(0);
    });

    it("flips an existing upvote straight to a downvote", async () => {
        // given
        stubShip({ ship: makeShip({ user_vote: 1 }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "▽" }));

        // then
        expect(mocks.vote).toHaveBeenCalledWith(-1);
    });

    it("survives a vote the server rejects", async () => {
        // given
        mocks.vote.mockRejectedValue(new Error("too many votes"));
        stubShip();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "△" }));

        // then
        await waitFor(() => {
            expect(screen.getByRole("button", { name: "△" })).toBeEnabled();
        });
    });
});

describe("ShipDetailPage ownership and moderation", () => {
    it("hides edit and delete from a member who did not declare the ship", () => {
        // given
        stubShip();

        // when
        renderPage(makeUser({ id: "someone-else" }));

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("hides edit and delete from a signed out visitor", () => {
        // given
        stubShip();

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("offers edit and delete to the member who declared the ship", () => {
        // given
        stubShip();

        // when
        renderPage(makeUser({ id: "author-1" }));

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("offers edit and delete to a moderator who did not declare the ship", () => {
        // given
        stubShip();

        // when
        renderPage(makeUser({ id: "mod-1", role: "moderator" }));

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });
});

describe("ShipDetailPage deleting", () => {
    it("asks before deleting the ship", async () => {
        // given
        stubShip();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this ship? This cannot be undone.");
        expect(mocks.deleteShip).not.toHaveBeenCalled();
    });

    it("deletes the ship and returns to the list once confirmed", async () => {
        // given
        stubShip();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(mocks.deleteShip).toHaveBeenCalledWith("ship-1");
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/ships");
        });
    });
});

describe("ShipDetailPage editing", () => {
    async function openEditor() {
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "Edit" }));

        return user;
    }

    it("fills the editor with the ship as it stands", async () => {
        // given
        stubShip();

        // when
        await openEditor();

        // then
        expect(screen.getByDisplayValue("Battler and Beatrice")).toBeInTheDocument();
        expect(screen.getByLabelText("Why do you ship it")).toHaveValue("the golden witch and her opponent");
    });

    it("saves the trimmed title, description and characters", async () => {
        // given
        stubShip();
        const user = await openEditor();
        const titleBox = screen.getByDisplayValue("Battler and Beatrice");
        await user.clear(titleBox);
        await user.type(titleBox, "  Battler and the Golden Witch  ");

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.updateShip).toHaveBeenCalledWith({
            title: "Battler and the Golden Witch",
            description: "the golden witch and her opponent",
            characters: [
                { series: "umineko", character_id: "battler", character_name: "Battler", sort_order: 0 },
                { series: "umineko", character_id: "beatrice", character_name: "Beatrice", sort_order: 1 },
            ],
        });
    });

    it("refreshes the ship once the edit has been saved", async () => {
        // given
        const { refresh } = stubShip();
        const user = await openEditor();

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("adds a character to the pairing while editing", async () => {
        // given
        stubShip();
        const user = await openEditor();
        await user.click(screen.getByRole("button", { name: "pick an OC" }));

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.updateShip).toHaveBeenCalledWith(
            expect.objectContaining({
                characters: expect.arrayContaining([
                    { series: "oc", character_name: "Featherine Junior", sort_order: 2 },
                ]),
            }),
        );
    });

    it("locks saving once the pairing drops below two characters", async () => {
        // given
        stubShip();
        const user = await openEditor();

        // when
        await user.click(screen.getAllByRole("button", { name: "Remove character" })[0]);

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    it("reports why the edit could not be saved", async () => {
        // given
        mocks.updateShip.mockRejectedValue(new Error("the witch forbids it"));
        stubShip();
        const user = await openEditor();

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(await screen.findByText("the witch forbids it")).toBeInTheDocument();
    });

    it("falls back to a generic message when the failure carries no reason", async () => {
        // given
        mocks.updateShip.mockRejectedValue("boom");
        stubShip();
        const user = await openEditor();

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(await screen.findByText("Failed to update ship")).toBeInTheDocument();
    });

    it("throws the editor away when the edit is cancelled", async () => {
        // given
        stubShip();
        const user = await openEditor();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByLabelText("Why do you ship it")).not.toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Battler and Beatrice" })).toBeInTheDocument();
        expect(mocks.updateShip).not.toHaveBeenCalled();
    });
});
