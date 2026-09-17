import { useEffect, useState } from "react";
import { Capacitor } from "@capacitor/core";
import { clearInstallPrompt, getInstallPrompt, subscribeInstallPrompt } from "../../platform/installPrompt";
import { Banner, BannerButton, BannerDismiss } from "../Banner/Banner";

const DISMISSED_KEY = "dismissed_install_prompt";

type PromptMode = "none" | "event" | "ios";

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
    const [deferred, setDeferred] = useState(getInstallPrompt);
    const [gone, setGone] = useState(false);

    useEffect(() => {
        if (mode === "none") {
            return;
        }

        function handleInstalled() {
            setGone(true);
        }

        const unsubscribe = subscribeInstallPrompt(setDeferred);
        window.addEventListener("appinstalled", handleInstalled);

        return () => {
            unsubscribe();
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
        clearInstallPrompt();
        setDeferred(null);
    }

    function handleDismiss() {
        rememberDismissal();
        setGone(true);
    }

    const actions = (
        <>
            {mode === "event" ? <BannerButton onClick={handleInstall}>Install</BannerButton> : null}
            <BannerDismiss onDismiss={handleDismiss} />
        </>
    );

    return (
        <Banner colour="teal" role="region" label="Install City of Books" actions={actions}>
            {mode === "ios"
                ? "Add City of Books to your home screen: tap Share, then Add to Home Screen."
                : "Add City of Books to your home screen and it opens like an app, without the browser bars."}
        </Banner>
    );
}
