import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { SiteInfoProvider } from "./context/SiteInfoContext";
import { ThemeProvider } from "./context/ThemeContext";
import { AuthProvider } from "./context/AuthContext";
import { NotificationProvider } from "./context/NotificationContext";
import "./styles/variables.css";
import "./styles/global.css";

createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <SiteInfoProvider>
            <AuthProvider>
                <ThemeProvider>
                    <NotificationProvider>
                        <App />
                    </NotificationProvider>
                </ThemeProvider>
            </AuthProvider>
        </SiteInfoProvider>
    </StrictMode>,
);
