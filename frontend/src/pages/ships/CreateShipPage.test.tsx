import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { ShipCharacter } from "../../types/api";
import { CreateShipPage } from "./CreateShipPage";

const mocks = vi.hoisted(() => ({
    createShip: vi.fn(),
    uploadImage: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/mutations/ship", () => ({
    useCreateShip: () => ({ mutateAsync: mocks.createShip }),
    useUploadShipImageById: () => ({ mutateAsync: mocks.uploadImage }),
}));

vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => mocks.navigate };
});

vi.mock("../../components/CharacterPicker/CharacterPicker", () => ({
    CharacterPicker: ({ onAdd }: { onAdd: (c: ShipCharacter) => void }) => (
        <div>
            <button
                onClick={() =>
                    onAdd({ series: "umineko", character_id: "battler", character_name: "Battler", sort_order: 0 })
                }
            >
                pick Battler
            </button>
            <button
                onClick={() =>
                    onAdd({ series: "umineko", character_id: "beatrice", character_name: "Beatrice", sort_order: 0 })
                }
            >
                pick Beatrice
            </button>
            <button onClick={() => onAdd({ series: "oc", character_name: "Featherine Junior", sort_order: 0 })}>
                pick an OC
            </button>
        </div>
    ),
}));

vi.mock("../../components/MentionTextArea/MentionTextArea", () => ({
    MentionTextArea: ({
        value,
        onChange,
        placeholder,
    }: {
        value: string;
        onChange: (v: string) => void;
        placeholder?: string;
    }) => <textarea placeholder={placeholder} value={value} onChange={e => onChange(e.target.value)} />,
}));

function renderPage() {
    return renderWithProviders(<CreateShipPage />, { user: makeUser({ id: "me" }), route: "/ships/new" });
}

function titleBox(): HTMLElement {
    return screen.getByPlaceholderText("e.g. Battler × Beatrice");
}

function declareButton(): HTMLElement {
    return screen.getByRole("button", { name: "Declare Ship" });
}

function fileInput(container: HTMLElement): HTMLInputElement {
    const input = container.querySelector<HTMLInputElement>('input[type="file"]');
    if (!input) {
        throw new Error("the form has no file input");
    }
    return input;
}

function imageFile(name = "ship.png"): File {
    return new File(["butterflies"], name, { type: "image/png" });
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
    await user.type(titleBox(), "Battler × Beatrice");
    await user.click(screen.getByRole("button", { name: "pick Battler" }));
    await user.click(screen.getByRole("button", { name: "pick Beatrice" }));
}

beforeEach(() => {
    mocks.createShip.mockResolvedValue({ id: "ship-9" });
    mocks.uploadImage.mockResolvedValue({});
});

describe("CreateShipPage form gating", () => {
    it("keeps the declare button out of reach on an empty form", () => {
        // given
        renderPage();

        // when
        const button = declareButton();

        // then
        expect(button).toBeDisabled();
    });

    it("still refuses to submit with a title but only one character", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.type(titleBox(), "Battler × Beatrice");
        await user.click(screen.getByRole("button", { name: "pick Battler" }));

        // then
        expect(declareButton()).toBeDisabled();
    });

    it("treats a title of only whitespace as no title at all", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByRole("button", { name: "pick Battler" }));
        await user.click(screen.getByRole("button", { name: "pick Beatrice" }));

        // when
        await user.type(titleBox(), "   ");

        // then
        expect(declareButton()).toBeDisabled();
    });

    it("opens the declare button once a title and two characters are in place", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await fillValidForm(user);

        // then
        expect(declareButton()).toBeEnabled();
    });
});

describe("CreateShipPage character list", () => {
    it("shows every character that has been picked", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await fillValidForm(user);

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("drops a character again when its remove cross is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);

        // when
        await user.click(screen.getAllByRole("button", { name: "Remove character" })[0]);

        // then
        expect(screen.queryByText("Battler")).not.toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("renumbers the remaining characters after one is removed", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);
        await user.click(screen.getByRole("button", { name: "pick an OC" }));

        // when
        await user.click(screen.getAllByRole("button", { name: "Remove character" })[0]);
        await user.click(declareButton());

        // then
        expect(mocks.createShip).toHaveBeenCalledWith(
            expect.objectContaining({
                characters: [
                    { series: "umineko", character_id: "beatrice", character_name: "Beatrice", sort_order: 0 },
                    { series: "oc", character_name: "Featherine Junior", sort_order: 1 },
                ],
            }),
        );
    });
});

describe("CreateShipPage submitting", () => {
    it("sends the trimmed title and description with the picked characters", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);
        await user.type(screen.getByPlaceholderText("Tell us why this pairing works..."), "  they deserve it  ");

        // when
        await user.click(declareButton());

        // then
        expect(mocks.createShip).toHaveBeenCalledWith({
            title: "Battler × Beatrice",
            description: "they deserve it",
            characters: [
                { series: "umineko", character_id: "battler", character_name: "Battler", sort_order: 0 },
                { series: "umineko", character_id: "beatrice", character_name: "Beatrice", sort_order: 1 },
            ],
        });
    });

    it("opens the freshly declared ship once it is saved", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);

        // when
        await user.click(declareButton());

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/ships/ship-9");
        });
    });

    it("uploads the chosen image against the new ship before leaving", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderPage();
        await fillValidForm(user);
        const file = imageFile();
        await user.upload(fileInput(container), file);

        // when
        await user.click(declareButton());

        // then
        await waitFor(() => {
            expect(mocks.uploadImage).toHaveBeenCalledWith({ id: "ship-9", file });
        });
    });

    it("uploads nothing when no image was chosen", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);

        // when
        await user.click(declareButton());

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/ships/ship-9");
        });
        expect(mocks.uploadImage).not.toHaveBeenCalled();
    });

    it("still opens the ship when the image upload fails", async () => {
        // given
        mocks.uploadImage.mockRejectedValue(new Error("the disk is full"));
        const user = userEvent.setup();
        const { container } = renderPage();
        await fillValidForm(user);
        await user.upload(fileInput(container), imageFile());

        // when
        await user.click(declareButton());

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/ships/ship-9");
        });
    });

    it("reports why the ship could not be declared", async () => {
        // given
        mocks.createShip.mockRejectedValue(new Error("that pairing already exists"));
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);

        // when
        await user.click(declareButton());

        // then
        expect(await screen.findByText("that pairing already exists")).toBeInTheDocument();
        expect(mocks.navigate).not.toHaveBeenCalled();
    });

    it("falls back to a generic message when the failure carries no reason", async () => {
        // given
        mocks.createShip.mockRejectedValue("boom");
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);

        // when
        await user.click(declareButton());

        // then
        expect(await screen.findByText("Failed to create ship")).toBeInTheDocument();
    });

    it("frees the declare button again after a failure", async () => {
        // given
        mocks.createShip.mockRejectedValue(new Error("nope"));
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);

        // when
        await user.click(declareButton());

        // then
        await waitFor(() => {
            expect(declareButton()).toBeEnabled();
        });
    });
});

describe("CreateShipPage image picking", () => {
    it("previews the chosen image and offers to drop it", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderPage();

        // when
        await user.upload(fileInput(container), imageFile());

        // then
        expect(screen.getByRole("img", { name: "preview" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
    });

    it("clears the preview when the image is dropped", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderPage();
        await user.upload(fileInput(container), imageFile());

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(screen.queryByRole("img", { name: "preview" })).not.toBeInTheDocument();
    });

    it("sends no image after the chosen one has been dropped", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderPage();
        await fillValidForm(user);
        await user.upload(fileInput(container), imageFile());
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // when
        await user.click(declareButton());

        // then
        await waitFor(() => {
            expect(mocks.navigate).toHaveBeenCalledWith("/ships/ship-9");
        });
        expect(mocks.uploadImage).not.toHaveBeenCalled();
    });
});

describe("CreateShipPage leaving", () => {
    it("returns to the ship list from the back link", async () => {
        // given
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("← All Ships"));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/ships");
    });

    it("returns to the ship list from cancel without saving anything", async () => {
        // given
        const user = userEvent.setup();
        renderPage();
        await fillValidForm(user);

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(mocks.navigate).toHaveBeenCalledWith("/ships");
        expect(mocks.createShip).not.toHaveBeenCalled();
    });
});
