import { matchPath } from "react-router";

const AUTH_PATHS = ["/login", "/forgot-password", "/reset-password", "/set-email", "/verify-email"];

let lastLocation: string | null = null;

export function isAuthPath(pathname: string): boolean {
    let decoded: string;
    try {
        decoded = decodeURIComponent(pathname);
    } catch {
        return true;
    }

    for (const authPath of AUTH_PATHS) {
        if (matchPath(authPath, decoded)) {
            return true;
        }
    }

    return false;
}

export function recordLocation(pathname: string, search: string, hash: string): void {
    if (isAuthPath(pathname)) {
        return;
    }

    lastLocation = pathname + search + hash;
}

export function getLastLocation(): string | null {
    return lastLocation;
}
