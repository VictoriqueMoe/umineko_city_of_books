import { Capacitor } from "@capacitor/core";

const NOT_FOUND_PATH = "/not-found";

export function shouldRedirectToNotFound(pathname: string): boolean {
    return !Capacitor.isNativePlatform() && pathname !== NOT_FOUND_PATH;
}

export function redirectToNotFound(): void {
    window.location.replace(NOT_FOUND_PATH);
}
