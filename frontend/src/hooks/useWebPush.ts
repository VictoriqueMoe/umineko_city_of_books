import { useCallback, useState } from "react";
import { useSiteInfo } from "./useSiteInfo";
import { useDeviceTokens } from "./mutations/device";
import {
    disableWebPush,
    enableWebPush,
    webPushConfigured,
    webPushEnabled,
    webPushSupported,
} from "../platform/webPush";
import { errorMessage } from "../utils/errorMessage";

const UNKNOWN_REASON = "the browser gave no reason";

export interface WebPushControls {
    available: boolean;
    enabled: boolean;
    busy: boolean;
    error: string;
    setEnabled: (next: boolean) => void;
}

export function useWebPush(): WebPushControls {
    const siteInfo = useSiteInfo();
    const deviceTokens = useDeviceTokens();
    const [enabled, setEnabled] = useState(() => webPushEnabled());
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");

    const config = siteInfo.web_push;
    const available = siteInfo.push_enabled && webPushConfigured(config) && webPushSupported();

    const change = useCallback(
        (next: boolean) => {
            setBusy(true);
            setError("");

            const action = next ? enableWebPush(config, deviceTokens) : disableWebPush(config, deviceTokens);

            action
                .then(() => setEnabled(next))
                .catch((thrown: unknown) => {
                    const reason = errorMessage(thrown, UNKNOWN_REASON);

                    setError(
                        next
                            ? `Could not turn notifications on: ${reason}`
                            : `Could not turn notifications off: ${reason}`,
                    );
                })
                .finally(() => setBusy(false));
        },
        [config, deviceTokens],
    );

    return { available, enabled, busy, error, setEnabled: change };
}
