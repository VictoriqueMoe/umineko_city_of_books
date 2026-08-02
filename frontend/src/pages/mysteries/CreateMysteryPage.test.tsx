import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { MysteryDetail } from "../../types/api";
import { CreateMysteryPage } from "./CreateMysteryPage";

const {
    useMystery,
    useCreateMystery,
    useUpdateMystery,
    useDeleteMysteryMedia,
    useUploadMysteryAttachmentToAny,
    useUploadMysteryMediaToAny,
    navigate,
} = vi.hoisted(() => ({
    useMystery: vi.fn(),
    useCreateMystery: vi.fn(),
    useUpdateMystery: vi.fn(),
    useDeleteMysteryMedia: vi.fn(),
    useUploadMysteryAttachmentToAny: vi.fn(),
    useUploadMysteryMediaToAny: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/mystery", () => ({ useMystery }));
vi.mock("../../api/mutations/mystery", () => ({
    useCreateMystery,
    useUpdateMystery,
    useDeleteMysteryMedia,
    useUploadMysteryAttachmentToAny,
    useUploadMysteryMediaToAny,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

function makeMysteryDetail(overrides: Partial<MysteryDetail> = {}): MysteryDetail {
    return {
        id: "mystery-1",
        title: "The sealed guest room",
        body: "Six people died behind a chained door.",
        difficulty: "hard",
        author: { id: "gm-1", username: "beatrice", display_name: "Beatrice" },
        solved: false,
        paused: false,
        gm_away: false,
        free_for_all: false,
        keep_open_after_solve: false,
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

interface StubOptions {
    mystery?: MysteryDetail | null;
    loading?: boolean;
    create?: () => Promise<{ id: string }>;
    update?: () => Promise<unknown>;
}

function stubMystery(options: StubOptions = {}) {
    useMystery.mockReturnValue({
        mystery: options.mystery ?? null,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
    const createAsync = vi.fn(options.create ?? (() => Promise.resolve({ id: "mystery-9" })));
    const updateAsync = vi.fn(options.update ?? (() => Promise.resolve({})));
    const deleteMediaAsync = vi.fn(() => Promise.resolve({}));
    const uploadAttachmentAsync = vi.fn(() => Promise.resolve({}));
    const uploadMediaAsync = vi.fn(() => Promise.resolve({}));
    useCreateMystery.mockReturnValue({ mutateAsync: createAsync });
    useUpdateMystery.mockReturnValue({ mutateAsync: updateAsync });
    useDeleteMysteryMedia.mockReturnValue({ mutateAsync: deleteMediaAsync });
    useUploadMysteryAttachmentToAny.mockReturnValue({ mutateAsync: uploadAttachmentAsync });
    useUploadMysteryMediaToAny.mockReturnValue({ mutateAsync: uploadMediaAsync });

    return { createAsync, updateAsync, deleteMediaAsync, uploadAttachmentAsync, uploadMediaAsync };
}

function renderCreatePage() {
    return renderWithProviders(<CreateMysteryPage />, { route: "/mystery/new" });
}

function renderEditPage() {
    return renderWithProviders(<CreateMysteryPage />, { route: "/mystery/mystery-1/edit", path: "/mystery/:id/edit" });
}

function makeFile(name: string, type: string, size = 8): File {
    const file = new File(["beatrice"], name, { type });
    Object.defineProperty(file, "size", { value: size });

    return file;
}

async function fillScenario(user: ReturnType<typeof userEvent.setup>) {
    await user.type(screen.getByPlaceholderText("Mystery title..."), "  The sealed guest room  ");
    await user.type(
        screen.getByPlaceholderText(
            "Write your mystery scenario... Set the scene, introduce the characters, present the puzzle.",
        ),
        "  Six people died.  ",
    );
}

describe("CreateMysteryPage", () => {
    it("greets a new game master with the guide", () => {
        // given
        stubMystery();

        // when
        renderCreatePage();

        // then
        expect(screen.getByRole("heading", { name: "Create a Mystery" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "Game Master's Guide" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Present Mystery" })).toBeDisabled();
    });

    it("unlocks the submit button only once a title and a scenario exist", async () => {
        // given
        stubMystery();
        const user = userEvent.setup();
        renderCreatePage();

        // when
        await user.type(screen.getByPlaceholderText("Mystery title..."), "A title");

        // then
        expect(screen.getByRole("button", { name: "Present Mystery" })).toBeDisabled();
        await user.type(
            screen.getByPlaceholderText(
                "Write your mystery scenario... Set the scene, introduce the characters, present the puzzle.",
            ),
            "A scenario",
        );
        expect(screen.getByRole("button", { name: "Present Mystery" })).toBeEnabled();
    });

    it("sends a trimmed scenario with the default difficulty and no empty clues", async () => {
        // given
        const { createAsync } = stubMystery();
        const user = userEvent.setup();
        renderCreatePage();
        await fillScenario(user);

        // when
        await user.click(screen.getByRole("button", { name: "Present Mystery" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({
            title: "The sealed guest room",
            body: "Six people died.",
            difficulty: "medium",
            free_for_all: false,
            keep_open_after_solve: false,
            clues: [],
        });
    });

    it("carries the chosen difficulty and both game modes into the payload", async () => {
        // given
        const { createAsync } = stubMystery();
        const user = userEvent.setup();
        renderCreatePage();
        await fillScenario(user);
        await user.selectOptions(screen.getByDisplayValue("Medium"), "nightmare");
        await user.click(screen.getByRole("switch", { name: "Free-for-all mode" }));
        await user.click(screen.getByRole("switch", { name: "Ongoing mode" }));

        // when
        await user.click(screen.getByRole("button", { name: "Present Mystery" }));

        // then
        expect(createAsync).toHaveBeenCalledWith(
            expect.objectContaining({ difficulty: "nightmare", free_for_all: true, keep_open_after_solve: true }),
        );
    });

    it("keeps only the clues that were actually written", async () => {
        // given
        const { createAsync } = stubMystery();
        const user = userEvent.setup();
        renderCreatePage();
        await fillScenario(user);
        await user.click(screen.getByRole("button", { name: "+ Add Clue" }));
        await user.type(screen.getByPlaceholderText("Red truth #1..."), "The door was chained");
        await user.selectOptions(screen.getAllByDisplayValue("Red")[0], "purple");

        // when
        await user.click(screen.getByRole("button", { name: "Present Mystery" }));

        // then
        expect(createAsync).toHaveBeenCalledWith(
            expect.objectContaining({ clues: [{ body: "The door was chained", truth_type: "purple" }] }),
        );
    });

    it("adds and removes clue rows", async () => {
        // given
        stubMystery();
        const user = userEvent.setup();
        renderCreatePage();

        // when
        await user.click(screen.getByRole("button", { name: "+ Add Clue" }));

        // then
        expect(screen.getByPlaceholderText("Red truth #2...")).toBeInTheDocument();
        await user.click(screen.getAllByRole("button", { name: "✕" })[1]);
        expect(screen.queryByPlaceholderText("Red truth #2...")).not.toBeInTheDocument();
    });

    it("does not present the mystery when a clue row is removed", async () => {
        // given
        const { createAsync } = stubMystery();
        const user = userEvent.setup();
        renderCreatePage();
        await fillScenario(user);
        await user.click(screen.getByRole("button", { name: "+ Add Clue" }));

        // when
        await user.click(screen.getAllByRole("button", { name: "✕" })[1]);

        // then
        expect(createAsync).not.toHaveBeenCalled();
        expect(navigate).not.toHaveBeenCalled();
        expect(screen.queryByPlaceholderText("Red truth #2...")).not.toBeInTheDocument();
    });

    it("offers no way to remove the only clue row", () => {
        // given
        stubMystery();

        // when
        renderCreatePage();

        // then
        expect(screen.queryByRole("button", { name: "✕" })).not.toBeInTheDocument();
    });

    it("opens the newly presented mystery", async () => {
        // given
        stubMystery({ create: () => Promise.resolve({ id: "mystery-77" }) });
        const user = userEvent.setup();
        renderCreatePage();
        await fillScenario(user);

        // when
        await user.click(screen.getByRole("button", { name: "Present Mystery" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/mystery/mystery-77");
        });
    });

    it("reports why the mystery could not be presented", async () => {
        // given
        stubMystery({ create: () => Promise.reject(new Error("the game board is full")) });
        const user = userEvent.setup();
        renderCreatePage();
        await fillScenario(user);

        // when
        await user.click(screen.getByRole("button", { name: "Present Mystery" }));

        // then
        expect(await screen.findByText("the game board is full")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("uploads every attached file against the mystery it just created", async () => {
        // given
        const { uploadAttachmentAsync, uploadMediaAsync } = stubMystery({
            create: () => Promise.resolve({ id: "mystery-5" }),
        });
        const user = userEvent.setup();
        const { container } = renderCreatePage();
        await fillScenario(user);
        const inputs = container.querySelectorAll<HTMLInputElement>("input[type='file']");
        await user.upload(inputs[0], makeFile("scene.png", "image/png"));
        await user.upload(inputs[1], makeFile("notes.txt", "text/plain", 2048));

        // when
        await user.click(screen.getByRole("button", { name: "Present Mystery" }));

        // then
        await waitFor(() => {
            expect(uploadAttachmentAsync).toHaveBeenCalledWith(
                expect.objectContaining({ mysteryId: "mystery-5", file: expect.any(File) }),
            );
        });
        expect(uploadMediaAsync).toHaveBeenCalledWith(
            expect.objectContaining({ mysteryId: "mystery-5", file: expect.any(File) }),
        );
    });

    it("lists an attachment with a human readable size and lets it be dropped", async () => {
        // given
        stubMystery();
        const user = userEvent.setup();
        const { container } = renderCreatePage();
        const inputs = container.querySelectorAll<HTMLInputElement>("input[type='file']");

        // when
        await user.upload(inputs[1], makeFile("notes.txt", "text/plain", 2048));

        // then
        expect(screen.getByText("notes.txt")).toBeInTheDocument();
        expect(screen.getByText("2.0 KB")).toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: "×" }));
        expect(screen.queryByText("notes.txt")).not.toBeInTheDocument();
    });

    it("returns to the mystery list when the composer is abandoned", async () => {
        // given
        stubMystery();
        const user = userEvent.setup();
        renderCreatePage();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/mysteries");
    });

    it("waits on the form while an existing mystery is fetched for editing", () => {
        // given
        stubMystery({ loading: true });

        // when
        renderEditPage();

        // then
        expect(screen.getByText("Loading mystery...")).toBeInTheDocument();
    });

    it("seeds the editor with the mystery as it stands and drops the guide", () => {
        // given
        stubMystery({
            mystery: makeMysteryDetail({
                free_for_all: true,
                clues: [
                    { id: 1, body: "The door was chained", truth_type: "red", sort_order: 0 },
                    { id: 2, body: "Only for Battler", truth_type: "red", sort_order: 1, player_id: "player-1" },
                ],
            }),
        });

        // when
        renderEditPage();

        // then
        expect(screen.getByRole("heading", { name: "Edit Mystery" })).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Game Master's Guide" })).not.toBeInTheDocument();
        expect(screen.getByDisplayValue("The sealed guest room")).toBeInTheDocument();
        expect(screen.getByDisplayValue("Six people died behind a chained door.")).toBeInTheDocument();
        expect(screen.getByDisplayValue("Hard")).toBeInTheDocument();
        expect(screen.getByRole("switch", { name: "Free-for-all mode" })).toHaveAttribute("aria-checked", "true");
        expect(screen.getByDisplayValue("The door was chained")).toBeInTheDocument();
        expect(screen.queryByDisplayValue("Only for Battler")).not.toBeInTheDocument();
    });

    it("saves the edits and goes back to the mystery", async () => {
        // given
        const { updateAsync } = stubMystery({ mystery: makeMysteryDetail() });
        const user = userEvent.setup();
        renderEditPage();
        await user.clear(screen.getByDisplayValue("The sealed guest room"));
        await user.type(screen.getByPlaceholderText("Mystery title..."), "A locked room after all");

        // when
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        expect(updateAsync).toHaveBeenCalledWith(
            expect.objectContaining({ title: "A locked room after all", difficulty: "hard" }),
        );
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/mystery/mystery-1");
        });
    });

    it("removes the images that were marked for removal when the edit is saved", async () => {
        // given
        const { deleteMediaAsync } = stubMystery({
            mystery: makeMysteryDetail({
                media: [{ id: 11, media_url: "/m/11.png", media_type: "image", sort_order: 0 }],
            }),
        });
        const user = userEvent.setup();
        renderEditPage();

        // when
        await user.click(screen.getByRole("button", { name: "Remove on save" }));
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        await waitFor(() => {
            expect(deleteMediaAsync).toHaveBeenCalledWith(11);
        });
    });

    it("lets a marked image be spared again before saving", async () => {
        // given
        const { deleteMediaAsync } = stubMystery({
            mystery: makeMysteryDetail({
                media: [{ id: 11, media_url: "/m/11.png", media_type: "image", sort_order: 0 }],
            }),
        });
        const user = userEvent.setup();
        renderEditPage();

        // when
        await user.click(screen.getByRole("button", { name: "Remove on save" }));
        await user.click(screen.getByRole("button", { name: "Undo remove" }));
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/mystery/mystery-1");
        });
        expect(deleteMediaAsync).not.toHaveBeenCalled();
    });

    it("returns to the mystery when the edit is abandoned", async () => {
        // given
        stubMystery({ mystery: makeMysteryDetail() });
        const user = userEvent.setup();
        renderEditPage();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/mystery/mystery-1");
    });

    it("reports why the edit could not be saved", async () => {
        // given
        stubMystery({ mystery: makeMysteryDetail(), update: () => Promise.reject(new Error("the board is sealed")) });
        const user = userEvent.setup();
        renderEditPage();

        // when
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        expect(await screen.findByText("the board is sealed")).toBeInTheDocument();
    });
});
