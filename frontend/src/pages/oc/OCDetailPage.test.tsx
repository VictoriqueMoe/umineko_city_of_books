import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { OCDetail, OCImage, UserProfile } from "../../types/api";
import { OCDetailPage } from "./OCDetailPage";

const mocks = vi.hoisted(() => ({
    useOC: vi.fn(),
    vote: vi.fn(),
    favourite: vi.fn(),
    navigate: vi.fn(),
    noop: vi.fn(),
}));

vi.mock("../../api/queries/oc", () => ({ useOC: mocks.useOC }));

vi.mock("../../api/mutations/oc", () => ({
    useVoteOC: () => ({ mutateAsync: mocks.vote }),
    useFavouriteOC: () => ({ mutateAsync: mocks.favourite }),
    useCreateOCComment: () => ({ mutateAsync: mocks.noop }),
    useUpdateOCComment: () => ({ mutateAsync: mocks.noop }),
    useDeleteOCComment: () => ({ mutateAsync: mocks.noop }),
    useLikeOCComment: () => ({ mutateAsync: mocks.noop }),
    useUnlikeOCComment: () => ({ mutateAsync: mocks.noop }),
    useUploadOCCommentMedia: () => ({ mutateAsync: mocks.noop }),
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

const author = { id: "author-1", username: "ronove", display_name: "Ronove" };

function makeImage(overrides: Partial<OCImage> = {}): OCImage {
    return {
        id: 1,
        image_url: "/gallery-1-full.png",
        thumbnail_url: "/gallery-1-thumb.png",
        sort_order: 0,
        ...overrides,
    };
}

function makeOC(overrides: Partial<OCDetail> = {}): OCDetail {
    return {
        id: "oc-1",
        author,
        name: "Featherine Junior",
        description: "a witch in training",
        series: "umineko",
        gallery: [],
        vote_score: 3,
        favourite_count: 2,
        user_favourited: false,
        comment_count: 0,
        is_crack_oc: false,
        created_at: "2026-07-01T10:00:00Z",
        comments: [],
        viewer_blocked: false,
        ...overrides,
    };
}

interface OCState {
    oc?: OCDetail | null;
    loading?: boolean;
}

function stubOC(state: OCState = {}) {
    const refresh = vi.fn(() => Promise.resolve());
    mocks.useOC.mockReturnValue({
        oc: state.oc === undefined ? makeOC() : state.oc,
        loading: state.loading ?? false,
        refresh,
    });
    return { refresh };
}

function renderPage(viewer: UserProfile | null = makeUser({ id: "member-1" })) {
    return renderWithProviders(<OCDetailPage />, { user: viewer, route: "/oc/oc-1", path: "/oc/:id" });
}

beforeEach(() => {
    mocks.vote.mockResolvedValue({});
    mocks.favourite.mockResolvedValue({});
});

describe("OCDetailPage loading and missing states", () => {
    it("says it is loading while the character is on its way", () => {
        // given
        stubOC({ oc: null, loading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading OC...")).toBeInTheDocument();
    });

    it("says the character could not be found once the fetch has settled", () => {
        // given
        stubOC({ oc: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("OC not found.")).toBeInTheDocument();
    });
});

describe("OCDetailPage content", () => {
    it("shows the name, the series and the description", () => {
        // given
        stubOC();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Featherine Junior" })).toBeInTheDocument();
        expect(screen.getByText("umineko")).toBeInTheDocument();
        expect(screen.getByText("a witch in training")).toBeInTheDocument();
    });

    it("labels a custom universe by the name its author gave it", () => {
        // given
        stubOC({ oc: makeOC({ series: "custom", custom_series_name: "Rose Guns Days" }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("Rose Guns Days")).toBeInTheDocument();
    });

    it("falls back to Custom when a custom universe has no name", () => {
        // given
        stubOC({ oc: makeOC({ series: "custom" }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("Custom")).toBeInTheDocument();
    });

    it("marks a positive score with a plus sign", () => {
        // given
        stubOC({ oc: makeOC({ vote_score: 3 }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("+3")).toBeInTheDocument();
    });

    it("hands the character's own comments to the comments section", () => {
        // given
        stubOC({ oc: makeOC({ id: "oc-7", viewer_blocked: true }) });

        // when
        renderPage();

        // then
        expect(screen.getByTestId("comments")).toHaveAttribute("data-target", "oc-7");
        expect(screen.getByTestId("comments")).toHaveAttribute("data-blocked", "true");
    });

    it("links back to the full character list", () => {
        // given
        stubOC();

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: "← Back to all OCs" })).toHaveAttribute("href", "/oc");
    });
});

describe("OCDetailPage gallery", () => {
    it("shows no gallery heading when the character has no extra images", () => {
        // given
        stubOC();

        // when
        renderPage();

        // then
        expect(screen.queryByRole("heading", { name: "Gallery" })).not.toBeInTheDocument();
    });

    it("shows every gallery image with its caption", () => {
        // given
        stubOC({
            oc: makeOC({
                gallery: [makeImage({ caption: "in the rose garden" }), makeImage({ id: 2, caption: "at the table" })],
            }),
        });

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Gallery" })).toBeInTheDocument();
        expect(screen.getByText("in the rose garden")).toBeInTheDocument();
        expect(screen.getByText("at the table")).toBeInTheDocument();
    });

    it("prefers the thumbnail for a gallery image", () => {
        // given
        stubOC({ oc: makeOC({ gallery: [makeImage({ caption: "in the rose garden" })] }) });

        // when
        renderPage();

        // then
        expect(screen.getByRole("img", { name: "in the rose garden" })).toHaveAttribute("src", "/gallery-1-thumb.png");
    });

    it("opens the full sized gallery image in the lightbox", async () => {
        // given
        stubOC({ oc: makeOC({ gallery: [makeImage({ caption: "in the rose garden" })] }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("img", { name: "in the rose garden" }));

        // then
        const lightbox = screen.getByRole("dialog");
        expect(lightbox).toBeInTheDocument();
        expect(screen.getAllByRole("img", { name: "in the rose garden" })[1]).toHaveAttribute(
            "src",
            "/gallery-1-full.png",
        );
    });

    it("opens the main portrait in the lightbox", async () => {
        // given
        stubOC({ oc: makeOC({ image_url: "/portrait.png" }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getAllByRole("img", { name: "Featherine Junior" })[0]);

        // then
        expect(screen.getByRole("dialog")).toBeInTheDocument();
    });

    it("closes the lightbox again", async () => {
        // given
        stubOC({ oc: makeOC({ image_url: "/portrait.png" }) });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getAllByRole("img", { name: "Featherine Junior" })[0]);

        // when
        await user.click(screen.getByRole("button", { name: "Close" }));

        // then
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
});

describe("OCDetailPage voting and favouriting", () => {
    it("locks voting and favouriting for a signed out visitor", () => {
        // given
        stubOC();

        // when
        renderPage(null);

        // then
        expect(screen.getByRole("button", { name: "△" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "▽" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "♡ 2" })).toBeDisabled();
    });

    it("casts an upvote for a member who has not voted yet", async () => {
        // given
        stubOC();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "△" }));

        // then
        expect(mocks.vote).toHaveBeenCalledWith(1);
    });

    it("casts a downvote for a member who has not voted yet", async () => {
        // given
        stubOC();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "▽" }));

        // then
        expect(mocks.vote).toHaveBeenCalledWith(-1);
    });

    it("clears an existing upvote when the same arrow is pressed again", async () => {
        // given
        stubOC({ oc: makeOC({ user_vote: 1 }) });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "▲" }));

        // then
        expect(mocks.vote).toHaveBeenCalledWith(0);
    });

    it("refreshes the character after a vote lands", async () => {
        // given
        const { refresh } = stubOC();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "△" }));

        // then
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("reports why a vote could not be cast", async () => {
        // given
        mocks.vote.mockRejectedValue(new Error("the witch forbids it"));
        stubOC();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "△" }));

        // then
        expect(await screen.findByText("the witch forbids it")).toBeInTheDocument();
    });

    it("falls back to a generic message when a vote failure carries no reason", async () => {
        // given
        mocks.vote.mockRejectedValue("boom");
        stubOC();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "△" }));

        // then
        expect(await screen.findByText("Failed to vote")).toBeInTheDocument();
    });

    it("favourites the character and refreshes it", async () => {
        // given
        const { refresh } = stubOC();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "♡ 2" }));

        // then
        expect(mocks.favourite).toHaveBeenCalledWith("oc-1");
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("shows a filled heart once the character is already favourited", () => {
        // given
        stubOC({ oc: makeOC({ user_favourited: true, favourite_count: 3 }) });

        // when
        renderPage();

        // then
        expect(screen.getByRole("button", { name: "♥ 3" })).toBeInTheDocument();
    });

    it("reports why the character could not be favourited", async () => {
        // given
        mocks.favourite.mockRejectedValue(new Error("already yours"));
        stubOC();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "♡ 2" }));

        // then
        expect(await screen.findByText("already yours")).toBeInTheDocument();
    });

    it("falls back to a generic message when a favourite failure carries no reason", async () => {
        // given
        mocks.favourite.mockRejectedValue("boom");
        stubOC();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "♡ 2" }));

        // then
        expect(await screen.findByText("Failed to favourite")).toBeInTheDocument();
    });
});

describe("OCDetailPage ownership", () => {
    it("hides the edit shortcut from a member who does not own the character", () => {
        // given
        stubOC();

        // when
        renderPage(makeUser({ id: "someone-else" }));

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    });

    it("hides the edit shortcut from a signed out visitor", () => {
        // given
        stubOC();

        // when
        renderPage(null);

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    });

    it("takes the owner to the editor", async () => {
        // given
        stubOC();
        const user = userEvent.setup();
        renderPage(makeUser({ id: "author-1" }));

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/oc/oc-1/edit");
    });
});
