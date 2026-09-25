import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { MysteryAttempt, MysteryDetail, UserProfile } from "../../types/api";
import { MysteryDetailPage } from "./MysteryDetailPage";

const mocked = vi.hoisted(() => ({
    useMystery: vi.fn(),
    navigate: vi.fn(),
    mutations: {
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
    },
}));

vi.mock("../../hooks/queries/mystery", () => ({ useMystery: mocked.useMystery }));
vi.mock("../../hooks/mutations/mystery", () => mocked.mutations);
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocked.navigate };
});

interface AttemptStubProps {
    attempt: MysteryAttempt;
    authorAlreadyWon: boolean;
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

type MutationHook = keyof typeof mocked.mutations;

interface ToggleCase {
    name: string;
    overrides: Partial<MysteryDetail>;
    button: string;
    hook: MutationHook;
    value: boolean;
    absent: string;
}

const gameMaster = { id: "gm-1", username: "beatrice", display_name: "Beatrice" };
const player = { id: "player-1", username: "battler", display_name: "Battler" };
const otherPlayer = { id: "player-2", username: "ange", display_name: "Ange" };

const gameMasterUser = makeUser({ id: "gm-1", username: "beatrice", display_name: "Beatrice" });
const playerUser = makeUser({ id: "player-1", username: "battler", display_name: "Battler" });
const moderatorUser = makeUser({ id: "mod-1", username: "virgilia", display_name: "Virgilia", role: "moderator" });

const composerPlaceholder = "Declare your blue truth...";

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

function stubMystery(overrides: Partial<MysteryDetail> | null = {}, loading = false) {
    const refresh = vi.fn();
    mocked.useMystery.mockReturnValue({
        mystery: overrides === null ? null : makeMysteryDetail(overrides),
        loading,
        refresh,
    });

    const mutateAsync = {} as Record<MutationHook, Mock>;
    for (const hook of Object.keys(mocked.mutations) as MutationHook[]) {
        mutateAsync[hook] = vi.fn(() => Promise.resolve({}));
        mocked.mutations[hook].mockReturnValue({ mutateAsync: mutateAsync[hook] });
    }

    return { refresh, mutateAsync };
}

function renderPage(user: UserProfile | null = null) {
    return renderWithProviders(<MysteryDetailPage />, {
        user,
        route: "/mystery/mystery-1",
        path: "/mystery/:id",
    });
}

describe("MysteryDetailPage", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it.each([
        { name: "investigates while the mystery is loading", loading: true, message: "Investigating the mystery..." },
        { name: "says the mystery is missing when the server has none", loading: false, message: "Mystery not found." },
    ])("$name", ({ loading, message }) => {
        // given
        stubMystery(null, loading);

        // when
        renderPage();

        // then
        expect(screen.getByText(message)).toBeInTheDocument();
    });

    it.each([
        {
            name: "presents the scenario with its difficulty and player count",
            playerCount: 2,
            badge: "2 pieces attempting",
        },
        { name: "uses the singular when only one piece is attempting", playerCount: 1, badge: "1 piece attempting" },
    ])("$name", ({ playerCount, badge }) => {
        // given
        stubMystery({ player_count: playerCount });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("heading", { name: "The sealed guest room" })).toBeInTheDocument();
        expect(screen.getByText("Six people died behind a chained door.")).toBeInTheDocument();
        expect(screen.getByText("hard")).toBeInTheDocument();
        expect(screen.getByText("Open")).toBeInTheDocument();
        expect(screen.getByText(badge)).toBeInTheDocument();
    });

    it("celebrates the winner, closes the composer and opens the post game discussion once the mystery is solved", () => {
        // given
        stubMystery({ solved: true, winner: player });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText(/Mystery solved! Winner:/)).toHaveTextContent("Mystery solved! Winner: Battler");
        expect(screen.getByText("Solved")).toBeInTheDocument();
        expect(screen.queryByPlaceholderText(composerPlaceholder)).not.toBeInTheDocument();
        expect(screen.getByRole("region", { name: "comments" })).toHaveTextContent(
            "Post-Game Discussion for mystery-1",
        );
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
        expect(screen.queryByPlaceholderText(composerPlaceholder)).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("keeps the game master's tools and the post game discussion away from a piece on an open mystery", () => {
        // given
        stubMystery();

        // when
        renderPage(playerUser);

        // then
        expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Pause" })).not.toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Add a new red truth clue...")).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Attachments" })).not.toBeInTheDocument();
        expect(screen.queryByRole("region", { name: "comments" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Report" })).toBeInTheDocument();
    });

    it("gives the game master the board controls but no composer, and no close button unless the mystery is ongoing", async () => {
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
        expect(screen.queryByRole("button", { name: "Mark Permanently Solved" })).not.toBeInTheDocument();
        expect(screen.queryByPlaceholderText(composerPlaceholder)).not.toBeInTheDocument();
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
        const { mutateAsync } = stubMystery();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this mystery? This cannot be undone.");
        expect(mutateAsync.useDeleteMystery).toHaveBeenCalledWith("mystery-1");
        await waitFor(() => {
            expect(mocked.navigate).toHaveBeenCalledWith("/mysteries");
        });
    });

    it.each<ToggleCase>([
        {
            name: "pauses the mystery and leaves the refetch to the mutation",
            overrides: {},
            button: "Pause",
            hook: "useSetMysteryPaused",
            value: true,
            absent: "Resume",
        },
        {
            name: "offers to resume a paused mystery and hides the away toggle meanwhile",
            overrides: { paused: true },
            button: "Resume",
            hook: "useSetMysteryPaused",
            value: false,
            absent: "Mark as away",
        },
        {
            name: "lets the game master step away and come back",
            overrides: { gm_away: true },
            button: "I'm back",
            hook: "useSetMysteryGmAway",
            value: false,
            absent: "Mark as away",
        },
    ])("$name", async ({ overrides, button, hook, value, absent }) => {
        // given
        const { mutateAsync, refresh } = stubMystery(overrides);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: button }));

        // then
        expect(mutateAsync[hook]).toHaveBeenCalledWith(value);
        expect(refresh).not.toHaveBeenCalled();
        expect(screen.queryByRole("button", { name: absent })).not.toBeInTheDocument();
    });

    it("closes an ongoing mystery once the game master confirms", async () => {
        // given
        const { mutateAsync, refresh } = stubMystery({ keep_open_after_solve: true });
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Mark Permanently Solved" }));

        // then
        expect(confirm).toHaveBeenCalled();
        await waitFor(() => {
            expect(mutateAsync.useCloseMystery).toHaveBeenCalled();
        });
        expect(refresh).not.toHaveBeenCalled();
    });

    it("lists the global red truths, leaves the private ones out and copies one to the clipboard", async () => {
        // given
        stubMystery({
            clues: [
                { id: 1, body: "The door was chained", truth_type: "red", sort_order: 0 },
                { id: 2, body: "Only Ange may know this", truth_type: "red", sort_order: 1, player_id: otherPlayer.id },
            ],
        });
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
        const user = userEvent.setup();

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("heading", { name: "Red Truths" })).toBeInTheDocument();
        expect(screen.getByText("The door was chained")).toBeInTheDocument();
        expect(screen.queryByText("Only Ange may know this")).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Copy to clipboard" }));

        // then
        expect(writeText).toHaveBeenCalledWith("The door was chained");
    });

    it("lets the game master declare a new global red truth", async () => {
        // given
        const { mutateAsync, refresh } = stubMystery();
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // then
        expect(screen.getByRole("button", { name: "Add global Red Truth" })).toBeDisabled();

        // when
        await user.type(screen.getByPlaceholderText("Add a new red truth clue..."), "  The window was latched  ");
        await user.click(screen.getByRole("button", { name: "Add global Red Truth" }));

        // then
        expect(mutateAsync.useAddMysteryClue).toHaveBeenCalledWith({
            body: "The window was latched",
            truthType: "red",
        });
        await waitFor(() => {
            expect(screen.getByPlaceholderText("Add a new red truth clue...")).toHaveValue("");
        });
        expect(refresh).not.toHaveBeenCalled();
    });

    it("lets the game master amend a player's private red truths and whisper a new one to them", async () => {
        // given
        const { mutateAsync, refresh } = stubMystery({
            attempts: [makeAttempt({ id: "a1", body: "First guess" })],
            clues: [
                { id: 5, body: "Only Battler may know this", truth_type: "red", sort_order: 0, player_id: player.id },
            ],
        });
        const user = userEvent.setup();

        // when
        renderPage(gameMasterUser);

        // then
        expect(screen.getByText("Only Battler may know this")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "edit" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "delete" })).toBeInTheDocument();

        // when
        await user.type(screen.getByPlaceholderText("Private red truth for this player..."), "  Your key is a lie  ");
        await user.click(screen.getByRole("button", { name: "Add private Red Truth" }));

        // then
        await waitFor(() => {
            expect(mutateAsync.useAddMysteryClue).toHaveBeenCalledWith({
                body: "Your key is a lie",
                truthType: "red",
                playerId: "player-1",
            });
        });
        expect(refresh).not.toHaveBeenCalled();
    });

    it("groups the attempts by player for the game master, offers jump pills and folds a thread away", async () => {
        // given
        stubMystery({
            attempts: [
                makeAttempt({ id: "a1", body: "First guess" }),
                makeAttempt({ id: "a2", body: "Second guess" }),
                makeAttempt({ id: "a3", author: otherPlayer, body: "Ange's guess" }),
            ],
        });
        const user = userEvent.setup();

        // when
        renderPage(gameMasterUser);

        // then
        expect(screen.getByText("Blue Truth Attempts (3)")).toBeInTheDocument();
        expect(screen.getByTitle("Jump to Battler's attempts")).toBeInTheDocument();
        expect(screen.getByTitle("Jump to Ange's attempts")).toBeInTheDocument();
        expect(screen.getByText("2 attempts")).toBeInTheDocument();
        expect(screen.getByText("1 attempt")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: /1 attempt/ }));

        // then
        expect(screen.queryByText("Ange's guess")).not.toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: /1 attempt/ }));
        expect(screen.getByText("Ange's guess")).toBeInTheDocument();
    });

    it("shows a piece only a flat thread with no pills in a private mystery", () => {
        // given
        stubMystery({ attempts: [makeAttempt({ id: "a1", body: "First guess" })] });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText("First guess")).toBeInTheDocument();
        expect(screen.queryByTitle("Jump to Battler's attempts")).not.toBeInTheDocument();
        expect(screen.queryByText("1 attempt")).not.toBeInTheDocument();
    });

    it("shows every piece the grouped attempts in a free-for-all but keeps the private red truth composer away", () => {
        // given
        stubMystery({
            free_for_all: true,
            attempts: [makeAttempt({ id: "a1" }), makeAttempt({ id: "a2", author: otherPlayer })],
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByTitle("Jump to Ange's attempts")).toBeInTheDocument();
        expect(screen.getAllByRole("article")).toHaveLength(2);
        expect(screen.queryByPlaceholderText("Private red truth for this player...")).not.toBeInTheDocument();
    });

    it("reveals every attempt once the mystery is solved and pins the winning one", () => {
        // given
        stubMystery({
            solved: true,
            winner: player,
            attempts: [
                makeAttempt({ id: "a1", body: "The chain trick", is_winner: true }),
                makeAttempt({ id: "a2", author: otherPlayer, body: "A wrong guess" }),
            ],
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
            attempts: [
                makeAttempt({ id: "a1", body: "The chain trick", is_winner: true }),
                makeAttempt({ id: "a2", author: otherPlayer, body: "A wrong guess" }),
            ],
        });

        // when
        renderPage(gameMasterUser);

        // then
        const battler = screen.getByRole("article", { name: "attempt by Battler" });
        const ange = screen.getByRole("article", { name: "attempt by Ange" });
        expect(within(battler).getByText("author already won")).toBeInTheDocument();
        expect(within(ange).getByText("author has not won")).toBeInTheDocument();
    });

    it.each([
        {
            name: "tells the game master that nobody has moved yet",
            viewer: gameMasterUser,
            playerCount: 0,
            message: "No attempts yet. Waiting for pieces to make their move.",
        },
        {
            name: "invites the first blue truth when nobody is playing yet",
            viewer: playerUser,
            playerCount: 0,
            message: "No attempts yet. Be the first to declare your blue truth!",
        },
        {
            name: "tells a piece how many others are already playing a private mystery",
            viewer: playerUser,
            playerCount: 3,
            message: "There are 3 pieces playing this mystery. Join the game board and declare your own blue truth!",
        },
    ])("$name", ({ viewer, playerCount, message }) => {
        // given
        stubMystery({ attempts: [], player_count: playerCount });

        // when
        renderPage(viewer);

        // then
        expect(screen.getByText(message)).toBeInTheDocument();
    });

    it("submits a trimmed blue truth and clears the composer", async () => {
        // given
        const { mutateAsync, refresh } = stubMystery();
        const user = userEvent.setup();
        renderPage(playerUser);

        // then
        expect(screen.getByRole("button", { name: "Submit Blue Truth" })).toBeDisabled();

        // when
        await user.type(screen.getByPlaceholderText(composerPlaceholder), "  The chain was faked  ");
        await user.click(screen.getByRole("button", { name: "Submit Blue Truth" }));

        // then
        expect(mutateAsync.useCreateMysteryAttempt).toHaveBeenCalledWith({ body: "The chain was faked" });
        await waitFor(() => {
            expect(screen.getByPlaceholderText(composerPlaceholder)).toHaveValue("");
        });
        expect(refresh).not.toHaveBeenCalled();
    });

    it.each([
        {
            name: "closes the composer and explains the pause to the pieces",
            overrides: { paused: true },
            banner: "The Game Master has paused this mystery. New attempts are temporarily disabled.",
            composers: 0,
        },
        {
            name: "warns that the game master is away but still takes attempts",
            overrides: { gm_away: true },
            banner: "The Game Master is currently away. You can still post theories, but responses may be delayed.",
            composers: 1,
        },
    ])("$name", ({ overrides, banner, composers }) => {
        // given
        stubMystery(overrides);

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByText(banner)).toBeInTheDocument();
        expect(screen.queryAllByPlaceholderText(composerPlaceholder)).toHaveLength(composers);
    });

    it("shows a piece the private red truths written for them and lets them fold them away", async () => {
        // given
        stubMystery({
            clues: [{ id: 5, body: "Your key never left you", truth_type: "red", sort_order: 0, player_id: player.id }],
        });
        const user = userEvent.setup();

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("button", { name: /Private Red Truths \(to you\) \(1\)/ })).toBeInTheDocument();
        expect(screen.getByText("Your key never left you")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: /Private Red Truths \(to you\)/ }));

        // then
        expect(screen.queryByText("Your key never left you")).not.toBeInTheDocument();
        expect(localStorage.getItem("mystery:mystery-1:private-clues:player-1:collapsed")).toBe("1");
    });

    it("lists the attachments with a readable size", () => {
        // given
        stubMystery({
            attachments: [{ id: 3, file_url: "/files/notes.pdf", file_name: "notes.pdf", file_size: 2048 }],
        });

        // when
        renderPage(playerUser);

        // then
        expect(screen.getByRole("link", { name: "notes.pdf" })).toHaveAttribute("href", "/files/notes.pdf");
        expect(screen.getByText("2.0 KB")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Add Attachment" })).not.toBeInTheDocument();
    });

    it("lets the game master attach a file to the mystery", async () => {
        // given
        const { mutateAsync, refresh } = stubMystery();
        const user = userEvent.setup();
        const { container } = renderPage(gameMasterUser);

        // when
        const input = container.querySelector<HTMLInputElement>("input[accept='.pdf,.txt,.docx']")!;
        await user.upload(input, new File(["evidence"], "notes.pdf", { type: "application/pdf" }));

        // then
        await waitFor(() => {
            expect(mutateAsync.useUploadMysteryAttachment).toHaveBeenCalledWith(expect.any(File));
        });
        expect(refresh).not.toHaveBeenCalled();
    });

    it("reports why an attachment could not be uploaded", async () => {
        // given
        const { mutateAsync } = stubMystery();
        mutateAsync.useUploadMysteryAttachment.mockRejectedValue(new Error("the file is cursed"));
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
        const { mutateAsync } = stubMystery({
            attachments: [{ id: 3, file_url: "/files/notes.pdf", file_name: "notes.pdf", file_size: 512 }],
        });
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByTitle("Delete attachment"));

        // then
        expect(confirm).toHaveBeenCalledWith('Delete attachment "notes.pdf"?');
        expect(mutateAsync.useDeleteMysteryAttachment).not.toHaveBeenCalled();
    });

    it("shows the media gallery and lets the game master remove an image", async () => {
        // given
        const { mutateAsync } = stubMystery({
            media: [{ id: 11, media_url: "/m/11.png", media_type: "image", sort_order: 0 }],
        });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage(gameMasterUser);

        // when
        await user.click(screen.getByRole("button", { name: "Remove image #11" }));

        // then
        expect(screen.getByTestId("media-gallery")).toHaveTextContent("1 media");
        expect(mutateAsync.useDeleteMysteryMedia).toHaveBeenCalledWith(11);
    });

    it("uploads the images the game master has queued up", async () => {
        // given
        const { mutateAsync, refresh } = stubMystery();
        const user = userEvent.setup();
        const { container } = renderPage(gameMasterUser);

        // when
        const input = container.querySelector<HTMLInputElement>("input[accept='image/*,video/*,audio/*,.mkv,.avi']")!;
        await user.upload(input, new File(["png"], "scene.png", { type: "image/png" }));
        await user.click(screen.getByRole("button", { name: "Upload 1" }));

        // then
        await waitFor(() => {
            expect(mutateAsync.useUploadMysteryMedia).toHaveBeenCalledWith({
                file: expect.any(File),
                isSpoiler: false,
            });
        });
        expect(refresh).not.toHaveBeenCalled();
    });
});
