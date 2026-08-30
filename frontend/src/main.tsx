import "./api/globalErrorHandlers";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import App from "./App";
import { queryClient } from "./api/queryClient";
import { reportClientError } from "./api/telemetry";
import { RootErrorBoundary } from "./components/RootErrorBoundary/RootErrorBoundary";
import { SiteInfoProvider } from "./context/SiteInfoContext";
import { ThemeProvider } from "./context/ThemeContext";
import { AuthProvider } from "./context/AuthContext";
import { NotificationProvider } from "./context/NotificationContext";
import { GifFavouritesProvider } from "./context/GifFavouritesContext";
import { MentionResolverProvider } from "./context/MentionResolverContext";
import { loadAuthToken } from "./api/authToken";
import { getOtaManifest, otaBundleUrl } from "./api/ota";
import { initAppUpdates } from "./platform/appUpdate";
import { setPlatformErrorReporter } from "./platform/errorReporter";
import { nonFatal } from "./utils/nonFatal";
import "./styles/variables.css";
import "./styles/global.css";
import { watchInstallPrompt } from "./platform/installPrompt";

setPlatformErrorReporter(reportClientError);
watchInstallPrompt();

function startAppUpdates(): void {
    initAppUpdates({ getManifest: getOtaManifest, bundleUrl: otaBundleUrl });
}

function reportBoundaryError(error: unknown, componentStack: string | null): void {
    reportClientError(error, { source: "error-boundary", componentStack });
}

function renderApp() {
    createRoot(document.getElementById("root")!, {
        onUncaughtError: (error, errorInfo) => {
            reportClientError(error, { source: "uncaught", componentStack: errorInfo.componentStack });
        },
        onCaughtError: (error, errorInfo) => {
            reportClientError(error, { source: "caught", componentStack: errorInfo.componentStack });
        },
        onRecoverableError: (error, errorInfo) => {
            reportClientError(error, { source: "recoverable", componentStack: errorInfo.componentStack });
        },
    }).render(
        <StrictMode>
            <RootErrorBoundary onError={reportBoundaryError}>
                <QueryClientProvider client={queryClient}>
                    <SiteInfoProvider>
                        <AuthProvider>
                            <ThemeProvider>
                                <NotificationProvider>
                                    <GifFavouritesProvider>
                                        <MentionResolverProvider>
                                            <App />
                                        </MentionResolverProvider>
                                    </GifFavouritesProvider>
                                </NotificationProvider>
                            </ThemeProvider>
                        </AuthProvider>
                    </SiteInfoProvider>
                </QueryClientProvider>
            </RootErrorBoundary>
        </StrictMode>,
    );
}

loadAuthToken()
    .catch(nonFatal)
    .then(renderApp)
    .then(startAppUpdates)
    .catch(error => {
        reportClientError(error, { source: "boot" });
    });
