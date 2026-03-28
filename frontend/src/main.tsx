import {StrictMode} from "react";
import {createRoot} from "react-dom/client";
import App from "./App";
import {ThemeProvider} from "./context/ThemeContext";
import {AuthProvider} from "./context/AuthContext";
import "./styles/variables.css";
import "./styles/global.css";

createRoot(document.getElementById("root")!).render(
    <StrictMode>
        <ThemeProvider>
            <AuthProvider>
                <App />
            </AuthProvider>
        </ThemeProvider>
    </StrictMode>,
);
