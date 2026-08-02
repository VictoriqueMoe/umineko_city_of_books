import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RichTextEditor } from "./RichTextEditor";

interface ChainCall {
    method: string;
    args: unknown[];
}

interface FakeEditor {
    calls: ChainCall[];
    chain: () => Record<string, (...args: unknown[]) => unknown>;
    isActive: (...args: unknown[]) => boolean;
    getAttributes: (name: string) => Record<string, unknown>;
    getHTML: () => string;
}

interface ConfiguredExtension {
    name: string;
    options: unknown;
}

interface EditorOptions {
    immediatelyRender: boolean;
    extensions: ConfiguredExtension[];
    content: string;
    onUpdate: (props: { editor: FakeEditor }) => void;
    onTransaction: () => void;
}

const { tiptap } = vi.hoisted(() => ({
    tiptap: {
        editor: null as FakeEditor | null,
        options: null as EditorOptions | null,
    },
}));

vi.mock("@tiptap/react", () => ({
    useEditor: (options: EditorOptions) => {
        tiptap.options = options;
        return tiptap.editor;
    },
    EditorContent: () => <div data-testid="editor-surface" />,
}));

vi.mock("@tiptap/starter-kit", () => ({
    default: { configure: (options: unknown) => ({ name: "starter-kit", options }) },
}));

vi.mock("@tiptap/extension-placeholder", () => ({
    default: { configure: (options: unknown) => ({ name: "placeholder", options }) },
}));

vi.mock("@tiptap/extension-text-align", () => ({
    default: { configure: (options: unknown) => ({ name: "text-align", options }) },
}));

vi.mock("@tiptap/extension-color", () => ({ default: { name: "colour", options: null } }));

vi.mock("@tiptap/extension-text-style", () => ({ TextStyle: { name: "text-style", options: null } }));

function makeEditor(): FakeEditor {
    const calls: ChainCall[] = [];
    const chain: Record<string, (...args: unknown[]) => unknown> = new Proxy(
        {},
        {
            get:
                (_target, property: string) =>
                (...args: unknown[]) => {
                    calls.push({ method: property, args });
                    return chain;
                },
        },
    );

    return {
        calls,
        chain: () => chain,
        isActive: () => false,
        getAttributes: () => ({}),
        getHTML: () => "<p>the golden truth</p>",
    };
}

function options(): EditorOptions {
    if (!tiptap.options) {
        throw new Error("expected the editor to have been configured");
    }

    return tiptap.options;
}

function extension(name: string): ConfiguredExtension {
    const found = options().extensions.find(e => e.name === name);
    if (!found) {
        throw new Error(`expected the ${name} extension to be installed`);
    }

    return found;
}

function editor(): FakeEditor {
    if (!tiptap.editor) {
        throw new Error("expected a fake editor to be in place");
    }

    return tiptap.editor;
}

function chainSince(from: number): ChainCall[] {
    return editor().calls.slice(from);
}

function methodsSince(from: number): string[] {
    return chainSince(from).map(call => call.method);
}

function press(label: string) {
    fireEvent.mouseDown(screen.getByRole("button", { name: label }));
}

function colourButtons(): HTMLElement[] {
    return screen.getAllByRole("button").filter(button => button.textContent === "");
}

function noop() {}

describe("RichTextEditor", () => {
    beforeEach(() => {
        tiptap.editor = makeEditor();
        tiptap.options = null;
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("shows nothing at all until the editor has been built", () => {
        // given
        tiptap.editor = null;

        // when
        const { container } = render(<RichTextEditor content="<p>hi</p>" onChange={noop} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("hands the writing surface to the editor once it exists", () => {
        // given
        const content = "<p>hi</p>";

        // when
        render(<RichTextEditor content={content} onChange={noop} />);

        // then
        expect(screen.getByTestId("editor-surface")).toBeInTheDocument();
    });

    it("seeds the editor with the content it was given and waits for the browser", () => {
        // given
        const content = "<p>without love it cannot be seen</p>";

        // when
        render(<RichTextEditor content={content} onChange={noop} />);

        // then
        expect(options().content).toBe(content);
        expect(options().immediatelyRender).toBe(false);
    });

    it("prompts the writer with its own words when no placeholder was given", () => {
        // given
        const content = "";

        // when
        render(<RichTextEditor content={content} onChange={noop} />);

        // then
        expect(extension("placeholder").options).toEqual({ placeholder: "Write your story..." });
    });

    it("prompts the writer with the placeholder it was given", () => {
        // given
        const placeholder = "Set out your theory";

        // when
        render(<RichTextEditor content="" onChange={noop} placeholder={placeholder} />);

        // then
        expect(extension("placeholder").options).toEqual({ placeholder });
    });

    it("offers only the two smaller heading levels", () => {
        // given
        const content = "";

        // when
        render(<RichTextEditor content={content} onChange={noop} />);

        // then
        expect(extension("starter-kit").options).toMatchObject({ heading: { levels: [2, 3] } });
    });

    it("makes links open in a new tab without passing on any credit", () => {
        // given
        const content = "";

        // when
        render(<RichTextEditor content={content} onChange={noop} />);

        // then
        expect(extension("starter-kit").options).toMatchObject({
            link: {
                openOnClick: false,
                HTMLAttributes: { rel: "noopener noreferrer nofollow", target: "_blank" },
            },
        });
    });

    it("allows alignment only on headings and paragraphs", () => {
        // given
        const content = "";

        // when
        render(<RichTextEditor content={content} onChange={noop} />);

        // then
        expect(extension("text-align").options).toEqual({ types: ["heading", "paragraph"] });
    });

    it("reports the html back to the caller whenever the editor changes", () => {
        // given
        const onChange = vi.fn();
        render(<RichTextEditor content="" onChange={onChange} />);

        // when
        act(() => options().onUpdate({ editor: editor() }));

        // then
        expect(onChange).toHaveBeenCalledWith("<p>the golden truth</p>");
    });

    it("toggles bold on the current selection", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("B");

        // then
        expect(methodsSince(before)).toEqual(["focus", "toggleBold", "run"]);
    });

    it("toggles italics on the current selection", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("I");

        // then
        expect(methodsSince(before)).toEqual(["focus", "toggleItalic", "run"]);
    });

    it("turns the current block into a second level heading", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("H2");

        // then
        expect(chainSince(before)).toEqual([
            { method: "focus", args: [] },
            { method: "toggleHeading", args: [{ level: 2 }] },
            { method: "run", args: [] },
        ]);
    });

    it("turns the current block into a third level heading", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("H3");

        // then
        expect(chainSince(before)).toContainEqual({ method: "toggleHeading", args: [{ level: 3 }] });
    });

    it("centres the current block", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("Centre");

        // then
        expect(chainSince(before)).toContainEqual({ method: "setTextAlign", args: ["center"] });
    });

    it("drops in a horizontal rule", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("HR");

        // then
        expect(methodsSince(before)).toEqual(["focus", "setHorizontalRule", "run"]);
    });

    it("paints the selection in the colour that was pressed", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        fireEvent.mouseDown(colourButtons()[0]);

        // then
        expect(chainSince(before)).toContainEqual({ method: "setColor", args: ["#e53935"] });
    });

    it("offers a colour for each of the four truths", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        for (const button of colourButtons()) {
            fireEvent.mouseDown(button);
        }

        // then
        expect(chainSince(before).filter(call => call.method === "setColor").length).toBe(4);
        expect(chainSince(before)).toContainEqual({ method: "setColor", args: ["#ab47bc"] });
    });

    it("strips the colour back off again", () => {
        // given
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("✕");

        // then
        expect(methodsSince(before)).toEqual(["focus", "unsetColor", "run"]);
    });

    it("marks the toolbar buttons that match the selection", () => {
        // given
        const fake = editor();
        fake.isActive = (...args: unknown[]) => args[0] === "bold";

        // when
        render(<RichTextEditor content="" onChange={noop} />);

        // then
        expect(screen.getByRole("button", { name: "B" }).className).toContain("toolbarBtnActive");
        expect(screen.getByRole("button", { name: "I" }).className).not.toContain("toolbarBtnActive");
    });

    it("re-reads the selection whenever the editor reports a transaction", () => {
        // given
        const fake = editor();
        render(<RichTextEditor content="" onChange={noop} />);
        expect(screen.getByRole("button", { name: "Quote" }).className).not.toContain("toolbarBtnActive");

        // when
        fake.isActive = (...args: unknown[]) => args[0] === "blockquote";
        act(() => options().onTransaction());

        // then
        expect(screen.getByRole("button", { name: "Quote" }).className).toContain("toolbarBtnActive");
    });

    it("offers the link the writer already has as the starting point", () => {
        // given
        const prompt = vi.spyOn(window, "prompt").mockReturnValue(null);
        editor().getAttributes = () => ({ href: "https://witch.example" });
        render(<RichTextEditor content="" onChange={noop} />);

        // when
        press("Link");

        // then
        expect(prompt).toHaveBeenCalledWith("URL", "https://witch.example");
    });

    it("starts a new link from a bare https when there is nothing to edit", () => {
        // given
        const prompt = vi.spyOn(window, "prompt").mockReturnValue(null);
        render(<RichTextEditor content="" onChange={noop} />);

        // when
        press("Link");

        // then
        expect(prompt).toHaveBeenCalledWith("URL", "https://");
    });

    it("leaves the document untouched when the link prompt is dismissed", () => {
        // given
        vi.spyOn(window, "prompt").mockReturnValue(null);
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("Link");

        // then
        expect(chainSince(before)).toEqual([]);
    });

    it("applies the link across the whole mark once a url is given", () => {
        // given
        vi.spyOn(window, "prompt").mockReturnValue("https://witch.example/tea");
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("Link");

        // then
        expect(chainSince(before)).toEqual([
            { method: "focus", args: [] },
            { method: "extendMarkRange", args: ["link"] },
            { method: "setLink", args: [{ href: "https://witch.example/tea" }] },
            { method: "run", args: [] },
        ]);
    });

    it("removes the link when the prompt is emptied out", () => {
        // given
        vi.spyOn(window, "prompt").mockReturnValue("");
        render(<RichTextEditor content="" onChange={noop} />);
        const before = editor().calls.length;

        // when
        press("Link");

        // then
        expect(methodsSince(before)).toEqual(["focus", "extendMarkRange", "unsetLink", "run"]);
    });
});
