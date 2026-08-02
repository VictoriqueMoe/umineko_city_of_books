import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CharacterListEntry, OCSummary, ShipCharacter } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { CharacterPicker } from "./CharacterPicker";

const mocks = vi.hoisted(() => ({
    useCharacterList: vi.fn(),
    useUserOCSummaries: vi.fn(),
}));

vi.mock("../../api/queries/character", () => ({ useCharacterList: mocks.useCharacterList }));
vi.mock("../../api/queries/oc", () => ({ useUserOCSummaries: mocks.useUserOCSummaries }));

const author = makeUser({ id: "user-1" });

const groupedCast: CharacterListEntry[] = [
    { id: "kanon", name: "Kanon", group: "additional" },
    { id: "beatrice", name: "Beatrice", group: "main" },
    { id: "battler", name: "Battler Ushiromiya", group: "main" },
];

const flatCast: CharacterListEntry[] = [
    { id: "virgilia", name: "Virgilia" },
    { id: "ange", name: "Ange" },
];

const savedOCs: OCSummary[] = [{ id: "oc-1", name: "Clair", series: "umineko" }];

function renderPicker(onAdd: (character: ShipCharacter) => void, existing: ShipCharacter[] = [], max?: number) {
    return renderWithProviders(<CharacterPicker onAdd={onAdd} existing={existing} maxCharacters={max} />, {
        user: author,
    });
}

describe("CharacterPicker", () => {
    beforeEach(() => {
        mocks.useCharacterList.mockReturnValue({ characters: groupedCast, loading: false });
        mocks.useUserOCSummaries.mockReturnValue({ summaries: [], loading: false });
    });

    it("replaces the picker with a notice once the character limit is reached", () => {
        // given
        const existing: ShipCharacter[] = [
            { series: "umineko", character_id: "battler", character_name: "Battler Ushiromiya", sort_order: 0 },
            { series: "umineko", character_id: "beatrice", character_name: "Beatrice", sort_order: 1 },
        ];

        // when
        renderPicker(vi.fn(), existing, 2);

        // then
        expect(screen.getByText("Maximum 2 characters reached.")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Add" })).not.toBeInTheDocument();
    });

    it("asks for the canon list of the series that is on show", async () => {
        // given
        const user = userEvent.setup();
        renderPicker(vi.fn());
        expect(mocks.useCharacterList).toHaveBeenLastCalledWith("umineko", true);

        // when
        await user.click(screen.getByRole("button", { name: "Higurashi" }));

        // then
        expect(mocks.useCharacterList).toHaveBeenLastCalledWith("higurashi", true);
    });

    it("stops asking for a canon list on the OC tab", async () => {
        // given
        const user = userEvent.setup();
        renderPicker(vi.fn());

        // when
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // then
        expect(mocks.useCharacterList).toHaveBeenLastCalledWith("oc", false);
    });

    it("reads the signed in author's own saved OCs", () => {
        // given
        const onAdd = vi.fn();

        // when
        renderPicker(onAdd);

        // then
        expect(mocks.useUserOCSummaries).toHaveBeenCalledWith("user-1", "user-1");
    });

    it("lists canon characters in alphabetical order", () => {
        // given
        mocks.useCharacterList.mockReturnValue({ characters: flatCast, loading: false });

        // when
        renderPicker(vi.fn());

        // then
        const labels = screen.getAllByRole("option").map(option => option.textContent);
        expect(labels).toEqual(["-- choose a character --", "Ange", "Virgilia"]);
    });

    it("separates the main cast from the additional cast when there is one", () => {
        // given
        mocks.useCharacterList.mockReturnValue({ characters: groupedCast, loading: false });

        // when
        renderPicker(vi.fn());

        // then
        expect(screen.getByRole("group", { name: "Main cast" })).toBeInTheDocument();
        expect(screen.getByRole("group", { name: "Additional" })).toBeInTheDocument();
    });

    it("locks the picker while the canon list is still loading", () => {
        // given
        mocks.useCharacterList.mockReturnValue({ characters: [], loading: true });

        // when
        renderPicker(vi.fn());

        // then
        expect(screen.getByRole("combobox")).toBeDisabled();
        expect(screen.getByRole("option", { name: "Loading..." })).toBeInTheDocument();
    });

    it("keeps the add button disabled until a canon character is chosen", async () => {
        // given
        const user = userEvent.setup();
        renderPicker(vi.fn());
        expect(screen.getByRole("button", { name: "Add" })).toBeDisabled();

        // when
        await user.selectOptions(screen.getByRole("combobox"), "beatrice");

        // then
        expect(screen.getByRole("button", { name: "Add" })).toBeEnabled();
    });

    it("adds the chosen canon character at the end of the existing cast", async () => {
        // given
        const onAdd = vi.fn();
        const existing: ShipCharacter[] = [
            { series: "higurashi", character_id: "keiichi", character_name: "Keiichi Maebara", sort_order: 0 },
        ];
        const user = userEvent.setup();
        renderPicker(onAdd, existing);

        // when
        await user.selectOptions(screen.getByRole("combobox"), "battler");
        await user.click(screen.getByRole("button", { name: "Add" }));

        // then
        expect(onAdd).toHaveBeenCalledWith({
            series: "umineko",
            character_id: "battler",
            character_name: "Battler Ushiromiya",
            sort_order: 1,
        });
    });

    it("clears the selection after a character has been added", async () => {
        // given
        const user = userEvent.setup();
        renderPicker(vi.fn());
        await user.selectOptions(screen.getByRole("combobox"), "battler");

        // when
        await user.click(screen.getByRole("button", { name: "Add" }));

        // then
        expect(screen.getByRole("combobox")).toHaveValue("");
        expect(screen.getByRole("button", { name: "Add" })).toBeDisabled();
    });

    it("refuses to add a canon character that is already in the cast", async () => {
        // given
        const onAdd = vi.fn();
        const existing: ShipCharacter[] = [
            { series: "umineko", character_id: "beatrice", character_name: "Beatrice", sort_order: 0 },
        ];
        const user = userEvent.setup();
        renderPicker(onAdd, existing);

        // when
        await user.selectOptions(screen.getByRole("combobox"), "beatrice");
        await user.click(screen.getByRole("button", { name: "Add" }));

        // then
        expect(onAdd).not.toHaveBeenCalled();
    });

    it("forgets the current selection when the series is switched", async () => {
        // given
        const user = userEvent.setup();
        renderPicker(vi.fn());
        await user.selectOptions(screen.getByRole("combobox"), "battler");

        // when
        await user.click(screen.getByRole("button", { name: "Higurashi" }));

        // then
        expect(screen.getByRole("combobox")).toHaveValue("");
        expect(screen.getByRole("button", { name: "Add" })).toBeDisabled();
    });

    it("invites a plain name when the author has no saved OCs", async () => {
        // given
        const user = userEvent.setup();
        renderPicker(vi.fn());

        // when
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // then
        expect(screen.getByPlaceholderText("Character name...")).toBeInTheDocument();
        expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    });

    it("offers the author's saved OCs alongside a one-off name", async () => {
        // given
        mocks.useUserOCSummaries.mockReturnValue({ summaries: savedOCs, loading: false });
        const user = userEvent.setup();
        renderPicker(vi.fn());

        // when
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // then
        expect(screen.getByRole("option", { name: "Clair" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Or type a one-off OC name...")).toBeInTheDocument();
    });

    it("adds a saved OC with the identifier it already has", async () => {
        // given
        mocks.useUserOCSummaries.mockReturnValue({ summaries: savedOCs, loading: false });
        const onAdd = vi.fn();
        const user = userEvent.setup();
        renderPicker(onAdd);
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // when
        await user.selectOptions(screen.getByRole("combobox"), "oc-1");
        await user.click(screen.getByRole("button", { name: "Add" }));

        // then
        expect(onAdd).toHaveBeenCalledWith({
            series: "oc",
            character_id: "oc-1",
            character_name: "Clair",
            sort_order: 0,
        });
    });

    it("adds a one-off OC by name with no identifier", async () => {
        // given
        const onAdd = vi.fn();
        const user = userEvent.setup();
        renderPicker(onAdd);
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // when
        await user.type(screen.getByPlaceholderText("Character name..."), "  Sayo  ");
        await user.click(screen.getByRole("button", { name: "Add" }));

        // then
        expect(onAdd).toHaveBeenCalledWith({ series: "oc", character_name: "Sayo", sort_order: 0 });
        expect(screen.getByPlaceholderText("Character name...")).toHaveValue("");
    });

    it("adds the typed OC when the author presses enter", async () => {
        // given
        const onAdd = vi.fn();
        const user = userEvent.setup();
        renderPicker(onAdd);
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // when
        await user.type(screen.getByPlaceholderText("Character name..."), "Sayo{Enter}");

        // then
        expect(onAdd).toHaveBeenCalledWith({ series: "oc", character_name: "Sayo", sort_order: 0 });
    });

    it("refuses to add a saved OC that is already in the cast", async () => {
        // given
        mocks.useUserOCSummaries.mockReturnValue({ summaries: savedOCs, loading: false });
        const onAdd = vi.fn();
        const existing: ShipCharacter[] = [
            { series: "oc", character_id: "oc-1", character_name: "Clair", sort_order: 0 },
        ];
        const user = userEvent.setup();
        renderPicker(onAdd, existing);
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // when
        await user.selectOptions(screen.getByRole("combobox"), "oc-1");
        await user.click(screen.getByRole("button", { name: "Add" }));

        // then
        expect(onAdd).not.toHaveBeenCalled();
    });

    it("refuses a one-off OC whose name is already in the cast as a saved OC", async () => {
        // given
        const onAdd = vi.fn();
        const existing: ShipCharacter[] = [
            { series: "oc", character_id: "oc-1", character_name: "Clair", sort_order: 0 },
        ];
        const user = userEvent.setup();
        renderPicker(onAdd, existing);
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // when
        await user.type(screen.getByPlaceholderText("Character name..."), "clair");
        await user.click(screen.getByRole("button", { name: "Add" }));

        // then
        expect(onAdd).not.toHaveBeenCalled();
    });

    it("refuses a one-off OC whose name is already in the cast whatever its casing", async () => {
        // given
        const onAdd = vi.fn();
        const existing: ShipCharacter[] = [{ series: "oc", character_name: "Sayo", sort_order: 0 }];
        const user = userEvent.setup();
        renderPicker(onAdd, existing);
        await user.click(screen.getByRole("button", { name: "OC / Other" }));

        // when
        await user.type(screen.getByPlaceholderText("Character name..."), "sayo");
        await user.click(screen.getByRole("button", { name: "Add" }));

        // then
        expect(onAdd).not.toHaveBeenCalled();
    });

    it("drops the saved OC choice as soon as a name is typed instead", async () => {
        // given
        mocks.useUserOCSummaries.mockReturnValue({ summaries: savedOCs, loading: false });
        const user = userEvent.setup();
        renderPicker(vi.fn());
        await user.click(screen.getByRole("button", { name: "OC / Other" }));
        await user.selectOptions(screen.getByRole("combobox"), "oc-1");

        // when
        await user.type(screen.getByPlaceholderText("Or type a one-off OC name..."), "Sayo");

        // then
        expect(screen.getByRole("combobox")).toHaveValue("");
    });

    it("keeps the add button disabled on the OC tab until there is something to add", async () => {
        // given
        const user = userEvent.setup();
        renderPicker(vi.fn());
        await user.click(screen.getByRole("button", { name: "OC / Other" }));
        expect(screen.getByRole("button", { name: "Add" })).toBeDisabled();

        // when
        await user.type(screen.getByPlaceholderText("Character name..."), "   ");

        // then
        expect(screen.getByRole("button", { name: "Add" })).toBeDisabled();
    });
});
