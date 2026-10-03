export interface ShortcutKeyEvent {
    key: string;
    ctrlKey: boolean;
    metaKey: boolean;
    isComposing: boolean;
}

export function isSubmitShortcut(event: ShortcutKeyEvent): boolean {
    return event.key === "Enter" && (event.ctrlKey || event.metaKey) && !event.isComposing;
}

function isSubmitButton(element: Element): element is HTMLButtonElement | HTMLInputElement {
    if (element instanceof HTMLButtonElement) {
        return element.type === "submit";
    }

    return element instanceof HTMLInputElement && (element.type === "submit" || element.type === "image");
}

export function submitFormLikeEnter(form: HTMLFormElement): boolean {
    const defaultButton = Array.from(form.elements).find(isSubmitButton);

    if (!defaultButton) {
        form.requestSubmit();
        return true;
    }

    if (defaultButton.disabled) {
        return false;
    }

    form.requestSubmit(defaultButton);
    return true;
}
