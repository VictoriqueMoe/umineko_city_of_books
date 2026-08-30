import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type {
    PostComment,
    SecretComment,
    SecretDetailResponse,
    SecretLeaderboardEntry,
    UserProfile,
} from "../../types/api";
import { SecretDetailPage } from "./SecretDetailPage";

const {
    useSecretRoom,
    useCreateSecretComment,
    useUpdateSecretComment,
    useDeleteSecretComment,
    useLikeSecretComment,
    useUnlikeSecretComment,
    useUploadSecretCommentMedia,
} = vi.hoisted(() => ({
    useSecretRoom: vi.fn(),
    useCreateSecretComment: vi.fn(),
    useUpdateSecretComment: vi.fn(),
    useDeleteSecretComment: vi.fn(),
    useLikeSecretComment: vi.fn(),
    useUnlikeSecretComment: vi.fn(),
    useUploadSecretCommentMedia: vi.fn(),
}));

vi.mock("../../hooks/useSecretRoom", () => ({ useSecretRoom }));
vi.mock("../../hooks/mutations/secret", () => ({
    useCreateSecretComment,
    useUpdateSecretComment,
    useDeleteSecretComment,
    useLikeSecretComment,
    useUnlikeSecretComment,
    useUploadSecretCommentMedia,
}));
vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: (props: {
        comments: PostComment[];
        targetId: string;
        title?: string;
        emptyText?: string | null;
        composerPosition?: string;
        linkPrefix?: string;
        reportType?: string;
        onChanged: () => void;
        likeFn?: (id: string) => Promise<unknown>;
        unlikeFn?: (id: string) => Promise<unknown>;
        deleteFn?: (id: string) => Promise<unknown>;
        updateFn?: (id: string, body: string) => Promise<unknown>;
        createCommentFn?: (targetId: string, body: string, parentId?: string) => Promise<unknown>;
        uploadMediaFn?: (commentId: string, file: File) => Promise<unknown>;
    }) => (
        <section
            data-testid="comments-section"
            data-target={props.targetId}
            data-title={props.title ?? ""}
            data-empty={props.emptyText ?? ""}
            data-composer={props.composerPosition ?? ""}
            data-prefix={props.linkPrefix ?? ""}
            data-report={props.reportType ?? ""}
        >
            <span>{`${props.comments.length} comments`}</span>
            <button onClick={props.onChanged}>stub changed</button>
            <button onClick={() => props.likeFn?.("comment-9")}>stub like</button>
            <button onClick={() => props.unlikeFn?.("comment-9")}>stub unlike</button>
            <button onClick={() => props.deleteFn?.("comment-9")}>stub delete</button>
            <button onClick={() => props.updateFn?.("comment-9", "edited body")}>stub update</button>
            <button onClick={() => props.createCommentFn?.("ignored", "new body", "parent-1")}>stub create</button>
            <button onClick={() => props.uploadMediaFn?.("comment-9", new File(["x"], "clue.png"))}>stub upload</button>
        </section>
    ),
}));

const beatrice = { id: "user-1", username: "beatrice", display_name: "Beatrice" };
const ange = { id: "user-2", username: "ange", display_name: "Ange" };
const battler = { id: "user-3", username: "battler", display_name: "Battler" };
const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });

function makeEntry(overrides: Partial<SecretLeaderboardEntry> = {}): SecretLeaderboardEntry {
    return {
        user: beatrice,
        pieces_collected: 1,
        solved: false,
        ...overrides,
    };
}

function makeSecretComment(overrides: Partial<SecretComment> = {}): SecretComment {
    return {
        id: "comment-1",
        author: beatrice,
        body: "A quiet clue.",
        media: [],
        like_count: 0,
        user_liked: false,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

function makeDetail(overrides: Partial<SecretDetailResponse> = {}): SecretDetailResponse {
    return {
        id: "secret-1",
        title: "The First Twilight",
        description: "Six chosen by the key.",
        total_pieces: 5,
        solved: false,
        viewer_progress: 0,
        comment_count: 0,
        riddle: "Who is the golden witch?",
        leaderboard: [],
        comments: [],
        ...overrides,
    };
}

interface StubOptions {
    detail?: SecretDetailResponse | null;
    loading?: boolean;
    solvedByName?: string | null;
}

function stubSecret(options: StubOptions = {}) {
    const refresh = vi.fn();
    const dismissSolved = vi.fn();
    useSecretRoom.mockReturnValue({
        detail: options.detail === undefined ? makeDetail() : options.detail,
        loading: options.loading ?? false,
        refresh,
        solvedByName: options.solvedByName ?? null,
        dismissSolved,
    });

    const createAsync = vi.fn(() => Promise.resolve({ id: "comment-new" }));
    const updateAsync = vi.fn(() => Promise.resolve({ id: "comment-9" }));
    const deleteAsync = vi.fn(() => Promise.resolve(undefined));
    const likeAsync = vi.fn(() => Promise.resolve(undefined));
    const unlikeAsync = vi.fn(() => Promise.resolve(undefined));
    const uploadAsync = vi.fn(() => Promise.resolve(undefined));

    useCreateSecretComment.mockReturnValue({ mutateAsync: createAsync });
    useUpdateSecretComment.mockReturnValue({ mutateAsync: updateAsync });
    useDeleteSecretComment.mockReturnValue({ mutateAsync: deleteAsync });
    useLikeSecretComment.mockReturnValue({ mutateAsync: likeAsync });
    useUnlikeSecretComment.mockReturnValue({ mutateAsync: unlikeAsync });
    useUploadSecretCommentMedia.mockReturnValue({ mutateAsync: uploadAsync });

    return { refresh, dismissSolved, createAsync, updateAsync, deleteAsync, likeAsync, unlikeAsync, uploadAsync };
}

interface PageOptions {
    user?: UserProfile | null;
    route?: string;
}

function renderPage(options: PageOptions = {}) {
    return renderWithProviders(<SecretDetailPage />, {
        user: options.user ?? null,
        route: options.route ?? "/secrets/secret-1",
        path: "/secrets/:id",
    });
}

describe("SecretDetailPage", () => {
    afterEach(() => {
        vi.useRealTimers();
    });

    it("consults the game board while the hunt is loading", () => {
        // given
        stubSecret({ loading: true, detail: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Consulting the game board...")).toBeInTheDocument();
    });

    it("offers a way back when the hunt does not exist", () => {
        // given
        stubSecret({ detail: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Secret not found.")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Back to secrets" })).toHaveAttribute("href", "/secrets");
    });

    it("shows the title, the description and the riddle", () => {
        // given
        stubSecret();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "The First Twilight" })).toBeInTheDocument();
        expect(screen.getByText("Six chosen by the key.")).toBeInTheDocument();
        expect(screen.getByText("Who is the golden witch?")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Secrets" })).toHaveAttribute("href", "/secrets");
    });

    it("says the hunt is still open when nobody has answered", () => {
        // given
        stubSecret({ detail: makeDetail({ solved: false }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("Open. No one has spoken the answer yet.")).toBeInTheDocument();
    });

    it("names the hunter who spoke the answer", () => {
        // given
        stubSecret({ detail: makeDetail({ solved: true, solver: beatrice, solved_at: "2026-07-02T10:00:00Z" }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("Solved")).toBeInTheDocument();
    });

    it("keeps the progress chip away from a signed out visitor", () => {
        // given
        stubSecret({ detail: makeDetail({ viewer_progress: 3 }) });

        // when
        renderPage({ user: null });

        // then
        expect(screen.queryByText("You: 3 / 5")).not.toBeInTheDocument();
    });

    it("tells a signed in hunter how many pieces they hold", () => {
        // given
        stubSecret({ detail: makeDetail({ viewer_progress: 3 }) });

        // when
        renderPage({ user: reader });

        // then
        expect(screen.getByText(/You:/)).toHaveTextContent("You: 3 / 5");
    });

    it("invites the first piece when nobody has started", () => {
        // given
        stubSecret({ detail: makeDetail({ leaderboard: [] }) });

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Progress (0 hunters)" })).toBeInTheDocument();
        expect(screen.getByText("No one has picked up a piece yet. Be the first.")).toBeInTheDocument();
    });

    it("counts a lone hunter in the singular", () => {
        // given
        stubSecret({ detail: makeDetail({ leaderboard: [makeEntry()] }) });

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Progress (1 hunter)" })).toBeInTheDocument();
    });

    it("ranks the hunters down the page in the order the hook handed them over", () => {
        // given
        stubSecret({
            detail: makeDetail({
                leaderboard: [
                    makeEntry({ user: beatrice, pieces_collected: 1, solved: true }),
                    makeEntry({ user: ange, pieces_collected: 2, solved: false }),
                    makeEntry({ user: battler, pieces_collected: 2, solved: false }),
                ],
            }),
        });

        // when
        const { container } = renderPage();

        // then
        const names = Array.from(container.querySelectorAll("span")).filter(el => el.textContent?.startsWith("#"));
        expect(names.map(el => el.textContent)).toEqual(["#1", "#2", "#3"]);
        expect(screen.getByText("#1").parentElement).toHaveTextContent("Beatrice");
        expect(screen.getByText("#2").parentElement).toHaveTextContent("Ange");
        expect(screen.getByText("#3").parentElement).toHaveTextContent("Battler");
    });

    it("shows each hunter's pieces out of the total", () => {
        // given
        stubSecret({ detail: makeDetail({ total_pieces: 7, leaderboard: [makeEntry({ pieces_collected: 4 })] }) });

        // when
        renderPage();

        // then
        expect(screen.getByText("4 / 7")).toBeInTheDocument();
    });

    it("announces the moment the hook says someone spoke the witch's name", () => {
        // given
        stubSecret({ solvedByName: "Beatrice" });

        // when
        renderPage();

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Beatrice spoke the witch's name.");
    });

    it("stays quiet while nobody has spoken the name", () => {
        // given
        stubSecret({ solvedByName: null });

        // when
        renderPage();

        // then
        expect(screen.queryByRole("status")).not.toBeInTheDocument();
    });

    it("tells the hook to forget the announcement once the toast has had its say", () => {
        // given
        vi.useFakeTimers();
        const { dismissSolved } = stubSecret({ solvedByName: "Beatrice" });
        renderPage();

        // when
        act(() => {
            vi.advanceTimersByTime(6000);
        });

        // then
        expect(dismissSolved).toHaveBeenCalledOnce();
    });

    it("labels the discussion with its own wording", () => {
        // given
        stubSecret({ detail: makeDetail({ comments: [makeSecretComment(), makeSecretComment({ id: "comment-2" })] }) });

        // when
        renderPage({ user: reader });

        // then
        const section = screen.getByTestId("comments-section");
        expect(section).toHaveAttribute("data-title", "Discussion");
        expect(section).toHaveAttribute("data-empty", "No one has left a word yet.");
        expect(section).toHaveAttribute("data-composer", "top");
        expect(section).toHaveAttribute("data-prefix", "/secrets/secret-1");
        expect(section).toHaveAttribute("data-report", "secret_comment");
        expect(screen.getByText("2 comments")).toBeInTheDocument();
    });

    it("copes with a hunt whose comments the server left out", () => {
        // given
        stubSecret({ detail: makeDetail({ comments: undefined as unknown as SecretComment[] }) });

        // when
        renderPage({ user: reader });

        // then
        expect(screen.getByText("0 comments")).toBeInTheDocument();
    });

    it("posts a new word through the hunt's own comment mutation", async () => {
        // given
        const { createAsync } = stubSecret();
        const user = userEvent.setup();
        renderPage({ user: reader });

        // when
        await user.click(screen.getByRole("button", { name: "stub create" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({ body: "new body", parentId: "parent-1" });
    });

    it("edits a word through the hunt's own comment mutation", async () => {
        // given
        const { updateAsync } = stubSecret();
        const user = userEvent.setup();
        renderPage({ user: reader });

        // when
        await user.click(screen.getByRole("button", { name: "stub update" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith({ id: "comment-9", commentId: "comment-9", body: "edited body" });
    });

    it("likes, unlikes and removes a word through the hunt's own mutations", async () => {
        // given
        const { likeAsync, unlikeAsync, deleteAsync } = stubSecret();
        const user = userEvent.setup();
        renderPage({ user: reader });

        // when
        await user.click(screen.getByRole("button", { name: "stub like" }));
        await user.click(screen.getByRole("button", { name: "stub unlike" }));
        await user.click(screen.getByRole("button", { name: "stub delete" }));

        // then
        expect(likeAsync).toHaveBeenCalledWith("comment-9");
        expect(unlikeAsync).toHaveBeenCalledWith("comment-9");
        expect(deleteAsync).toHaveBeenCalledWith("comment-9");
    });

    it("uploads an attachment against the comment it belongs to", async () => {
        // given
        const { uploadAsync } = stubSecret();
        const user = userEvent.setup();
        renderPage({ user: reader });

        // when
        await user.click(screen.getByRole("button", { name: "stub upload" }));

        // then
        expect(uploadAsync).toHaveBeenCalledWith({ commentId: "comment-9", file: expect.any(File) });
    });

    it("refetches the hunt after the discussion changes", async () => {
        // given
        const { refresh } = stubSecret();
        const user = userEvent.setup();
        renderPage({ user: reader });

        // when
        await user.click(screen.getByRole("button", { name: "stub changed" }));

        // then
        expect(refresh).toHaveBeenCalledOnce();
    });
});
