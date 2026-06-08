import { Capacitor } from "@capacitor/core";
import { Preferences } from "@capacitor/preferences";

const TOKEN_KEY = "ut_session_token";

let cachedToken: string | null = null;

export function isNativeApp(): boolean {
    return Capacitor.isNativePlatform();
}

export function clientPlatform(): string {
    return Capacitor.getPlatform();
}

export function getAuthToken(): string | null {
    return cachedToken;
}

export function setAuthToken(token: string): void {
    cachedToken = token;
    Preferences.set({ key: TOKEN_KEY, value: token }).catch(() => {});
}

export function clearAuthToken(): void {
    cachedToken = null;
    Preferences.remove({ key: TOKEN_KEY }).catch(() => {});
}

export async function loadAuthToken(): Promise<void> {
    if (!Capacitor.isNativePlatform()) {
        return;
    }

    const { value } = await Preferences.get({ key: TOKEN_KEY });
    cachedToken = value;
}
