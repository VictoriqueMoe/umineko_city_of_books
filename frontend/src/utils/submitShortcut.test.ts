import { describe, expect, it, vi } from "vitest";
import { isSubmitShortcut, submitFormLikeEnter } from "./submitShortcut";

describe("isSubmitShortcut", () => {
    const cases = [
        {
            name: "ctrl and enter",
            event: { key: "Enter", ctrlKey: true, metaKey: false, isComposing: false },
            want: true,
        },
        {
            name: "cmd and enter",
            event: { key: "Enter", ctrlKey: false, metaKey: true, isComposing: false },
            want: true,
        },
        {
            name: "plain enter",
            event: { key: "Enter", ctrlKey: false, metaKey: false, isComposing: false },
            want: false,
        },
        {
            name: "ctrl and another key",
            event: { key: "a", ctrlKey: true, metaKey: false, isComposing: false },
            want: false,
        },
        {
            name: "ctrl and enter while an input method is composing",
            event: { key: "Enter", ctrlKey: true, metaKey: false, isComposing: true },
            want: false,
        },
    ];

    for (const { name, event, want } of cases) {
        it(`answers ${want} for ${name}`, () => {
            // when
            const result = isSubmitShortcut(event);

            // then
            expect(result).toBe(want);
        });
    }
});

describe("submitFormLikeEnter", () => {
    function buildForm(buttonMarkup: string): { form: HTMLFormElement; onSubmit: ReturnType<typeof vi.fn> } {
        const form = document.createElement("form");
        form.innerHTML = `<textarea></textarea>${buttonMarkup}`;
        const onSubmit = vi.fn((e: Event) => e.preventDefault());
        form.addEventListener("submit", onSubmit);
        document.body.append(form);
        return { form, onSubmit };
    }

    it("submits through the form's default button", () => {
        // given
        const { form, onSubmit } = buildForm(`<button type="button">Preview</button><button>Post</button>`);

        // when
        const submitted = submitFormLikeEnter(form);

        // then
        expect(submitted).toBe(true);
        expect(onSubmit).toHaveBeenCalledTimes(1);
        expect((onSubmit.mock.calls[0][0] as SubmitEvent).submitter).toHaveTextContent("Post");
    });

    it("does nothing while the default button is disabled", () => {
        // given
        const { form, onSubmit } = buildForm(`<button type="submit" disabled>Post</button>`);

        // when
        const submitted = submitFormLikeEnter(form);

        // then
        expect(submitted).toBe(false);
        expect(onSubmit).not.toHaveBeenCalled();
    });

    it("submits a form that has no submit button", () => {
        // given
        const { form, onSubmit } = buildForm("");

        // when
        const submitted = submitFormLikeEnter(form);

        // then
        expect(submitted).toBe(true);
        expect(onSubmit).toHaveBeenCalledTimes(1);
    });
});
