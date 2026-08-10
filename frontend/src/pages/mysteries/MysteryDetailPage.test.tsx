import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, type Mock } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { MysteryAttempt, MysteryDetail, UserProfile } from "../../types/api";
import { MysteryDetailPage } from "./MysteryDetailPage";

const mocked = vi.hoisted(() => ({
    useMystery: vi.fn(),
    useAddMysteryClue: vi.fn(),
    useCloseMystery: vi.fn(),
    useCreateMysteryAttempt: vi.fn(),
    useCreateMysteryComment: vi.fn(),
    useDeleteMystery: vi.fn(),
    useDeleteMysteryAttachment: vi.fn(),
    useDeleteMysteryClue: vi.fn(),
    useDeleteMysteryComment: vi.fn(),
    useDeleteMysteryMedia: vi.fn(),
    useLikeMysteryComment: vi.fn(),
    useSetMysteryGmAway: vi.fn(),
    useSetMysteryPaused: vi.fn(),
    useUnlikeMysteryComment: vi.fn(),
    useUpdateMysteryClue: vi.fn(),
    useUpdateMysteryComment: vi.fn(),
    useUploadMysteryAttachment: vi.fn(),
    useUploadMysteryCommentMedia: vi.fn(),
    useUploadMysteryMedia: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/mystery", () => ({ useMystery: mocked.useMystery }));
vi.mock("../../api/mutations/mystery", () => ({
    useAddMysteryClue: mocked.useAddMysteryClue,
    useCloseMystery: mocked.useCloseMystery,
    useCreateMysteryAttempt: mocked.useCreateMysteryAttempt,
    useCreateMysteryComment: mocked.useCreateMysteryComment,
    useDeleteMystery: mocked.useDeleteMystery,
    useDeleteMysteryAttachment: mocked.useDeleteMysteryAttachment,
    useDeleteMysteryClue: mocked.useDeleteMysteryClue,
    useDeleteMysteryComment: mocked.useDeleteMysteryComment,
    useDeleteMysteryMedia: mocked.useDeleteMysteryMedia,
    useLikeMysteryComment: mocked.useLikeMysteryComment,
    useSetMysteryGmAway: mocked.useSetMysteryGmAway,
    useSetMysteryPaused: mocked.useSetMysteryPaused,
    useUnlikeMysteryComment: mocked.useUnlikeMysteryComment,
    useUpdateMysteryClue: mocked.useUpdateMysteryClue,
    useUpdateMysteryComment: mocked.useUpdateMysteryComment,
    useUploadMysteryAttachment: mocked.useUploadMysteryAttachment,
    useUploadMysteryCommentMedia: mocked.useUploadMysteryCommentMedia,
    useUploadMysteryMedia: mocked.useUploadMysteryMedia,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocked.navigate };
});

interface AttemptStubProps {
    attempt: MysteryAttempt;
    authorAlreadyWon: boolean;
    mysteryPaused: boolean;
    mysterySolved: boolean;
}

vi.mock("./AttemptItem", () => ({
    AttemptItem: (props: AttemptStubProps) => (
        <article aria-label={`attempt by ${props.attempt.author.display_name}`}>
            <p>{props.attempt.body}</p>
            <p>{props.authorAlreadyWon ? "author already won" : "author has not won"}</p>
        </article>
    ),
}));

vi.mock("../../components/post/CommentsSection/CommentsSection", () => ({
    CommentsSection: ({ title, targetId }: { title: string; targetId: string }) => (
        <section aria-label="comments">{`${title} for ${targetId}`}</section>
    ),
}));

vi.mock("../../components/post/MediaGallery/MediaGallery", () => ({
    MediaGallery: ({ media }: { media: { id: number }[] }) => (
        <div data-testid="media-gallery">{`${media.length} media`}</div>
    ),
}));

const gameMaster = { id: "gm-1", username: "beatrice", display_name: "Beatrice" };
const player = { id: "player-1", username: "battler", display_name: "Battler" };
const otherPlayer = { id: "player-2", username: "ange", display_name: "Ange" };

const gameMasterUser = makeUser({ id: "gm-1", username: "beatrice", display_name: "Beatrice" });
const playerUser = makeUser({ id: "player-1", username: "battler", display_name: "Battler" });
const moderatorUser = makeUser({ id: "mod-1", username: "virgilia", display_name: "Virgilia", role: "moderator" });

function makeAttempt(overrides: Partial<MysteryAttempt> = {}): MysteryAttempt {
    return {
        id: "attempt-1",
        author: player,
        body: "The chain was fixed after the fact.",
        is_winner: false,
        vote_score: 0,
        created_at: "2026-07-01T11:00:00Z",
        ...overrides,
    };
}

function makeMysteryDetail(overrides: Partial<MysteryDetail> = {}): MysteryDetail {
    return {
        id: "mystery-1",
        title: "The sealed guest room",
        body: "Six people died behind a chained door.",
        difficulty: "hard",
        author: gameMaster,
        solved: false,
        paused: false,
        gm_away: false,
        free_for_all: false,
        keep_open_after_solve: false,
        knox_contract: {
            culprit_named_early: true,
            no_supernatural: true,
            passages_declared: true,
            no_unknown_poison: true,
            no_outsider: true,
            no_lucky_accident: true,
            detective_not_culprit: true,
            clues_shown: true,
            narrator_hides_nothing: true,
            no_unannounced_twins: true,
        },
        knox_contract_published: true,
        knox_contract_locked: false,
        solver_count: 0,
        viewer_has_solved: false,
        paused_duration_seconds: 0,
        clues: [],
        attempts: [],
        comments: [],
        player_count: 0,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

function mutation(hook: Mock, impl?: () => Promise<unknown>): Mock {
    const mutateAsync = vi.fn(impl ?? (() => Promise.resolve({})));
    hook.mockReturnValue({ mutateAsync });

    return mutateAsync;
}

interface StubOptions {
    mystery?: MysteryDetail | null;
    loading?: boolean;
}

function stubMystery(options: StubOptions = {}) {
    const refresh = vi.fn();
    mocked.useMystery.mockReturnValue({
        mystery: options.mystery === undefined ? makeMysteryDetail() : options.mystery,
        loading: options.loading ?? false,
        refresh,
    });

    const addClue = mutation(mocked.useAddMysteryClue);
    const closeMystery = mutation(mocked.useCloseMystery);
    const createAttempt = mutation(mocked.useCreateMysteryAttempt);
    const deleteMystery = mutation(mocked.useDeleteMystery);
    const deleteAttachment = mutation(mocked.useDeleteMysteryAttachment);
    const deleteClue = mutation(mocked.useDeleteMysteryClue);
    const deleteMedia = mutation(mocked.useDeleteMysteryMedia);
    const setGmAway = mutation(mocked.useSetMysteryGmAway);
    const setPaused = mutation(mocked.useSetMysteryPaused);
    const updateClue = mutation(mocked.useUpdateMysteryClue);
    const uploadAttachment = mutation(mocked.useUploadMysteryAttachment);
    const uploadMedia = mutation(mocked.useUploadMysteryMedia);
    mutation(mocked.useCreateMysteryComment);
    mutation(mocked.useDeleteMysteryComment);
    mutation(mocked.useLikeMysteryComment);
    mutation(mocked.useUnlikeMysteryComment);
    mutation(mocked.useUpdateMysteryComment);
    mutation(mocked.useUploadMysteryCommentMedia);

    return {
        refresh,
        addClue,
        closeMystery,
        createAttempt,
        deleteMystery,
        deleteAttachment,
        deleteClue,
        deleteMedia,
        setGmAway,
        setPaused,
        updateClue,
        uploadAttachment,
        uploadMedia,
    };
}

function renderPage(user: UserProfile | null = null) {
    return renderWithProviders(<MysteryDetailPage />, {
        user,
        route: "/mystery/mystery-1",
        path: "/mystery/:id",
    });
}

describe("MysteryDetailPage", () => {
    it("investigates while the mystery is loading", () => {
        // given
        stubMystery({ loading: true, mystery: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Investigating the mystery...")).toBeInTheDocument();
    });

    it("says the mystery is missing when the server has none", () => {
        // given
        stubMystery({ mystery: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("Mystery not found.")).toBeInTheDocument();
    });

    it("presents the scenario with its difficulty and player count", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ player_count: 2 }) });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("heading", { name: "The sealed guest room" })).toBeInTheDocument();
        expect(screen.getByText("Six people died behind a chained door.")).toBeInTheDocument();
        expect(screen.getByText("hard")).toBeInTheDocument();
        expect(screen.getByText("Open")).toBeInTheDocument();
        expect(screen.getByText("2 pieces attempting")).toBeInTheDocument();
    });

    it("uses the singular when only one piece is attempting", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ player_count: 1 }) });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText("1 piece attempting")).toBeInTheDocument();
    });

    it("celebrates the winner once the mystery is solved", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ solved: true, winner: player }) });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText("Mystery solved! Winner: Battler")).toBeInTheDocument();
        expect(screen.getByText("Solved")).toBeInTheDocument();
    });

    it("badges a paused mystery and drops the away badge behind it", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ paused: true, gm_away: true }) });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText("Paused")).toBeInTheDocument();
        expect(screen.queryByText("GM Away")).not.toBeInTheDocument();
    });

    it("badges an ongoing mystery and counts its solvers", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({ keep_open_after_solve: true, free_for_all: true, solver_count: 3 }),
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText("Ongoing")).toBeInTheDocument();
        expect(screen.getByText("Free-for-all")).toBeInTheDocument();
        expect(screen.getByText("3 solvers")).toBeInTheDocument();
    });

    it("gives a signed out visitor nothing but a prompt to sign in", async () => {
        // given
        stubMystery();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Sign in to attempt" }));

        // then
        expect(mocked.navigate).toHaveBeenCalledWith("/login");
        expect(screen.queryByPlaceholderText("Declare your blue truth...")).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("keeps the board controls away from an ordinary piece", () => {
        // given
        stubMystery();

        // when
        renderPage(playerUser);

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Pause" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Report" })).toBeInTheDocument();
    });

    it("gives the game master the board controls and the edit button", async () => {
        // given
        stubMystery();
        const user = userEvent.setup();

        // when
        renderPage(gameMasterUser);

        // then
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Pause" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Mark as away" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: "Edit" }));
        expect(mocked.navigate).toHaveBeenCalledWith("/mystery/mystery-1/edit");
    });

    it("lets a moderator edit and delete a mystery they did not set", async () => {
        // given
        stubMystery();
        const user = userEvent.setup();
        renderPage(moderatorUser);

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // then
        expect(mocked.navigate).toHaveBeenCalledWith("/mystery/mystery-1/edit");
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("asks before deleting the mystery and then returns to the list", async () => {
        // given
        const { deleteMystery } = stubMystery();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this mystery? This cannot be undone.");
        expect(deleteMystery).toHaveBeenCalledWith("mystery-1");
        await waitFor(() => {
            expect(mocked.navigate).toHaveBeenCalledWith("/mysteries");
        });
    });

    it("keeps the mystery when the deletion is waved away", async () => {
        // given
        const { deleteMystery } = stubMystery();
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(deleteMystery).not.toHaveBeenCalled();
    });

    it("pauses the mystery and refreshes the board", async () => {
        // given
        const { setPaused, refresh } = stubMystery();
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Pause" }));

        // then
        expect(setPaused).toHaveBeenCalledWith(true);
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("offers to resume a paused mystery and hides the away toggle meanwhile", async () => {
        // given
        const { setPaused } = stubMystery({ mystery: makeMysteryDetail({ paused: true }) });
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Resume" }));

        // then
        expect(setPaused).toHaveBeenCalledWith(false);
        expect(screen.queryByRole("button", { name: "Mark as away" })).not.toBeInTheDocument();
    });

    it("lets the game master step away and come back", async () => {
        // given
        const { setGmAway } = stubMystery({ mystery: makeMysteryDetail({ gm_away: true }) });
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "I'm back" }));

        // then
        expect(setGmAway).toHaveBeenCalledWith(false);
    });

    it("offers to close an ongoing mystery permanently and asks first", async () => {
        // given
        const { closeMystery } = stubMystery({ mystery: makeMysteryDetail({ keep_open_after_solve: true }) });
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Mark Permanently Solved" }));

        // then
        expect(confirm).toHaveBeenCalled();
        expect(closeMystery).not.toHaveBeenCalled();
    });

    it("closes an ongoing mystery once the game master confirms", async () => {
        // given
        const { closeMystery, refresh } = stubMystery({ mystery: makeMysteryDetail({ keep_open_after_solve: true }) });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Mark Permanently Solved" }));

        // then
        expect(closeMystery).toHaveBeenCalled();
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("does not offer to close a mystery that is not ongoing", () => {
        // given
        stubMystery();

        // when
        renderPage(gameMasterUser);

        // then
        expect(screen.queryByRole("button", { name: "Mark Permanently Solved" })).not.toBeInTheDocument();
    });

    it("lists the global red truths and leaves the private ones out", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                clues: [
                    { id: 1, body: "The door was chained", truth_type: "red", sort_order: 0 },
                    {
                        id: 2,
                        body: "Only Ange may know this",
                        truth_type: "red",
                        sort_order: 1,
                        player_id: otherPlayer.id,
                    },
                ],
            }),
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("heading", { name: "Red Truths" })).toBeInTheDocument();
        expect(screen.getByText("The door was chained")).toBeInTheDocument();
        expect(screen.queryByText("Only Ange may know this")).not.toBeInTheDocument();
    });

    it("copies a red truth to the clipboard", async () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                clues: [{ id: 1, body: "The door was chained", truth_type: "red", sort_order: 0 }],
            }),
        });
        const writeText = vi.fn(() => Promise.resolve());
        vi.spyOn(navigator.clipboard, "writeText").mockImplementation(writeText);
        const user = userEvent.setup();
        renderPage(playerUser);

        // when
        await user.click(screen.getByRole("button", { name: "Copy to clipboard" }));

        // then
        expect(writeText).toHaveBeenCalledWith("The door was chained");
    });

    it("lets the game master declare a new global red truth", async () => {
        // given
        const { addClue, refresh } = stubMystery();
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // then
        expect(screen.getByRole("button", { name: "Add global Red Truth" })).toBeDisabled();

        // when
        await user.type(screen.getByPlaceholderText("Add a new red truth clue..."), "  The window was latched  ");
        await user.click(screen.getByRole("button", { name: "Add global Red Truth" }));

        // then
        expect(addClue).toHaveBeenCalledWith({ body: "The window was latched", truthType: "red" });
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
        expect(screen.getByPlaceholderText("Add a new red truth clue...")).toHaveValue("");
    });

    it("keeps the red truth composer away from the pieces", () => {
        // given
        stubMystery();

        // when
        renderPage(playerUser);

        // then
        expect(screen.queryByPlaceholderText("Add a new red truth clue...")).not.toBeInTheDocument();
    });

    it("lets the game master amend a private red truth of their own mystery", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                attempts: [makeAttempt({ id: "a1", body: "First guess" })],
                clues: [
                    {
                        id: 5,
                        body: "Only Battler may know this",
                        truth_type: "red",
                        sort_order: 0,
                        player_id: player.id,
                    },
                ],
            }),
        });

        // when
        renderPage(gameMasterUser);

        // then
        expect(screen.getByText("Only Battler may know this")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "delete" })).toBeInTheDocument();
    });

    it("groups the attempts by player for the game master and offers jump pills", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                attempts: [
                    makeAttempt({ id: "a1", body: "First guess" }),
                    makeAttempt({ id: "a2", body: "Second guess" }),
                    makeAttempt({ id: "a3", author: otherPlayer, body: "Ange's guess" }),
                ],
            }),
        });

        // when
        renderPage(gameMasterUser);

        // then
        expect(screen.getByText("Blue Truth Attempts (3)")).toBeInTheDocument();
        expect(screen.getByTitle("Jump to Battler's attempts")).toBeInTheDocument();
        expect(screen.getByTitle("Jump to Ange's attempts")).toBeInTheDocument();
        expect(screen.getByText("2 attempts")).toBeInTheDocument();
        expect(screen.getByText("1 attempt")).toBeInTheDocument();
    });

    it("folds a player's thread away when the group header is clicked", async () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ attempts: [makeAttempt({ body: "First guess" })] }) });
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: /1 attempt/ }));

        // then
        expect(screen.queryByText("First guess")).not.toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: /1 attempt/ }));
        expect(screen.getByText("First guess")).toBeInTheDocument();
    });

    it("shows a piece only a flat thread with no pills in a private mystery", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                attempts: [makeAttempt({ id: "a1", body: "First guess" })],
            }),
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText("First guess")).toBeInTheDocument();
        expect(screen.queryByTitle("Jump to Battler's attempts")).not.toBeInTheDocument();
        expect(screen.queryByText("1 attempt")).not.toBeInTheDocument();
    });

    it("shows every piece the grouped attempts in a free-for-all", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                free_for_all: true,
                attempts: [makeAttempt({ id: "a1" }), makeAttempt({ id: "a2", author: otherPlayer })],
            }),
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByTitle("Jump to Ange's attempts")).toBeInTheDocument();
        expect(screen.getAllByRole("article")).toHaveLength(2);
    });

    it("reveals every attempt once the mystery is solved and pins the winning one", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                solved: true,
                winner: player,
                attempts: [
                    makeAttempt({ id: "a1", body: "The chain trick", is_winner: true }),
                    makeAttempt({ id: "a2", author: otherPlayer, body: "A wrong guess" }),
                ],
            }),
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText("Winning Attempt")).toBeInTheDocument();
        expect(screen.getAllByText("The chain trick").length).toBeGreaterThan(1);
        expect(screen.getByText("A wrong guess")).toBeInTheDocument();
        expect(screen.queryByTitle("Jump to Battler's attempts")).not.toBeInTheDocument();
    });

    it("tells the attempt list which authors have already won", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                attempts: [
                    makeAttempt({ id: "a1", body: "The chain trick", is_winner: true }),
                    makeAttempt({ id: "a2", author: otherPlayer, body: "A wrong guess" }),
                ],
            }),
        });

        // when
        renderPage(gameMasterUser);

        // then
        const battler = screen.getByRole("article", { name: "attempt by Battler" });
        const ange = screen.getByRole("article", { name: "attempt by Ange" });
        expect(within(battler).getByText("author already won")).toBeInTheDocument();
        expect(within(ange).getByText("author has not won")).toBeInTheDocument();
    });

    it("tells the game master that nobody has moved yet", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ attempts: [] }) });

        // when
        renderPage(gameMasterUser);

        // then
        expect(screen.getByText("No attempts yet. Waiting for pieces to make their move.")).toBeInTheDocument();
    });

    it("invites the first blue truth when nobody is playing yet", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ attempts: [], player_count: 0 }) });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText("No attempts yet. Be the first to declare your blue truth!")).toBeInTheDocument();
    });

    it("tells a piece how many others are already playing a private mystery", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ attempts: [], player_count: 3 }) });

        // when
        renderPage(playerUser);

        // then
        expect(
            screen.getByText(
                "There are 3 pieces playing this mystery. Join the game board and declare your own blue truth!",
            ),
        ).toBeInTheDocument();
    });

    it("submits a trimmed blue truth and clears the composer", async () => {
        // given
        const { createAttempt, refresh } = stubMystery();
        const user = userEvent.setup();
        renderPage(playerUser);

        // then
        expect(screen.getByRole("button", { name: "Submit Blue Truth" })).toBeDisabled();

        // when
        await user.type(screen.getByPlaceholderText("Declare your blue truth..."), "  The chain was faked  ");
        await user.click(screen.getByRole("button", { name: "Submit Blue Truth" }));

        // then
        expect(createAttempt).toHaveBeenCalledWith({ body: "The chain was faked" });
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
        expect(screen.getByPlaceholderText("Declare your blue truth...")).toHaveValue("");
    });

    it("closes the composer and explains the pause to the pieces", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ paused: true }) });

        // when
        renderPage(playerUser);

        // then
        expect(
            screen.getByText("The Game Master has paused this mystery. New attempts are temporarily disabled."),
        ).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Declare your blue truth...")).not.toBeInTheDocument();
    });

    it("warns that the game master is away but still takes attempts", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ gm_away: true }) });

        // when
        renderPage(playerUser);

        // then
        expect(
            screen.getByText(
                "The Game Master is currently away. You can still post theories, but responses may be delayed.",
            ),
        ).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Declare your blue truth...")).toBeInTheDocument();
    });

    it("closes the composer once the mystery is solved", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ solved: true, winner: player }) });

        // when
        renderPage(playerUser);

        // then
        expect(screen.queryByPlaceholderText("Declare your blue truth...")).not.toBeInTheDocument();
    });

    it("never offers the game master a composer of their own", () => {
        // given
        stubMystery();

        // when
        renderPage(gameMasterUser);

        // then
        expect(screen.queryByPlaceholderText("Declare your blue truth...")).not.toBeInTheDocument();
    });

    it("opens the post game discussion only once the mystery is solved", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ solved: true, winner: player }) });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("region", { name: "comments" })).toHaveTextContent(
            "Post-Game Discussion for mystery-1",
        );
    });

    it("keeps the post game discussion shut while the mystery is open", () => {
        // given
        stubMystery();

        // when
        renderPage(playerUser);

        // then
        expect(screen.queryByRole("region", { name: "comments" })).not.toBeInTheDocument();
    });

    it("shows a piece the private red truths written for them", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                clues: [
                    { id: 5, body: "Your key never left you", truth_type: "red", sort_order: 0, player_id: player.id },
                ],
            }),
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("button", { name: /Private Red Truths \(to you\) \(1\)/ })).toBeInTheDocument();
        expect(screen.getByText("Your key never left you")).toBeInTheDocument();
    });

    it("lets a piece fold their private red truths away", async () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                clues: [
                    { id: 5, body: "Your key never left you", truth_type: "red", sort_order: 0, player_id: player.id },
                ],
            }),
        });
        const user = userEvent.setup();
        renderPage(playerUser);

        // when
        await user.click(screen.getByRole("button", { name: /Private Red Truths \(to you\)/ }));

        // then
        expect(screen.queryByText("Your key never left you")).not.toBeInTheDocument();
        expect(localStorage.getItem("mystery:mystery-1:private-clues:player-1:collapsed")).toBe("1");
    });

    it("lets the game master whisper a private red truth to a player", async () => {
        // given
        const { addClue, refresh } = stubMystery({
            mystery: makeMysteryDetail({ attempts: [makeAttempt()] }),
        });
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.type(screen.getByPlaceholderText("Private red truth for this player..."), "  Your key is a lie  ");
        await user.click(screen.getByRole("button", { name: "Add private Red Truth" }));

        // then
        expect(addClue).toHaveBeenCalledWith({ body: "Your key is a lie", truthType: "red", playerId: "player-1" });
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("keeps the private red truth composer away from the pieces", () => {
        // given
        stubMystery({ mystery: makeMysteryDetail({ free_for_all: true, attempts: [makeAttempt()] }) });

        // when
        renderPage(playerUser);

        // then
        expect(screen.queryByPlaceholderText("Private red truth for this player...")).not.toBeInTheDocument();
    });

    it("lists the attachments with a readable size", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                attachments: [{ id: 3, file_url: "/files/notes.pdf", file_name: "notes.pdf", file_size: 2048 }],
            }),
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("link", { name: "notes.pdf" })).toHaveAttribute("href", "/files/notes.pdf");
        expect(screen.getByText("2.0 KB")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Add Attachment" })).not.toBeInTheDocument();
    });

    it("hides the attachments panel from a piece when there is nothing attached", () => {
        // given
        stubMystery();

        // when
        renderPage(playerUser);

        // then
        expect(screen.queryByRole("heading", { name: "Attachments" })).not.toBeInTheDocument();
    });

    it("lets the game master attach a file to the mystery", async () => {
        // given
        const { uploadAttachment, refresh } = stubMystery();
        const user = userEvent.setup();
        const { container } = renderPage(gameMasterUser);

        // when
        const input = container.querySelector<HTMLInputElement>("input[accept='.pdf,.txt,.docx']")!;
        await user.upload(input, new File(["evidence"], "notes.pdf", { type: "application/pdf" }));

        // then
        expect(uploadAttachment).toHaveBeenCalledWith(expect.any(File));
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });

    it("reports why an attachment could not be uploaded", async () => {
        // given
        stubMystery();
        mocked.useUploadMysteryAttachment.mockReturnValue({
            mutateAsync: vi.fn(() => Promise.reject(new Error("the file is cursed"))),
        });
        const user = userEvent.setup();
        const { container } = renderPage(gameMasterUser);

        // when
        const input = container.querySelector<HTMLInputElement>("input[accept='.pdf,.txt,.docx']")!;
        await user.upload(input, new File(["evidence"], "notes.pdf", { type: "application/pdf" }));

        // then
        expect(await screen.findByText("the file is cursed")).toBeInTheDocument();
    });

    it("asks before removing an attachment", async () => {
        // given
        const { deleteAttachment } = stubMystery({
            mystery: makeMysteryDetail({
                attachments: [{ id: 3, file_url: "/files/notes.pdf", file_name: "notes.pdf", file_size: 512 }],
            }),
        });
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByTitle("Delete attachment"));

        // then
        expect(confirm).toHaveBeenCalledWith('Delete attachment "notes.pdf"?');
        expect(deleteAttachment).not.toHaveBeenCalled();
    });

    it("shows the media gallery and lets the game master remove an image", async () => {
        // given
        const { deleteMedia } = stubMystery({
            mystery: makeMysteryDetail({
                media: [{ id: 11, media_url: "/m/11.png", media_type: "image", sort_order: 0 }],
            }),
        });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Remove image #11" }));

        // then
        expect(screen.getByTestId("media-gallery")).toHaveTextContent("1 media");
        expect(deleteMedia).toHaveBeenCalledWith(11);
    });

    it("uploads the images the game master has queued up", async () => {
        // given
        const { uploadMedia, refresh } = stubMystery();
        const user = userEvent.setup();
        const { container } = renderPage(gameMasterUser);

        // when
        const input = container.querySelector<HTMLInputElement>("input[accept='image/*,video/*,.mkv,.avi']")!;
        await user.upload(input, new File(["png"], "scene.png", { type: "image/png" }));
        await user.click(screen.getByRole("button", { name: "Upload 1" }));

        // then
        expect(uploadMedia).toHaveBeenCalledWith(expect.any(File));
        await waitFor(() => {
            expect(refresh).toHaveBeenCalled();
        });
    });
});
