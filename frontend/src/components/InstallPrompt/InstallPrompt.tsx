import { useEffect, useState } from "react";
import { Capacitor } from "@capacitor/core";
import styles from "./InstallPrompt.module.css";

const DISMISSED_KEY = "dismissed_install_prompt";

type PromptMode = "none" | "event" | "ios";

interface BeforeInstallPromptEvent extends Event {
    prompt: () => Promise<void>;
}

function wasDismissed(): boolean {
    try {
        return window.localStorage.getItem(DISMISSED_KEY) === "1";
    } catch {
        return false;
    }
}

function rememberDismissal(): void {
    try {
        window.localStorage.setItem(DISMISSED_KEY, "1");
    } catch {
        return;
    }
}

function alreadyInstalled(): boolean {
    if (window.matchMedia("(display-mode: standalone)").matches) {
        return true;
    }

    return (window.navigator as Navigator & { standalone?: boolean })?.standalone ?? false;
}

function isIosSafari(): boolean {
    const ua = window.navigator.userAgent;
    const ios = /iPad|iPhone|iPod/.test(ua) || (/Macintosh/.test(ua) && window.navigator.maxTouchPoints > 1);

    if (!ios) {
        return false;
    }

    return !/CriOS|FxiOS|EdgiOS|OPiOS/.test(ua);
}

function initialMode(): PromptMode {
    if (Capacitor.isNativePlatform() || alreadyInstalled() || wasDismissed()) {
        return "none";
    }

    return isIosSafari() ? "ios" : "event";
}

export function InstallPrompt() {
    const [mode] = useState(initialMode);
    const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(null);
    const [gone, setGone] = useState(false);

    useEffect(() => {
        if (mode === "none") {
            return;
        }

        function handleBeforeInstallPrompt(event: Event) {
            event.preventDefault();
            setDeferred(event as BeforeInstallPromptEvent);
        }

        function handleInstalled() {
            setGone(true);
        }

        window.addEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
        window.addEventListener("appinstalled", handleInstalled);

        return () => {
            window.removeEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
            window.removeEventListener("appinstalled", handleInstalled);
        };
    }, [mode]);

    if (gone || mode === "none" || (mode === "event" && !deferred)) {
        return null;
    }

    function handleInstall() {
        if (!deferred) {
            return;
        }

        deferred.prompt().catch(() => {});
        setDeferred(null);
    }

    function handleDismiss() {
        rememberDismissal();
        setGone(true);
    }

    return (
        <div className={styles.banner} role="region" aria-label="Install City of Books">
            <span className={styles.text}>
                {mode === "ios"
                    ? "Add City of Books to your home screen: tap Share, then Add to Home Screen."
                    : "Add City of Books to your home screen and it opens like an app, without the browser bars."}
            </span>
            {mode === "event" ? (
                <button type="button" onClick={handleInstall} className={styles.button}>
                    Install
                </button>
            ) : null}
            <button type="button" onClick={handleDismiss} className={styles.dismiss} aria-label="Dismiss">
                &times;
            </button>
        </div>
    );
}
