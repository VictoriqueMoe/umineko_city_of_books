import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import App from "./App";
import { queryClient } from "./api/queryClient";
import { SiteInfoProvider } from "./context/SiteInfoContext";
import { ThemeProvider } from "./context/ThemeContext";
import { AuthProvider } from "./context/AuthContext";
import { NotificationProvider } from "./context/NotificationContext";
import { GifFavouritesProvider } from "./context/GifFavouritesContext";
import "./styles/variables.css";
import "./styles/global.css";

createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <QueryClientProvider client={queryClient}>
            <SiteInfoProvider>
                <AuthProvider>
                    <ThemeProvider>
                        <NotificationProvider>
                            <GifFavouritesProvider>
                                <App />
                            </GifFavouritesProvider>
                        </NotificationProvider>
                    </ThemeProvider>
                </AuthProvider>
            </SiteInfoProvider>
        </QueryClientProvider>
    </StrictMode>,
);
