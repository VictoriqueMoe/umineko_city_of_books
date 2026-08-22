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
        CapacitorUpdater: {
            autoUpdate: false,
            publicKey:
                "-----BEGIN RSA PUBLIC KEY-----\nMIIBCgKCAQEAzZsbVY/9lAcl6OV6gup4hoVyXuzgKIBT+HVeNhCnodVrAxfArdhx\n7lV1A+lzJnOyOdyGkxd5ELKBSjXKzAU5DzHkPU9E4G1qFTHd5DpRP/ftZHifaflB\nvB0dZi62RxIoVoBvg1SVskE7UAnE04vsfS4J2SJ24f080wZnWPOC/xV7U/lh9axv\nzGOVHJO0U0q1uM793ga1t+wOWU1UkrtlqAylN+xjlS5+fU1bBGxj18aWtyB1M7LP\nHPHHOMvuMeYHr/kPQbmEapVALWpLzjaH7Zjo1gfohEFZiK1IqV6g9bO6k8wVEUwz\nIm87DbGpAyFDNaf+Qn4/1lZN/wbIpu4UaQIDAQAB\n-----END RSA PUBLIC KEY-----\n",
        },
    },
    ...(devServerUrl ? { server: { url: devServerUrl, cleartext: true } } : {}),
};

export default config;
