import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { ArtDetail, UserProfile } from "../../types/api";
import { ArtDetailPage } from "./ArtDetailPage";

const mocks = vi.hoisted(() => ({
    useArt: vi.fn(),
    likeArt: vi.fn(),
    unlikeArt: vi.fn(),
    deleteArt: vi.fn(),
    updateArt: vi.fn(),
    navigate: vi.fn(),
    noop: vi.fn(),
}));

vi.mock("../../api/queries/art", () => ({ useArt: mocks.useArt }));

vi.mock("../../api/mutations/art", () => ({
    useLikeArt: () => ({ mutateAsync: mocks.likeArt }),
    useUnlikeArt: () => ({ mutateAsync: mocks.unlikeArt }),
    useDeleteArt: () => ({ mutateAsync: mocks.deleteArt }),
    useUpdateArt: () => ({ mutateAsync: mocks.updateArt }),
    useCreateArtComment: () => ({ mutateAsync: mocks.noop }),
    useUpdateArtComment: () => ({ mutateAsync: mocks.noop }),
    useDeleteArtComment: () => ({ mutateAsync: mocks.noop }),
    useLikeArtComment: () => ({ mutateAsync: mocks.noop }),
    useUnlikeArtComment: () => ({ mutateAsync: mocks.noop }),
    useUploadArtCommentMedia: () => ({ mutateAsync: mocks.noop }),
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

vi.mock("../../components/ReportButton/ReportButton", () => ({
    ReportButton: ({ targetId }: { targetId: string }) => <div data-testid="report" data-target={targetId} />,
}));

vi.mock("../../components/art/TagInput/TagInput", () => ({
    TagInput: ({ tags, onChange }: { tags: string[]; onChange: (t: string[]) => void }) => (
        <button onClick={() => onChange([...tags, "witches"])}>add the witches tag</button>
    ),
}));

vi.mock("../../components/MentionTextArea/MentionTextArea", () => ({
    MentionTextArea: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
        <textarea aria-label="Description" value={value} onChange={e => onChange(e.target.value)} />
    ),
}));

const author = { id: "author-1", username: "ronove", display_name: "Ronove" };

function makeArt(overrides: Partial<ArtDetail> = {}): ArtDetail {
    return {
        id: "art-1",
        author,
        corner: "general",
        art_type: "drawing",
        title: "Beatrice at dusk",
        description: "drawn in the rose garden",
        image_url: "/art-1-full.png",
        thumbnail_url: "/art-1-thumb.png",
        tags: [],
        like_count: 4,
        comment_count: 0,
        view_count: 17,
        user_liked: false,
        is_spoiler: false,
        created_at: "2026-07-01T10:00:00Z",
        comments: [],
        liked_by: [],
        viewer_blocked: false,
        ...overrides,
    };
}

interface ArtState {
    art?: ArtDetail | null;
    loading?: boolean;
}

function stubArt(state: ArtState = {}) {
    const refresh = vi.fn(() => Promise.resolve());
    mocks.useArt.mockReturnValue({
        art: state.art === undefined ? makeArt() : state.art,
        loading: state.loading ?? false,
        refresh,
    });
    return { refresh };
}

function renderPage(viewer: UserProfile | null = makeUser({ id: "member-1" }), route = "/gallery/art/art-1") {
    return renderWithProviders(<ArtDetailPage />, { user: viewer, route, path: "/gallery/art/:id" });
}

beforeEach(() => {
    mocks.likeArt.mockResolvedValue({});
    mocks.unlikeArt.mockResolvedValue({});
    mocks.deleteArt.mockResolvedValue({});
    mocks.updateArt.mockResolvedValue({});
});

describe("ArtDetailPage loading and missing states", () => {
    it("says it is loading while the piece is on its way", () => {
        // given
        stubArt({ art: null, loading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading art...")).toBeInTheDocument();
    });

    it("says the piece could not be found once the fetch has settled", () => {
        // given
        stubArt({ art: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Art not found.")).toBeInTheDocument();
    });
});

describe("ArtDetailPage content", () => {
    it("shows the title, the description and the counts", () => {
        // given
        stubArt();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Beatrice at dusk" })).toBeInTheDocument();
        expect(screen.getByText("drawn in the rose garden")).toBeInTheDocument();
        expect(screen.getByText("♥ 4")).toBeInTheDocument();
        expect(screen.getByText("👁 17")).toBeInTheDocument();
    });

    it("shows the upload date in day month year order", () => {
        // given
        stubArt({ art: makeArt({ created_at: "2026-07-01T10:00:00Z" }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("1 Jul 2026")).toBeInTheDocument();
    });

    it("marks a piece that has been edited since it was uploaded", () => {
        // given
        stubArt({ art: makeArt({ updated_at: "2026-07-05T10:00:00Z" }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("(edited)")).toBeInTheDocument();
    });

    it("leaves an untouched piece unmarked", () => {
        // given
        stubArt();

        // when
        renderPage();

        // then
        expect(screen.queryByText("(edited)")).not.toBeInTheDocument();
    });

    it("shows every tag the piece carries", () => {
        // given
        stubArt({ art: makeArt({ tags: ["beatrice", "sunset"] }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("#beatrice")).toBeInTheDocument();
        expect(screen.getByText("#sunset")).toBeInTheDocument();
    });

    it("filters the gallery by a tag that was clicked", async () => {
        // given
        stubArt({ art: makeArt({ tags: ["golden butterflies"] }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("#golden butterflies"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/gallery?tag=golden%20butterflies");
    });

    it("takes the visitor to the artist's profile", async () => {
        // given
        stubArt();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("More by Ronove"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/user/ronove");
    });

    it("offers no gallery link for a piece that belongs to no gallery", () => {
        // given
        stubArt();

        // when
        renderPage();

        // then
        expect(screen.queryByText("View gallery")).not.toBeInTheDocument();
    });

    it("takes the visitor to the gallery the piece belongs to", async () => {
        // given
        stubArt({ art: makeArt({ gallery_id: "gallery-3" }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("View gallery"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/gallery/view/gallery-3");
    });

    it("lists the people who liked the piece", () => {
        // given
        stubArt({
            art: makeArt({ liked_by: [{ id: "fan-1", username: "battler", display_name: "Battler" }] }),
        });

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Liked by (1)" })).toBeInTheDocument();
        expect(screen.getByText("Battler")).toBeInTheDocument();
    });

    it("leaves the liked by section out when nobody has liked the piece", () => {
        // given
        stubArt();

        // when
        renderPage();

        // then
        expect(screen.queryByRole("heading", { name: /Liked by/ })).not.toBeInTheDocument();
    });

    it("hands the piece's own comments to the comments section", () => {
        // given
        stubArt({ art: makeArt({ id: "art-7", viewer_blocked: true }) });

        // when
        renderPage();

        // then
        expect(screen.getByTestId("comments")).toHaveAttribute("data-target", "art-7");
        expect(screen.getByTestId("comments")).toHaveAttribute("data-blocked", "true");
    });

    it("goes back the way the visitor came", async () => {
        // given
        stubArt();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("← Back to Gallery"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith(-1);
    });

    it("blurs a piece marked as a spoiler until it is revealed", async () => {
        // given
        stubArt({ art: makeArt({ is_spoiler: true }) });
        const user = userEvent.setup();
        renderPage();
        expect(screen.getByText("Click to reveal")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("img", { name: "Beatrice at dusk" }));

        // then
        expect(screen.queryByText("Click to reveal")).not.toBeInTheDocument();
    });

    it("opens the lightbox when a piece that is not a spoiler is clicked", async () => {
        // given
        stubArt();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("img", { name: "Beatrice at dusk" }));

        // then
        expect(screen.getAllByRole("img", { name: "Beatrice at dusk" })).toHaveLength(2);
    });
});

describe("ArtDetailPage liking", () => {
    it("locks the like button for a signed out visitor", () => {
        // given
        stubArt();

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("button", { name: "♥ 4" })).toBeDisabled();
    });

    it("likes a piece the member has not liked yet", async () => {
        // given
        stubArt();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "♥ 4" }));

        // then
        expect(mocks.likeArt).toHaveBeenCalledWith("art-1");
        expect(mocks.unlikeArt).not.toHaveBeenCalled();
    });

    it("unlikes a piece the member has already liked", async () => {
        // given
        stubArt({ art: makeArt({ user_liked: true }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "♥ 4" }));

        // then
        expect(mocks.unlikeArt).toHaveBeenCalledWith("art-1");
        expect(mocks.likeArt).not.toHaveBeenCalled();
    });

    it("survives a like the server rejects", async () => {
        // given
        mocks.likeArt.mockRejectedValue(new Error("too many likes"));
        stubArt();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "♥ 4" }));

        // then
        await waitFor(() => {
            expect(screen.getByRole("button", { name: "♥ 4" })).toBeEnabled();
        });
    });
});

describe("ArtDetailPage ownership and moderation", () => {
    it("hides edit and delete from a member who did not upload the piece", () => {
        // given
        stubArt();

        // when
        renderPage(makeUser({ id: "someone-else" }));

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("offers edit and delete to the artist", () => {
        // given
        stubArt();

        // when
        renderPage(makeUser({ id: "author-1" }));

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("offers edit and delete to a moderator who is not the artist", () => {
        // given
        stubArt();

        // when
        renderPage(makeUser({ id: "mod-1", role: "moderator" }));

        // then
        expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("offers reporting to a member who is not the artist", () => {
        // given
        stubArt();

        // when
        renderPage(makeUser({ id: "someone-else" }));

        // then
        expect(screen.getByTestId("report")).toHaveAttribute("data-target", "art-1");
    });

    it("does not let the artist report their own piece", () => {
        // given
        stubArt();

        // when
        renderPage(makeUser({ id: "author-1" }));

        // then
        expect(screen.queryByTestId("report")).not.toBeInTheDocument();
    });

    it("does not offer reporting to a signed out visitor", () => {
        // given
        stubArt();

        // when
        renderPage(null);

        // then
        expect(screen.queryByTestId("report")).not.toBeInTheDocument();
    });
});

describe("ArtDetailPage editing", () => {
    async function openEditor() {
        const user = userEvent.setup();
        renderPage(makeUser({ id: "author-1" }));
        await user.click(screen.getByRole("button", { name: "Edit" }));

        return user;
    }

    it("fills the editor with the piece as it stands", async () => {
        // given
        stubArt({ art: makeArt({ tags: ["beatrice"], is_spoiler: true }) });

        // when
        await openEditor();

        // then
        expect(screen.getByPlaceholderText("Title")).toHaveValue("Beatrice at dusk");
        expect(screen.getByLabelText("Description")).toHaveValue("drawn in the rose garden");
        expect(screen.getByRole("switch", { name: "Contains spoilers" })).toHaveAttribute("aria-checked", "true");
    });

    it("saves the trimmed title and description with the tags and spoiler flag", async () => {
        // given
        stubArt({ art: makeArt({ tags: ["beatrice"] }) });
        const user = await openEditor();
        const titleBox = screen.getByPlaceholderText("Title");
        await user.clear(titleBox);
        await user.type(titleBox, "  Beatrice at dawn  ");
        await user.click(screen.getByRole("button", { name: "add the witches tag" }));
        await user.click(screen.getByRole("switch", { name: "Contains spoilers" }));

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        expect(mocks.updateArt).toHaveBeenCalledWith({
            title: "Beatrice at dawn",
            description: "drawn in the rose garden",
            tags: ["beatrice", "witches"],
            is_spoiler: true,
        });
    });

    it("refreshes the piece once the edit has been saved", async () => {
        // given
        const { refresh } = stubArt();
        const user = await openEditor();

        // when
        await user.click(screen.getByRole("button", { name: "Save" }));

        // then
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("refuses to save a piece whose title has been emptied", async () => {
        // given
        stubArt();
        const user = await openEditor();

        // when
        await user.clear(screen.getByPlaceholderText("Title"));

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
        await user.click(screen.getByRole("button", { name: "Save" }));
        expect(mocks.updateArt).not.toHaveBeenCalled();
        expect(screen.getByPlaceholderText("Title")).toBeInTheDocument();
    });

    it("offers save again once a title is typed back in", async () => {
        // given
        stubArt();
        const user = await openEditor();
        await user.clear(screen.getByPlaceholderText("Title"));

        // when
        await user.type(screen.getByPlaceholderText("Title"), "Beatrice at dawn");

        // then
        expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });

    it("throws the editor away when the edit is cancelled", async () => {
        // given
        stubArt();
        const user = await openEditor();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByPlaceholderText("Title")).not.toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Beatrice at dusk" })).toBeInTheDocument();
        expect(mocks.updateArt).not.toHaveBeenCalled();
    });

    it("hides the edit button while the editor is open", async () => {
        // given
        stubArt();

        // when
        await openEditor();

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    });
});

describe("ArtDetailPage deleting", () => {
    it("keeps the confirmation closed until delete is pressed", () => {
        // given
        stubArt();

        // when
        renderPage(makeUser({ id: "author-1" }));

        // then
        expect(screen.queryByRole("button", { name: "Delete Art" })).not.toBeInTheDocument();
    });

    it("warns that deleting cannot be undone", async () => {
        // given
        stubArt();
        const user = userEvent.setup();
        renderPage(makeUser({ id: "author-1" }));

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(
            screen.getByText("Are you sure you want to delete this art? This cannot be undone."),
        ).toBeInTheDocument();
    });

    it("deletes the piece and goes back once confirmed", async () => {
        // given
        stubArt();
        const user = userEvent.setup();
        renderPage(makeUser({ id: "author-1" }));
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // when
        await user.click(screen.getByRole("button", { name: "Delete Art" }));

        // then
        expect(mocks.deleteArt).toHaveBeenCalledWith("art-1");
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith(-1);
        });
    });

    it("keeps the piece when the confirmation is dismissed", async () => {
        // given
        stubArt();
        const user = userEvent.setup();
        renderPage(makeUser({ id: "author-1" }));
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByRole("button", { name: "Delete Art" })).not.toBeInTheDocument();
        expect(mocks.deleteArt).not.toHaveBeenCalled();
    });
});
