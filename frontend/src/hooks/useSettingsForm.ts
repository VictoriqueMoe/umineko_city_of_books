import React, { useEffect, useState } from "react";
import { useAuth } from "./useAuth";
import { useProfile } from "./useProfile";
import { getCharacters, updateProfile, uploadAvatar, uploadBanner } from "../api/endpoints";
import type { UpdateProfilePayload, UserProfile } from "../types/api";

const GENDER_OPTIONS = ["Prefer not to say", "Male", "Female", "Custom"];

const PRONOUN_DEFAULTS: Record<string, { subject: string; possessive: string }> = {
    Male: { subject: "he", possessive: "his" },
    Female: { subject: "she", possessive: "her" },
    "Prefer not to say": { subject: "they", possessive: "their" },
    Custom: { subject: "they", possessive: "their" },
};

function initGender(profile: UserProfile) {
    const g = profile.gender || "";
    if (GENDER_OPTIONS.includes(g)) {
        return { gender: g, customGender: "" };
    }
    if (g) {
        return { gender: "Custom", customGender: g };
    }
    return { gender: "Prefer not to say", customGender: "" };
}

function initPronouns(profile: UserProfile) {
    const subject = profile.pronoun_subject || "they";
    const possessive = profile.pronoun_possessive || "their";
    const defaults = PRONOUN_DEFAULTS[GENDER_OPTIONS.includes(profile.gender) ? profile.gender : "Custom"];
    const isCustom =
        (!!profile.pronoun_subject && subject !== defaults.subject) ||
        (!!profile.pronoun_possessive && possessive !== defaults.possessive);
    return { subject, possessive, isCustom };
}

export function useSettingsForm() {
    const { user, setUser } = useAuth();
    const { profile, loading: profileLoading } = useProfile(user?.username ?? "");

    const [displayName, setDisplayName] = useState("");
    const [bio, setBio] = useState("");
    const [avatarUrl, setAvatarUrl] = useState("");
    const [bannerUrl, setBannerUrl] = useState("");
    const [bannerPosition, setBannerPosition] = useState(50);
    const [favouriteCharacter, setFavouriteCharacter] = useState("");
    const [gender, setGender] = useState("");
    const [customGender, setCustomGender] = useState("");
    const [pronounSubject, setPronounSubject] = useState("they");
    const [pronounPossessive, setPronounPossessive] = useState("their");
    const [customPronouns, setCustomPronouns] = useState(false);
    const [socialTwitter, setSocialTwitter] = useState("");
    const [socialDiscord, setSocialDiscord] = useState("");
    const [socialWaifulist, setSocialWaifulist] = useState("");
    const [socialTumblr, setSocialTumblr] = useState("");
    const [socialGithub, setSocialGithub] = useState("");
    const [website, setWebsite] = useState("");
    const [dmsEnabled, setDmsEnabled] = useState(true);
    const [episodeProgress, setEpisodeProgress] = useState(0);

    const [characters, setCharacters] = useState<Record<string, string>>({});
    const [saving, setSaving] = useState(false);
    const [uploadingAvatar, setUploadingAvatar] = useState(false);
    const [uploadingBanner, setUploadingBanner] = useState(false);
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");

    useEffect(() => {
        if (profile) {
            setDisplayName(profile.display_name);
            setBio(profile.bio);
            setAvatarUrl(profile.avatar_url);
            setBannerUrl(profile.banner_url);
            setBannerPosition(profile.banner_position ?? 50);
            setFavouriteCharacter(profile.favourite_character);
            setSocialTwitter(profile.social_twitter);
            setSocialDiscord(profile.social_discord);
            setSocialWaifulist(profile.social_waifulist);
            setSocialTumblr(profile.social_tumblr);
            setSocialGithub(profile.social_github);
            setWebsite(profile.website);
            setDmsEnabled(profile.dms_enabled ?? true);
            setEpisodeProgress(profile.episode_progress ?? 0);

            const g = initGender(profile);
            setGender(g.gender);
            setCustomGender(g.customGender);

            const p = initPronouns(profile);
            setPronounSubject(p.subject);
            setPronounPossessive(p.possessive);
            setCustomPronouns(p.isCustom);
        }
    }, [profile]);

    useEffect(() => {
        getCharacters()
            .then(setCharacters)
            .catch(() => setCharacters({}));
    }, []);

    function handleGenderChange(newGender: string) {
        setGender(newGender);
        if (!customPronouns) {
            const defaults = PRONOUN_DEFAULTS[newGender] ?? PRONOUN_DEFAULTS["Custom"];
            setPronounSubject(defaults.subject);
            setPronounPossessive(defaults.possessive);
        }
    }

    function handleCustomPronounsToggle(checked: boolean) {
        setCustomPronouns(checked);
        if (!checked) {
            const defaults = PRONOUN_DEFAULTS[gender] ?? PRONOUN_DEFAULTS["Custom"];
            setPronounSubject(defaults.subject);
            setPronounPossessive(defaults.possessive);
        }
    }

    async function handleAvatarChange(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        setUploadingAvatar(true);
        setError("");
        try {
            const result = await uploadAvatar(file);
            setAvatarUrl(result.avatar_url);
            if (user) {
                setUser({ ...user, avatar_url: result.avatar_url });
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to upload avatar.");
        } finally {
            setUploadingAvatar(false);
            e.target.value = "";
        }
    }

    async function handleBannerChange(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        setUploadingBanner(true);
        setError("");
        try {
            const result = await uploadBanner(file);
            setBannerUrl(result.banner_url);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to upload banner.");
        } finally {
            setUploadingBanner(false);
        }
    }

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        setSaving(true);
        setError("");
        setSuccess("");

        const resolvedGender = gender === "Custom" ? customGender : gender;

        const payload: UpdateProfilePayload = {
            display_name: displayName,
            bio,
            avatar_url: avatarUrl,
            banner_url: bannerUrl,
            banner_position: Math.round(bannerPosition * 100) / 100,
            favourite_character: favouriteCharacter,
            gender: resolvedGender,
            pronoun_subject: pronounSubject,
            pronoun_possessive: pronounPossessive,
            social_twitter: socialTwitter,
            social_discord: socialDiscord,
            social_waifulist: socialWaifulist,
            social_tumblr: socialTumblr,
            social_github: socialGithub,
            website,
            dms_enabled: dmsEnabled,
            episode_progress: episodeProgress,
        };

        try {
            await updateProfile(payload);
            setSuccess("Profile updated successfully.");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to update profile.");
        } finally {
            setSaving(false);
        }
    }

    return {
        profileLoading,
        error,
        success,
        saving,

        displayName,
        setDisplayName,
        bio,
        setBio,
        avatarUrl,
        uploadingAvatar,
        handleAvatarChange,
        bannerUrl,
        uploadingBanner,
        handleBannerChange,
        bannerPosition,
        setBannerPosition,
        favouriteCharacter,
        setFavouriteCharacter,
        gender,
        handleGenderChange,
        customGender,
        setCustomGender,
        pronounSubject,
        setPronounSubject,
        pronounPossessive,
        setPronounPossessive,
        customPronouns,
        handleCustomPronounsToggle,
        socialTwitter,
        setSocialTwitter,
        socialDiscord,
        setSocialDiscord,
        socialWaifulist,
        setSocialWaifulist,
        socialTumblr,
        setSocialTumblr,
        socialGithub,
        setSocialGithub,
        website,
        setWebsite,
        dmsEnabled,
        setDmsEnabled,
        episodeProgress,
        setEpisodeProgress,
        characters,

        handleSubmit,
        genderOptions: GENDER_OPTIONS,
    };
}
