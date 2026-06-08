import type { CapacitorConfig } from "@capacitor/cli";

declare const process: { env: Record<string, string | undefined> };

const devServerUrl = process.env.CAP_SERVER_URL;

const config: CapacitorConfig = {
    appId: "moe.auaurora.cityofbooks",
    appName: "Umineko City of Books",
    webDir: "dist-app",
    backgroundColor: "#0A0612",
    android: {
        backgroundColor: "#0A0612",
    },
    plugins: {
        PushNotifications: {
            presentationOptions: ["badge", "sound", "alert"],
        },
        SplashScreen: {
            backgroundColor: "#0A0612",
            showSpinner: false,
        },
    },
    ...(devServerUrl ? { server: { url: devServerUrl, cleartext: true } } : {}),
};

export default config;
