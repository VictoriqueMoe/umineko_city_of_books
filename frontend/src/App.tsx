import {BrowserRouter, Route, Routes} from "react-router";
import {useTheme} from "./hooks/useTheme";
import {Header} from "./components/layout/Header";
import {Footer} from "./components/layout/Footer";
import {Butterflies} from "./components/layout/Butterflies";
import {FeedPage} from "./pages/FeedPage";
import {TheoryPage} from "./pages/TheoryPage";
import {CreateTheoryPage} from "./pages/CreateTheoryPage";
import {LoginPage} from "./pages/LoginPage";
import {QuoteBrowserPage} from "./pages/QuoteBrowserPage";
import {MyTheoriesPage} from "./pages/MyTheoriesPage";
import {EditTheoryPage} from "./pages/EditTheoryPage";
import {ProfilePage} from "./pages/ProfilePage";
import {SettingsPage} from "./pages/SettingsPage";

function AppLayout() {
    const { particlesEnabled } = useTheme();

    return (
        <>
            {particlesEnabled && <Butterflies />}
            <Header />
            <main className="main-content">
                <Routes>
                    <Route path="/" element={<FeedPage />} />
                    <Route path="/theory/new" element={<CreateTheoryPage />} />
                    <Route path="/theory/:id" element={<TheoryPage />} />
                    <Route path="/theory/:id/edit" element={<EditTheoryPage />} />
                    <Route path="/my-theories" element={<MyTheoriesPage />} />
                    <Route path="/quotes" element={<QuoteBrowserPage />} />
                    <Route path="/user/:username" element={<ProfilePage />} />
                    <Route path="/settings" element={<SettingsPage />} />
                    <Route path="/login" element={<LoginPage />} />
                </Routes>
            </main>
            <Footer />
        </>
    );
}

export default function App() {
    return (
        <BrowserRouter>
            <AppLayout />
        </BrowserRouter>
    );
}
