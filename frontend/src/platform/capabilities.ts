import { Capacitor } from "@capacitor/core";

export function isNativeApp(): boolean {
    return Capacitor.isNativePlatform();
}

export function clientPlatform(): string {
    return Capacitor.getPlatform();
}
