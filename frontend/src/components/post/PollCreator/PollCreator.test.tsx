import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { PollCreator } from "./PollCreator";

function noop() {}

interface SetupOptions {
    options?: string[];
    duration?: number;
    onOptionsChange?: (options: string[]) => void;
    onDurationChange?: (seconds: number) => void;
    onRemove?: () => void;
}

function setup(overrides: SetupOptions = {}) {
    return renderWithProviders(
        <PollCreator
            options={overrides.options ?? ["Beatrice", "Battler"]}
            duration={overrides.duration ?? 86400}
            onOptionsChange={overrides.onOptionsChange ?? noop}
            onDurationChange={overrides.onDurationChange ?? noop}
            onRemove={overrides.onRemove ?? noop}
        />,
    );
}

function manyOptions(count: number): string[] {
    const out: string[] = [];
    for (let i = 0; i < count; i++) {
        out.push(`Choice ${i + 1}`);
    }
    return out;
}

describe("PollCreator", () => {
    it("renders a numbered field for every option it is given", () => {
        // given
        const options = ["Beatrice", "Battler", "Erika"];

        // when
        setup({ options });

        // then
        expect(screen.getByPlaceholderText("Option 1")).toHaveValue("Beatrice");
        expect(screen.getByPlaceholderText("Option 2")).toHaveValue("Battler");
        expect(screen.getByPlaceholderText("Option 3")).toHaveValue("Erika");
    });

    it("reports the edited option without disturbing its neighbours", async () => {
        // given
        const onOptionsChange = vi.fn();
        const user = userEvent.setup();
        setup({ options: ["Beatrice", "Battler"], onOptionsChange });

        // when
        await user.type(screen.getByPlaceholderText("Option 1"), "!");

        // then
        expect(onOptionsChange).toHaveBeenCalledWith(["Beatrice!", "Battler"]);
    });

    it("limits how much text an option may hold", () => {
        // given
        const options = ["Beatrice", "Battler"];

        // when
        setup({ options });

        // then
        expect(screen.getByPlaceholderText("Option 1")).toHaveAttribute("maxlength", "200");
    });

    it("offers no way to drop an option while only the minimum two remain", () => {
        // given
        const options = ["Beatrice", "Battler"];

        // when
        setup({ options });

        // then
        expect(screen.queryAllByRole("button", { name: "Remove option" })).toHaveLength(0);
    });

    it("offers a remove control per option once there are more than two", () => {
        // given
        const options = ["Beatrice", "Battler", "Erika"];

        // when
        setup({ options });

        // then
        expect(screen.getAllByRole("button", { name: "Remove option" })).toHaveLength(3);
    });

    it("drops only the option whose remove control was pressed", async () => {
        // given
        const onOptionsChange = vi.fn();
        const user = userEvent.setup();
        setup({ options: ["Beatrice", "Battler", "Erika"], onOptionsChange });

        // when
        await user.click(screen.getAllByRole("button", { name: "Remove option" })[1]);

        // then
        expect(onOptionsChange).toHaveBeenCalledWith(["Beatrice", "Erika"]);
    });

    it("appends an empty option when another one is added", async () => {
        // given
        const onOptionsChange = vi.fn();
        const user = userEvent.setup();
        setup({ options: ["Beatrice", "Battler"], onOptionsChange });

        // when
        await user.click(screen.getByRole("button", { name: "+ Add Option" }));

        // then
        expect(onOptionsChange).toHaveBeenCalledWith(["Beatrice", "Battler", ""]);
    });

    it("still allows a tenth option to be added", () => {
        // given
        const options = manyOptions(9);

        // when
        setup({ options });

        // then
        expect(screen.getByRole("button", { name: "+ Add Option" })).toBeEnabled();
    });

    it("refuses to add an eleventh option", () => {
        // given
        const options = manyOptions(10);

        // when
        setup({ options });

        // then
        expect(screen.getByRole("button", { name: "+ Add Option" })).toBeDisabled();
    });

    it("shows the current duration as the chosen one", () => {
        // given
        const duration = 604800;

        // when
        setup({ duration });

        // then
        expect(screen.getByRole("combobox")).toHaveValue("604800");
        expect(screen.getByRole("option", { name: "1 week" })).toBeInTheDocument();
    });

    it("reports a newly chosen duration as a number of seconds", async () => {
        // given
        const onDurationChange = vi.fn();
        const user = userEvent.setup();
        setup({ duration: 86400, onDurationChange });

        // when
        await user.selectOptions(screen.getByRole("combobox"), "3600");

        // then
        expect(onDurationChange).toHaveBeenCalledWith(3600);
    });

    it("asks its owner to remove the whole poll", async () => {
        // given
        const onRemove = vi.fn();
        const user = userEvent.setup();
        setup({ onRemove });

        // when
        await user.click(screen.getByRole("button", { name: "Remove Poll" }));

        // then
        expect(onRemove).toHaveBeenCalledOnce();
    });
});
