import { useMemo } from "react";
import { registerDeviceToken, unregisterDeviceToken } from "../../api/endpoints/auth";

export interface DeviceTokenCalls {
    registerToken: (token: string, platform: string) => Promise<void>;
    unregisterToken: (token: string) => Promise<void>;
}

export function useDeviceTokens(): DeviceTokenCalls {
    return useMemo(() => ({ registerToken: registerDeviceToken, unregisterToken: unregisterDeviceToken }), []);
}
