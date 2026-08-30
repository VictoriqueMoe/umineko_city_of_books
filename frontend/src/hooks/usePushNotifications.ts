import { useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "./useAuth";
import { useSiteInfo } from "./useSiteInfo";
import { useDeviceTokens } from "./mutations/device";
import { ensureNotificationPermission } from "../platform/desktopNotifications";
import { initPush } from "../platform/pushNative";
import { initWebPushRouting, resumeWebPush, webPushConfigured } from "../platform/webPush";

export function usePushNotifications(): void {
    const { user } = useAuth();
    const siteInfo = useSiteInfo();
    const navigate = useNavigate();
    const deviceTokens = useDeviceTokens();

    const config = siteInfo.web_push;
    const configRef = useRef(config);
    const userId = user?.id;
    const webPushReady = webPushConfigured(config);

    useEffect(() => {
        configRef.current = config;
    }, [config]);

    useEffect(() => {
        if (!userId) {
            return;
        }

        ensureNotificationPermission().catch(() => {});
        initPush({ navigate, registerToken: deviceTokens.registerToken }).catch(() => {});

        if (webPushReady) {
            resumeWebPush(configRef.current, deviceTokens).catch(() => {});
        }

        return initWebPushRouting(navigate);
    }, [userId, webPushReady, navigate, deviceTokens]);
}
