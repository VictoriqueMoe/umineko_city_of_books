type Listener = () => void;

const listeners = new Set<Listener>();

let lost = false;

function announce(): void {
    for (const listener of listeners) {
        listener();
    }
}

export function isSessionLost(): boolean {
    return lost;
}

export function markSessionLost(): void {
    if (lost) {
        return;
    }

    lost = true;
    announce();
}

export function clearSessionLost(): void {
    if (!lost) {
        return;
    }

    lost = false;
    announce();
}

export function subscribeSessionLost(listener: Listener): () => void {
    listeners.add(listener);

    return () => {
        listeners.delete(listener);
    };
}
