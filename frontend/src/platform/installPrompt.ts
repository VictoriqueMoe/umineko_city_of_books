export interface BeforeInstallPromptEvent extends Event {
    prompt: () => Promise<void>;
}

type Listener = (event: BeforeInstallPromptEvent | null) => void;

let captured: BeforeInstallPromptEvent | null = null;
let watching = false;
const listeners = new Set<Listener>();

function notify(): void {
    for (const listener of listeners) {
        listener(captured);
    }
}

export function watchInstallPrompt(): void {
    if (typeof window === "undefined" || watching) {
        return;
    }

    watching = true;

    window.addEventListener("beforeinstallprompt", event => {
        event.preventDefault();
        captured = event as BeforeInstallPromptEvent;
        notify();
    });

    window.addEventListener("appinstalled", () => {
        captured = null;
        notify();
    });
}

export function getInstallPrompt(): BeforeInstallPromptEvent | null {
    return captured;
}

export function clearInstallPrompt(): void {
    captured = null;
}

export function subscribeInstallPrompt(listener: Listener): () => void {
    listeners.add(listener);

    return () => {
        listeners.delete(listener);
    };
}
