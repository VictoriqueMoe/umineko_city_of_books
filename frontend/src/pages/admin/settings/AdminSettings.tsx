import { Button } from "../../../components/Button/Button";
import { useAdminSettingsForm } from "../../../hooks/useAdminSettingsForm";
import { AppearanceSection } from "./sections/AppearanceSection";
import { CacheSection } from "./sections/CacheSection";
import { ChatbotSection } from "./sections/ChatbotSection";
import { DroneBLSection } from "./sections/DroneBLSection";
import { EmailSection } from "./sections/EmailSection";
import { FeatureTogglesSection } from "./sections/FeatureTogglesSection";
import { FileSizeLimitsSection } from "./sections/FileSizeLimitsSection";
import { GeneralSection } from "./sections/GeneralSection";
import { LimitsSection } from "./sections/LimitsSection";
import { LinkPreviewsSection } from "./sections/LinkPreviewsSection";
import { LoggingSection } from "./sections/LoggingSection";
import { MobileAppSection } from "./sections/MobileAppSection";
import { PrivateModeSection } from "./sections/PrivateModeSection";
import { StreamingSection } from "./sections/StreamingSection";
import { TurnstileSection } from "./sections/TurnstileSection";
import { WebPushSection } from "./sections/WebPushSection";
import styles from "./AdminSettings.module.css";

export function AdminSettings() {
    const form = useAdminSettingsForm();

    if (form.loading) {
        return <div className={styles.loading}>Loading settings...</div>;
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Site Settings</h1>

            <FeatureTogglesSection form={form} />
            <PrivateModeSection form={form} />
            <DroneBLSection form={form} />
            <TurnstileSection form={form} />
            <StreamingSection form={form} />
            <MobileAppSection form={form} />
            <WebPushSection form={form} />
            <ChatbotSection form={form} />
            <GeneralSection form={form} />
            <CacheSection form={form} />
            <LimitsSection form={form} />
            <FileSizeLimitsSection form={form} />
            <EmailSection form={form} />
            <LoggingSection form={form} />
            <AppearanceSection form={form} />
            <LinkPreviewsSection form={form} />

            <div className={styles.saveRow}>
                <Button variant="primary" onClick={form.save} disabled={form.saving}>
                    {form.saving ? "Saving..." : "Save Settings"}
                </Button>
                {form.error && <span className={styles.saveError}>{form.error}</span>}
                {form.success && <span className={styles.success}>{form.success}</span>}
            </div>
        </div>
    );
}
