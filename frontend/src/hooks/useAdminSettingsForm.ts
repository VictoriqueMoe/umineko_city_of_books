import { useId, useRef, useState } from "react";
import type { ChangeEvent } from "react";
import { useAdminPermissions, useAdminSettings, useChatbotModels } from "./queries/admin";
import {
    useSendTestEmail,
    useTestChatbotModel,
    useUpdateAdminSettings,
    useUploadOGDefaultImage,
} from "./mutations/admin";
import { usePageTitle } from "./usePageTitle";
import { useSiteInfo } from "./useSiteInfo";
import { parseIgnoredClasses, toggleIgnoredClass } from "../domain/dronebl";
import { customFeeds, isKnownFeedEnabled, replaceCustomFeeds, toggleKnownFeed } from "../domain/crawlerFeeds";
import type { CrawlerFeed, KnownCrawlerFeed } from "../domain/crawlerFeeds";
import {
    boolValue,
    bytesToMB,
    chatbotOptInRoles,
    isEnabled,
    mbToBytes,
    mpToPixels,
    pixelsToMP,
    validateSiteSettings,
} from "../domain/siteSettings";
import { errorMessage } from "../utils/errorMessage";
import type { SiteSettings } from "../types/api";

export function useAdminSettingsForm() {
    usePageTitle("Admin - Settings");
    const { site_name } = useSiteInfo();
    const baseID = useId();
    const { settings: loadedSettings, loading } = useAdminSettings();
    const { models, modelsError, loading: modelsLoading, refresh: refreshModels } = useChatbotModels();
    const updateSettingsMutation = useUpdateAdminSettings();
    const sendTestEmailMutation = useSendTestEmail();
    const testModelMutation = useTestChatbotModel();
    const uploadOGImageMutation = useUploadOGDefaultImage();
    const ogImageInputRef = useRef<HTMLInputElement>(null);
    const [draft, setDraft] = useState<SiteSettings>({});
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");
    const [testMessage, setTestMessage] = useState("");
    const [testError, setTestError] = useState("");
    const [modelTestMessage, setModelTestMessage] = useState("");
    const [modelTestError, setModelTestError] = useState("");
    const [ogImageError, setOGImageError] = useState("");
    const [customDraft, setCustomDraft] = useState<CrawlerFeed[] | null>(null);

    const saving = updateSettingsMutation.isPending;
    const settings: SiteSettings = { ...(loadedSettings ?? {}), ...draft };
    const ignoredClasses = parseIgnoredClasses(settings.dronebl_ignored_classes ?? "");
    const custom = customDraft ?? customFeeds(settings.crawler_feeds ?? "");

    const chatbotKeySaved = (loadedSettings?.chatbot_api_key ?? "").trim() !== "";
    const chatbotLocked = !chatbotKeySaved || models.length === 0;

    const restrictChatbots = isEnabled(settings.chatbot_enabled) && isEnabled(settings.chatbot_require_permission);
    const { vanityRoles, loading: rolesLoading } = useAdminPermissions(restrictChatbots);
    const optInRoles = chatbotOptInRoles(vanityRoles);
    const optInRoleID = (settings.chatbot_opt_in_role ?? "").trim();
    const optInRoleListed = optInRoles.some(role => role.id === optInRoleID);

    function fieldID(name: string) {
        return `${baseID}-${name}`;
    }

    function updateField(key: string, value: string) {
        setDraft(prev => ({ ...prev, [key]: value }));
        setSuccess("");
    }

    function toggleField(key: string, enabled: boolean) {
        updateField(key, boolValue(enabled));
    }

    function getNumber(key: string): string {
        return settings[key] ?? "0";
    }

    function getMB(key: string): string {
        return bytesToMB(settings[key]);
    }

    function setMB(key: string, mb: string) {
        updateField(key, mbToBytes(mb));
    }

    function getMP(key: string): string {
        return pixelsToMP(settings[key]);
    }

    function setMP(key: string, mp: string) {
        updateField(key, mpToPixels(mp));
    }

    function toggleIgnoredDroneBLClass(id: number, ignored: boolean) {
        updateField("dronebl_ignored_classes", toggleIgnoredClass(settings.dronebl_ignored_classes ?? "", id, ignored));
    }

    function knownFeedEnabled(known: KnownCrawlerFeed): boolean {
        return isKnownFeedEnabled(settings.crawler_feeds ?? "", known);
    }

    function toggleKnownCrawlerFeed(known: KnownCrawlerFeed, enabled: boolean) {
        updateField("crawler_feeds", toggleKnownFeed(settings.crawler_feeds ?? "", known, enabled));
    }

    function writeCustomFeeds(next: CrawlerFeed[]) {
        setCustomDraft(next);
        updateField("crawler_feeds", replaceCustomFeeds(settings.crawler_feeds ?? "", next));
    }

    function updateCustomFeed(index: number, patch: Partial<CrawlerFeed>) {
        writeCustomFeeds(custom.map((entry, i) => (i === index ? { ...entry, ...patch } : entry)));
    }

    function addCustomFeed() {
        writeCustomFeeds([...custom, { name: "", url: "" }]);
    }

    function removeCustomFeed(index: number) {
        writeCustomFeeds(custom.filter((_, i) => i !== index));
    }

    async function save() {
        const validationError = validateSiteSettings(settings);
        if (validationError) {
            setError(validationError);
            return;
        }

        setError("");
        setSuccess("");
        try {
            await updateSettingsMutation.mutateAsync(settings);
            setSuccess("Settings saved successfully");
        } catch (e) {
            setError(errorMessage(e, "Failed to save settings"));
        }
    }

    function chooseOGImage() {
        ogImageInputRef.current?.click();
    }

    function clearOGImage() {
        updateField("og_default_image", "");
    }

    async function handleOGImageSelected(e: ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        e.target.value = "";
        if (!file) {
            return;
        }

        setOGImageError("");
        try {
            const res = await uploadOGImageMutation.mutateAsync(file);
            updateField("og_default_image", res.image_url);
        } catch (err) {
            setOGImageError(errorMessage(err, "Failed to upload image"));
        }
    }

    async function handleTestModel() {
        setModelTestMessage("");
        setModelTestError("");
        try {
            const result = await testModelMutation.mutateAsync((settings.chatbot_model ?? "").trim());

            if (result.ok) {
                setModelTestMessage("The model answered. Save your changes to put it live.");
            } else {
                setModelTestError(result.error ?? "The model did not answer");
            }
        } catch (e) {
            setModelTestError(errorMessage(e, "Failed to reach the model"));
        }
    }

    async function handleSendTestEmail() {
        setTestMessage("");
        setTestError("");
        try {
            await sendTestEmailMutation.mutateAsync();
            setTestMessage("Test email sent. Check your inbox.");
        } catch (e) {
            setTestError(errorMessage(e, "Failed to send test email"));
        }
    }

    return {
        loading,
        saving,
        settings,
        error,
        success,
        defaultSiteName: site_name,
        fieldID,
        updateField,
        toggleField,
        getNumber,
        getMB,
        setMB,
        getMP,
        setMP,
        save,
        ogImageInputRef,

        dronebl: {
            ignoredClasses,
            toggleClass: toggleIgnoredDroneBLClass,
        },

        feeds: {
            custom,
            add: addCustomFeed,
            update: updateCustomFeed,
            remove: removeCustomFeed,
            isKnownEnabled: knownFeedEnabled,
            toggleKnown: toggleKnownCrawlerFeed,
        },

        ogImage: {
            uploading: uploadOGImageMutation.isPending,
            error: ogImageError,
            hasCustom: (settings.og_default_image ?? "") !== "",
            choose: chooseOGImage,
            clear: clearOGImage,
            onSelected: handleOGImageSelected,
        },

        emailTest: {
            send: handleSendTestEmail,
            sending: sendTestEmailMutation.isPending,
            message: testMessage,
            error: testError,
        },

        chatbot: {
            keySaved: chatbotKeySaved,
            locked: chatbotLocked,
            models,
            modelsError,
            modelsLoading,
            refreshModels,
            restrict: restrictChatbots,
            rolesLoading,
            optInRoles,
            optInRoleID,
            optInRoleListed,
            testModel: handleTestModel,
            testing: testModelMutation.isPending,
            testMessage: modelTestMessage,
            testError: modelTestError,
        },
    };
}
