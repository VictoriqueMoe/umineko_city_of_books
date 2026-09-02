import type { useAdminSettingsForm } from "../../../../hooks/useAdminSettingsForm";

export type AdminSettingsForm = ReturnType<typeof useAdminSettingsForm>;

export interface SectionProps {
    form: AdminSettingsForm;
}
