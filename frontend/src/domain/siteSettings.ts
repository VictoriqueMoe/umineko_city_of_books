import type { SiteSettings } from "../types/api";

export interface ChatbotOptInRole {
    permissions: string[];
}

const BYTES_PER_MB = 1024 * 1024;
const PIXELS_PER_MP = 1_000_000;
const CHATBOT_PERMISSION = "use_chatbot";
const EMAIL_PROVIDER_CLOUDFLARE = "cloudflare";

export function isEnabled(value: string | undefined): boolean {
    return value === "true";
}

export function boolValue(enabled: boolean): string {
    return enabled ? "true" : "false";
}

export function bytesToMB(raw: string | undefined): string {
    const bytes = parseInt(raw ?? "0", 10);
    if (isNaN(bytes)) {
        return "0";
    }

    return String(Math.round(bytes / BYTES_PER_MB));
}

export function mbToBytes(mb: string): string {
    const mbNum = parseFloat(mb);
    if (isNaN(mbNum)) {
        return "0";
    }

    return String(Math.round(mbNum * BYTES_PER_MB));
}

export function pixelsToMP(raw: string | undefined): string {
    const pixels = parseInt(raw ?? "0", 10);
    if (isNaN(pixels)) {
        return "0";
    }

    return String(pixels / PIXELS_PER_MP);
}

export function mpToPixels(mp: string): string {
    const mpNum = parseFloat(mp);
    if (isNaN(mpNum)) {
        return "0";
    }

    return String(Math.round(mpNum * PIXELS_PER_MP));
}

export function chatbotOptInRoles<T extends ChatbotOptInRole>(roles: readonly T[]): T[] {
    return roles.filter(role => role.permissions.includes(CHATBOT_PERMISSION));
}

export function validateSiteSettings(settings: SiteSettings): string | null {
    const maxBody = parseInt(settings.max_body_size ?? "0", 10);
    const maxImage = parseInt(settings.max_image_size ?? "0", 10);
    const maxImagePixels = parseInt(settings.max_image_pixels ?? "0", 10);
    const maxVideo = parseInt(settings.max_video_size ?? "0", 10);
    const maxAudio = parseInt(settings.max_audio_size ?? "0", 10);
    const maxGeneral = parseInt(settings.max_general_size ?? "0", 10);
    const minPassword = parseInt(settings.min_password_length ?? "0", 10);
    const sessionDays = parseInt(settings.session_duration_days ?? "0", 10);
    const maxTheories = parseInt(settings.max_theories_per_day ?? "0", 10);
    const maxResponses = parseInt(settings.max_responses_per_day ?? "0", 10);
    const optInRoleID = (settings.chatbot_opt_in_role ?? "").trim();

    if (maxBody <= 0) {
        return "Max body size must be greater than 0";
    }
    if (maxImage <= 0) {
        return "Max image size must be greater than 0";
    }
    if (maxImagePixels <= 0) {
        return "Max image pixels must be greater than 0";
    }
    if (maxImage > maxBody) {
        return `Max image size (${Math.round(maxImage / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
    }
    if (maxVideo > maxBody) {
        return `Max video size (${Math.round(maxVideo / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
    }
    if (maxAudio > maxBody) {
        return `Max audio size (${Math.round(maxAudio / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
    }
    if (maxGeneral > maxBody) {
        return `Max general size (${Math.round(maxGeneral / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
    }
    if (minPassword < 1) {
        return "Minimum password length must be at least 1";
    }
    if (sessionDays < 1) {
        return "Session duration must be at least 1 day";
    }
    if (maxTheories < 0) {
        return "Max theories per day cannot be negative";
    }
    if (maxResponses < 0) {
        return "Max responses per day cannot be negative";
    }
    if (isEnabled(settings.voice_enabled)) {
        if (!settings.livekit_url || !settings.livekit_api_key || !settings.livekit_api_secret) {
            return "Voice chat requires LiveKit URL, API key and API secret";
        }
    }
    if (isEnabled(settings.chatbot_enabled)) {
        const maxOutputTokens = parseInt(settings.chatbot_max_output_tokens ?? "0", 10);
        const contextMessages = parseInt(settings.chatbot_context_messages ?? "0", 10);
        const maxReplyChain = parseInt(settings.chatbot_max_reply_chain ?? "0", 10);
        const replyCooldown = parseInt(settings.chatbot_reply_cooldown_seconds ?? "0", 10);
        const maxRepliesPerUser = parseInt(settings.chatbot_max_replies_per_user_per_day ?? "0", 10);
        const maxRepliesPerDay = parseInt(settings.chatbot_max_replies_per_day ?? "0", 10);

        if (maxOutputTokens < 1) {
            return "Chatbot max output tokens must be at least 1";
        }
        if (contextMessages < 0) {
            return "Chatbot context messages cannot be negative";
        }
        if (maxReplyChain < 0) {
            return "Chatbot max reply chain cannot be negative";
        }
        if (replyCooldown < 0) {
            return "Chatbot reply cooldown cannot be negative";
        }
        if (maxRepliesPerUser < 0) {
            return "Chatbot max replies per user per day cannot be negative";
        }
        if (maxRepliesPerDay < 0) {
            return "Chatbot max replies per day cannot be negative";
        }
    }
    if (isEnabled(settings.chatbot_require_permission) && optInRoleID === "") {
        if (!isEnabled(settings.chatbot_enabled)) {
            return "Restricting characters to a permission requires an opt-in role. Switch Enable Chatbot on to choose one.";
        }
        return "Restricting characters to a permission requires an opt-in role so members can opt in";
    }
    if (settings.email_provider === EMAIL_PROVIDER_CLOUDFLARE) {
        if (!settings.cloudflare_account_id || !settings.cloudflare_api_token || !settings.cloudflare_email_from) {
            return "Cloudflare email requires account ID, API token and from address";
        }
    }
    return null;
}
