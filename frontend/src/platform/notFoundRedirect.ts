import { Capacitor } from "@capacitor/core";

const NOT_FOUND_PATH = "/not-found";

export function redirectToNotFound(pathname: string): void {
    if (Capacitor.isNativePlatform() || pathname === NOT_FOUND_PATH) {
        return;
    }

    window.location.replace(NOT_FOUND_PATH);
}
