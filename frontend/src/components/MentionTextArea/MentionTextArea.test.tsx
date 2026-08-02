import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useRef, useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { User } from "../../types/api";
import { MentionTextArea, type MentionTextAreaHandle } from "./MentionTextArea";

const { search } = vi.hoisted(() => ({ search: { fetchSearchUsers: vi.fn() } }));

vi.mock("../../api/queries/misc", () => ({ fetchSearchUsers: search.fetchSearchUsers }));

const PLACEHOLDER = "Write something";

const POOL: User[] = [
    { id: "u1", username: "beatrice", display_name: "Beatrice", avatar_url: "https://img.example/beato.png" },
    { id: "u2", username: "lambdadelta", display_name: "Lambdadelta" },
    { id: "u3", username: "bern", display_name: "Golden Witch" },
];

const CROWD: User[] = Array.from({ length: 12 }, (_, i) => ({
    id: `crowd-${i}`,
    username: `stakes${i}a`,
    display_name: `Stake ${i}`,
}));

interface SearchResult extends User {
    viewer_follows: boolean;
    follows_viewer: boolean;
}

function makeResult(overrides: Partial<SearchResult> = {}): SearchResult {
    return {
        id: "s1",
        username: "beatrice",
        display_name: "Beatrice",
        viewer_follows: false,
        follows_viewer: false,
        ...overrides,
    };
}

interface HarnessProps {
    initial?: string;
    onChange?: (value: string) => void;
    rows?: number;
    mentionPool?: User[];
    showColours?: boolean;
    colourBarOpen?: boolean;
    onPasteFiles?: (files: File[]) => void;
}

function Harness({ initial = "", onChange, ...rest }: HarnessProps) {
    const [value, setValue] = useState(initial);

    function handleChange(next: string) {
        setValue(next);
        if (onChange) {
            onChange(next);
        }
    }

    return <MentionTextArea value={value} onChange={handleChange} placeholder={PLACEHOLDER} {...rest} />;
}

function FocusHarness() {
    const handle = useRef<MentionTextAreaHandle>(null);
    const [value, setValue] = useState("");

    return (
        <>
            <button onClick={() => handle.current?.focus()}>focus the box</button>
            <MentionTextArea ref={handle} value={value} onChange={setValue} placeholder={PLACEHOLDER} />
        </>
    );
}

function box(): HTMLTextAreaElement {
    const textarea = screen.getByPlaceholderText(PLACEHOLDER);
    if (!(textarea instanceof HTMLTextAreaElement)) {
        throw new Error("expected the mention box to be a textarea");
    }

    return textarea;
}

function suggestions(): HTMLElement[] {
    return screen.queryAllByRole("button").filter(button => !button.hasAttribute("aria-label"));
}

async function settle(ms = 30) {
    await act(async () => {
        await new Promise(resolve => {
            setTimeout(resolve, ms);
        });
    });
}

function makeFile(name: string, type: string, size = 16): File {
    const file = new File(["evidence"], name, { type });
    Object.defineProperty(file, "size", { value: size });

    return file;
}

describe("MentionTextArea", () => {
    beforeEach(() => {
        search.fetchSearchUsers.mockResolvedValue([]);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("shows the placeholder and the number of rows it was asked for", () => {
        // given
        const rows = 5;

        // when
        renderWithProviders(<Harness rows={rows} />);

        // then
        expect(box()).toHaveAttribute("rows", "5");
        expect(box()).toHaveAttribute("placeholder", PLACEHOLDER);
    });

    it("mirrors what is typed into the highlight layer behind the box", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<Harness />);

        // when
        await user.type(box(), "without love");

        // then
        expect(container.textContent).toContain("without love");
    });

    it("highlights a mention that stands on its own", () => {
        // given
        const initial = "hello @beatrice";

        // when
        const { container } = renderWithProviders(<Harness initial={initial} />);

        // then
        expect(container.querySelector(".mention-hl")?.textContent).toBe("@beatrice");
    });

    it("leaves an @ that is stuck to a word alone", () => {
        // given
        const initial = "mail@beatrice";

        // when
        const { container } = renderWithProviders(<Harness initial={initial} />);

        // then
        expect(container.querySelector(".mention-hl")).toBeNull();
    });

    it("paints a colour tag and keeps its brackets visible", () => {
        // given
        const initial = "[red]the truth[/red]";

        // when
        const { container } = renderWithProviders(<Harness initial={initial} />);

        // then
        expect(container.querySelector(".red-truth")?.textContent).toBe("the truth");
        const brackets = Array.from(container.querySelectorAll(".tag-bracket")).map(el => el.textContent);
        expect(brackets).toEqual(["[red]", "[/red]"]);
    });

    it("highlights a mention that sits inside a colour tag", () => {
        // given
        const initial = "[blue]ask @beatrice[/blue]";

        // when
        const { container } = renderWithProviders(<Harness initial={initial} />);

        // then
        expect(container.querySelector(".blue-truth .mention-hl")?.textContent).toBe("@beatrice");
    });

    it("escapes markup so nothing can be smuggled into the highlight layer", () => {
        // given
        const initial = "<img src=x onerror=alert(1)>";

        // when
        const { container } = renderWithProviders(<Harness initial={initial} />);

        // then
        expect(container.querySelector("img")).toBeNull();
        expect(container.innerHTML).toContain("&lt;img");
    });

    it("suggests nobody until the trigger character is typed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);

        // when
        await user.type(box(), "beatrice");
        await settle();

        // then
        expect(suggestions()).toHaveLength(0);
    });

    it("waits for at least one character after the trigger", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);

        // when
        await user.type(box(), "@");
        await settle();

        // then
        expect(suggestions()).toHaveLength(0);
    });

    it("suggests from the pool it was handed without troubling the server", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);

        // when
        await user.type(box(), "@bea");

        // then
        expect(await screen.findByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("@beatrice")).toBeInTheDocument();
        expect(search.fetchSearchUsers).not.toHaveBeenCalled();
    });

    it("matches on the display name as well as the username", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);

        // when
        await user.type(box(), "@golden");

        // then
        expect(await screen.findByText("Golden Witch")).toBeInTheDocument();
        expect(suggestions()).toHaveLength(1);
    });

    it("pays no attention to the case of what was typed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);

        // when
        await user.type(box(), "@BEA");

        // then
        expect(await screen.findByText("Beatrice")).toBeInTheDocument();
    });

    it("offers no more than eight names from a crowded pool", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={CROWD} />);

        // when
        await user.type(box(), "@a");
        await screen.findByText("Stake 0");

        // then
        expect(suggestions()).toHaveLength(8);
    });

    it("keeps quiet when the trigger follows a word", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);

        // when
        await user.type(box(), "mail@bea");
        await settle();

        // then
        expect(suggestions()).toHaveLength(0);
    });

    it("gives up once the query runs into a space", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@bea");
        await screen.findByText("Beatrice");

        // when
        await user.type(box(), " t");
        await settle();

        // then
        expect(suggestions()).toHaveLength(0);
    });

    it("shows an avatar for whoever has one and an initial for whoever does not", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<Harness mentionPool={POOL} />);

        // when
        await user.type(box(), "@a");
        await screen.findByText("Lambdadelta");

        // then
        expect(container.querySelectorAll("img")).toHaveLength(1);
        expect(screen.getByText("L")).toBeInTheDocument();
    });

    it("splices the chosen name into the text", async () => {
        // given
        const onChange = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} onChange={onChange} />);
        await user.type(box(), "hi @bea");
        await screen.findByText("Beatrice");

        // when
        await user.click(suggestions()[0]);

        // then
        expect(box()).toHaveValue("hi @beatrice ");
        expect(onChange).toHaveBeenLastCalledWith("hi @beatrice ");
    });

    it("keeps the rest of the line when a mention is chosen mid sentence", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness initial="hi @b there" mentionPool={POOL} />);

        // when
        await user.type(box(), "e", { initialSelectionStart: 5, initialSelectionEnd: 5 });
        await screen.findByText("Beatrice");
        await user.click(suggestions()[0]);

        // then
        expect(box()).toHaveValue("hi @beatrice there");
    });

    it("leaves the caret in front of the space that was already there", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness initial="hi @b there" mentionPool={POOL} />);

        // when
        await user.type(box(), "e", { initialSelectionStart: 5, initialSelectionEnd: 5 });
        await screen.findByText("Beatrice");
        await user.click(suggestions()[0]);

        // then
        await vi.waitFor(() => expect(box().selectionStart).toBe(12));
    });

    it("adds no space when the rest of the line starts on a new one", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness initial={"hi @b\nthere"} mentionPool={POOL} />);

        // when
        await user.type(box(), "e", { initialSelectionStart: 5, initialSelectionEnd: 5 });
        await screen.findByText("Beatrice");
        await user.click(suggestions()[0]);

        // then
        expect(box()).toHaveValue("hi @beatrice\nthere");
    });

    it("puts the suggestions away when the box loses focus", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@bea");
        await screen.findByText("Beatrice");

        // when
        await user.click(document.body);

        // then
        expect(suggestions()).toHaveLength(0);
    });

    it("leaves the caret just past the name it inserted", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "hi @bea");
        await screen.findByText("Beatrice");

        // when
        await user.click(suggestions()[0]);

        // then
        await vi.waitFor(() => expect(box().selectionStart).toBe(13));
    });

    it("puts the suggestions away once one has been chosen", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@bea");
        await screen.findByText("Beatrice");

        // when
        await user.click(suggestions()[0]);

        // then
        expect(suggestions()).toHaveLength(0);
    });

    it("walks down the suggestions with the down arrow", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@a");
        await screen.findByText("Lambdadelta");

        // when
        await user.keyboard("{ArrowDown}");

        // then
        expect(suggestions()[1].className).toContain("suggestionActive");
        expect(suggestions()[0].className).not.toContain("suggestionActive");
    });

    it("wraps back round to the first suggestion at the end of the list", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@a");
        await screen.findByText("Lambdadelta");

        // when
        await user.keyboard("{ArrowDown}{ArrowDown}");

        // then
        expect(suggestions()[0].className).toContain("suggestionActive");
    });

    it("wraps round to the last suggestion when arrowing up from the top", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@a");
        await screen.findByText("Lambdadelta");

        // when
        await user.keyboard("{ArrowUp}");

        // then
        expect(suggestions()[1].className).toContain("suggestionActive");
    });

    it("highlights whichever suggestion the pointer moves over", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@a");
        await screen.findByText("Lambdadelta");

        // when
        await user.hover(suggestions()[1]);

        // then
        expect(suggestions()[1].className).toContain("suggestionActive");
    });

    it("takes the highlighted suggestion when enter is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@a");
        await screen.findByText("Lambdadelta");

        // when
        await user.keyboard("{ArrowDown}{Enter}");

        // then
        expect(box()).toHaveValue("@lambdadelta ");
    });

    it("takes the highlighted suggestion when tab is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@a");
        await screen.findByText("Beatrice");

        // when
        await user.keyboard("{Tab}");

        // then
        expect(box()).toHaveValue("@beatrice ");
    });

    it("puts the suggestions away when escape is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness mentionPool={POOL} />);
        await user.type(box(), "@a");
        await screen.findByText("Beatrice");

        // when
        await user.keyboard("{Escape}");

        // then
        expect(suggestions()).toHaveLength(0);
    });

    it("asks the server once the typing has settled when it has no pool", async () => {
        // given
        const user = userEvent.setup();
        search.fetchSearchUsers.mockResolvedValue([makeResult()]);
        renderWithProviders(<Harness />);

        // when
        await user.type(box(), "@beat");
        await screen.findByText("Beatrice");

        // then
        expect(search.fetchSearchUsers).toHaveBeenCalledOnce();
        expect(search.fetchSearchUsers).toHaveBeenCalledWith("beat");
    });

    it("puts the people you both follow at the top of the search results", async () => {
        // given
        const user = userEvent.setup();
        search.fetchSearchUsers.mockResolvedValue([
            makeResult({ id: "s1", username: "stranger", display_name: "Stranger" }),
            makeResult({ id: "s2", username: "fan", display_name: "Fan", follows_viewer: true }),
            makeResult({
                id: "s3",
                username: "mutual",
                display_name: "Mutual",
                viewer_follows: true,
                follows_viewer: true,
            }),
            makeResult({ id: "s4", username: "idol", display_name: "Idol", viewer_follows: true }),
        ]);
        renderWithProviders(<Harness />);

        // when
        await user.type(box(), "@s");
        await screen.findByText("Mutual");

        // then
        expect(suggestions().map(button => button.textContent)).toEqual([
            "MMutual@mutualYou follow each other",
            "IIdol@idolFollowing",
            "FFan@fanFollows you",
            "SStranger@stranger",
        ]);
    });

    it("keeps its head when the search fails", async () => {
        // given
        const user = userEvent.setup();
        search.fetchSearchUsers.mockRejectedValue(new Error("the server is asleep"));
        renderWithProviders(<Harness />);

        // when
        await user.type(box(), "@bea");
        await settle(200);

        // then
        expect(suggestions()).toHaveLength(0);
    });

    it("offers a colour button for each truth when colours are switched on", () => {
        // given
        const showColours = true;

        // when
        renderWithProviders(<Harness showColours={showColours} />);

        // then
        const labels = ["Red truth", "Blue truth", "Gold truth", "Purple truth", "Green text", "Pink text"];
        for (const label of labels) {
            expect(screen.getByRole("button", { name: label })).toBeInTheDocument();
        }
    });

    it("hides the colour bar while it is folded away", () => {
        // given
        const colourBarOpen = false;

        // when
        renderWithProviders(<Harness showColours colourBarOpen={colourBarOpen} />);

        // then
        expect(screen.queryByRole("button", { name: "Red truth" })).not.toBeInTheDocument();
    });

    it("leaves the colour bar out entirely when colours are switched off", () => {
        // given
        const showColours = false;

        // when
        renderWithProviders(<Harness showColours={showColours} />);

        // then
        expect(screen.queryByRole("button", { name: "Red truth" })).not.toBeInTheDocument();
    });

    it("wraps the selection in the colour that was pressed", () => {
        // given
        const onChange = vi.fn();
        renderWithProviders(<Harness initial="the truth" showColours onChange={onChange} />);
        box().setSelectionRange(0, 9);

        // when
        fireEvent.mouseDown(screen.getByRole("button", { name: "Red truth" }));

        // then
        expect(onChange).toHaveBeenCalledWith("[red]the truth[/red]");
    });

    it("opens an empty pair of tags when nothing is selected", () => {
        // given
        const onChange = vi.fn();
        renderWithProviders(<Harness initial="" showColours onChange={onChange} />);
        box().setSelectionRange(0, 0);

        // when
        fireEvent.mouseDown(screen.getByRole("button", { name: "Gold truth" }));

        // then
        expect(onChange).toHaveBeenCalledWith("[gold][/gold]");
    });

    it("strips the colour again when the selection already sits inside it", () => {
        // given
        const onChange = vi.fn();
        renderWithProviders(<Harness initial="[red]the truth[/red]" showColours onChange={onChange} />);
        box().setSelectionRange(5, 14);

        // when
        fireEvent.mouseDown(screen.getByRole("button", { name: "Red truth" }));

        // then
        expect(onChange).toHaveBeenCalledWith("the truth");
    });

    it("strips the colour again when the whole tagged run is selected", () => {
        // given
        const onChange = vi.fn();
        renderWithProviders(<Harness initial="[red]the truth[/red]" showColours onChange={onChange} />);
        box().setSelectionRange(0, 20);

        // when
        fireEvent.mouseDown(screen.getByRole("button", { name: "Red truth" }));

        // then
        expect(onChange).toHaveBeenCalledWith("the truth");
    });

    it("nests a different colour inside one that is already there", () => {
        // given
        const onChange = vi.fn();
        renderWithProviders(<Harness initial="[red]the truth[/red]" showColours onChange={onChange} />);
        box().setSelectionRange(5, 14);

        // when
        fireEvent.mouseDown(screen.getByRole("button", { name: "Blue truth" }));

        // then
        expect(onChange).toHaveBeenCalledWith("[red][blue]the truth[/blue][/red]");
    });

    it("keeps the wrapped words selected so they can be coloured again", async () => {
        // given
        renderWithProviders(<Harness initial="the truth" showColours />);
        box().setSelectionRange(0, 9);

        // when
        fireEvent.mouseDown(screen.getByRole("button", { name: "Red truth" }));

        // then
        await vi.waitFor(() => expect(box().selectionStart).toBe(5));
        expect(box().selectionEnd).toBe(14);
    });

    it("takes pasted pictures and clips", () => {
        // given
        const onPasteFiles = vi.fn();
        const picture = makeFile("beatrice.png", "image/png");
        const clip = makeFile("tea.mp4", "video/mp4");
        renderWithProviders(<Harness onPasteFiles={onPasteFiles} />);

        // when
        fireEvent.paste(box(), { clipboardData: { files: [picture, clip] } });

        // then
        expect(onPasteFiles).toHaveBeenCalledWith([picture, clip]);
    });

    it("ignores a pasted file that is neither a picture nor a clip", () => {
        // given
        const onPasteFiles = vi.fn();
        const note = makeFile("notes.txt", "text/plain");
        renderWithProviders(<Harness onPasteFiles={onPasteFiles} />);

        // when
        fireEvent.paste(box(), { clipboardData: { files: [note] } });

        // then
        expect(onPasteFiles).not.toHaveBeenCalled();
    });

    it("ignores a pasted file with nothing in it", () => {
        // given
        const onPasteFiles = vi.fn();
        const empty = makeFile("empty.png", "image/png", 0);
        renderWithProviders(<Harness onPasteFiles={onPasteFiles} />);

        // when
        fireEvent.paste(box(), { clipboardData: { files: [empty] } });

        // then
        expect(onPasteFiles).not.toHaveBeenCalled();
    });

    it("lets an ordinary text paste through untouched", async () => {
        // given
        const onPasteFiles = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Harness onPasteFiles={onPasteFiles} />);

        // when
        await user.click(box());
        await user.paste("the golden truth");

        // then
        expect(onPasteFiles).not.toHaveBeenCalled();
        expect(box()).toHaveValue("the golden truth");
    });

    it("focuses the box when its handle is asked to", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<FocusHarness />);

        // when
        await user.click(screen.getByRole("button", { name: "focus the box" }));

        // then
        expect(document.activeElement).toBe(box());
    });
});
