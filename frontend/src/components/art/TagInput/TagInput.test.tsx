import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { TagInput } from "./TagInput";

interface HarnessProps {
    initialTags?: string[];
    maxTags?: number;
    onTagsChange?: (tags: string[]) => void;
}

function TagHarness({ initialTags = [], maxTags, onTagsChange }: HarnessProps) {
    const [tags, setTags] = useState<string[]>(initialTags);

    function handleChange(next: string[]) {
        setTags(next);
        onTagsChange?.(next);
    }

    return <TagInput tags={tags} onChange={handleChange} maxTags={maxTags} />;
}

function tagField(): HTMLInputElement {
    return screen.getByRole("textbox");
}

describe("TagInput", () => {
    it("adds a tag when enter is pressed", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "beatrice{Enter}");

        // then
        expect(onTagsChange).toHaveBeenCalledWith(["beatrice"]);
        expect(screen.getByText("beatrice")).toBeInTheDocument();
    });

    it("adds a tag when a comma is typed", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "epitaph,");

        // then
        expect(onTagsChange).toHaveBeenCalledWith(["epitaph"]);
        expect(tagField()).toHaveValue("");
    });

    it("lowercases the tag and strips characters that are not allowed", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "  Golden Land! 1986_x-y  {Enter}");

        // then
        expect(onTagsChange).toHaveBeenCalledWith(["goldenland1986_x-y"]);
    });

    it("ignores an entry that is only whitespace", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "   {Enter}");

        // then
        expect(onTagsChange).not.toHaveBeenCalled();
    });

    it("ignores an entry made entirely of stripped characters", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "!!!{Enter}");

        // then
        expect(onTagsChange).not.toHaveBeenCalled();
    });

    it("refuses a duplicate tag and keeps what was typed", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness initialTags={["beatrice"]} onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "Beatrice{Enter}");

        // then
        expect(onTagsChange).not.toHaveBeenCalled();
        expect(tagField()).toHaveValue("Beatrice");
        expect(screen.getAllByText("beatrice")).toHaveLength(1);
    });

    it("stops accepting tags once the maximum is reached", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness initialTags={["one"]} maxTags={2} onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "two{Enter}");

        // then
        expect(onTagsChange).toHaveBeenCalledWith(["one", "two"]);
        expect(tagField()).toBeDisabled();
        expect(screen.getByPlaceholderText("Max tags reached")).toBeInTheDocument();
    });

    it("defaults the maximum to ten tags", () => {
        // given
        const tags = ["a", "b", "c", "d", "e", "f", "g", "h", "i"];

        // when
        renderWithProviders(<TagHarness initialTags={tags} />);

        // then
        expect(screen.getByPlaceholderText("Add tag...")).toBeEnabled();
    });

    it("disables the field when ten tags have been added", () => {
        // given
        const tags = ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j"];

        // when
        renderWithProviders(<TagHarness initialTags={tags} />);

        // then
        expect(screen.getByPlaceholderText("Max tags reached")).toBeDisabled();
    });

    it("removes the last tag on backspace when the field is empty", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness initialTags={["beatrice", "epitaph"]} onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "{Backspace}");

        // then
        expect(onTagsChange).toHaveBeenCalledWith(["beatrice"]);
        expect(screen.queryByText("epitaph")).not.toBeInTheDocument();
    });

    it("leaves the tags alone on backspace while the field still has text", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness initialTags={["beatrice"]} onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "ab{Backspace}");

        // then
        expect(onTagsChange).not.toHaveBeenCalled();
        expect(screen.getByText("beatrice")).toBeInTheDocument();
    });

    it("does nothing on backspace when there are no tags left", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness onTagsChange={onTagsChange} />);

        // when
        await user.type(tagField(), "{Backspace}");

        // then
        expect(onTagsChange).not.toHaveBeenCalled();
    });

    it("removes a tag through its own remove control", async () => {
        // given
        const user = userEvent.setup();
        const onTagsChange = vi.fn();
        renderWithProviders(<TagHarness initialTags={["beatrice", "epitaph"]} onTagsChange={onTagsChange} />);

        // when
        await user.click(screen.getAllByRole("button", { name: "×" })[0]);

        // then
        expect(onTagsChange).toHaveBeenCalledWith(["epitaph"]);
        expect(screen.queryByText("beatrice")).not.toBeInTheDocument();
    });

    it("clears the field after a tag has been accepted", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<TagHarness />);

        // when
        await user.type(tagField(), "beatrice{Enter}");

        // then
        expect(tagField()).toHaveValue("");
    });
});
