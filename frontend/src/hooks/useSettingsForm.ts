import React, { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "./useAuth";
import { useProfile } from "../api/queries/profile";
import { useAllCharacters } from "../api/queries/characters";
import { useSiteInfo } from "./useSiteInfo";
import { useUpdateProfile, useUploadAvatar, useUploadBanner } from "../api/mutations/auth";
import { validateFileSize } from "../utils/fileValidation";
import type { UpdateProfilePayload, UserProfile } from "../types/api";

const GENDER_OPTIONS = ["Prefer not to say", "Male", "Female", "Custom"];

const PRONOUN_DEFAULTS: Record<string, { subject: string; possessive: string }> = {
    Male: { subject: "he", possessive: "his" },
    Female: { subject: "she", possessive: "her" },
    "Prefer not to say": { subject: "they", possessive: "their" },
    Custom: { subject: "they", possessive: "their" },
};

interface DerivedGender {
    gender: string;
    customGender: string;
}

interface DerivedPronouns {
    subject: string;
    possessive: string;
    isCustom: boolean;
}

function deriveGender(genderValue: string): DerivedGender {
    const g = genderValue || "";
    if (GENDER_OPTIONS.includes(g)) {
        return { gender: g, customGender: "" };
    }
    if (g) {
        return { gender: "Custom", customGender: g };
    }
    return { gender: "Prefer not to say", customGender: "" };
}

function derivePronouns(profile: UserProfile): DerivedPronouns {
    const subject = profile.pronoun_subject || "they";
    const possessive = profile.pronoun_possessive || "their";
    const defaults = PRONOUN_DEFAULTS[GENDER_OPTIONS.includes(profile.gender) ? profile.gender : "Custom"];
    const isCustom =
        (!!profile.pronoun_subject && subject !== defaults.subject) ||
        (!!profile.pronoun_possessive && possessive !== defaults.possessive);
    return { subject, possessive, isCustom };
}

interface FormDraft {
    profileId: string | null;
    display_name?: string;
    bio?: string;
    avatar_url?: string;
    banner_url?: string;
    banner_position?: number;
    favourite_character?: string;
    gender?: string;
    customGender?: string;
    pronoun_subject?: string;
    pronoun_possessive?: string;
    customPronouns?: boolean;
    social_twitter?: string;
    social_discord?: string;
    social_waifulist?: string;
    social_tumblr?: string;
    social_github?: string;
    website?: string;
    dms_enabled?: boolean;
    episode_progress?: number;
    higurashi_arc_progress?: number;
    ciconia_chapter_progress?: number;
    dob?: string;
    dob_public?: boolean;
    email?: string;
    email_public?: boolean;
    email_notifications?: boolean;
    play_message_sound?: boolean;
    play_notification_sound?: boolean;
    home_page?: string;
    game_board_sort?: string;
}

export function useSettingsForm() {
    const { user, setUser } = useAuth();
    const siteInfo = useSiteInfo();
    const { profile, loading: profileLoading } = useProfile(user?.username ?? "");
    const qc = useQueryClient();

    const [draft, setDraft] = useState<FormDraft>({ profileId: null });
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");

    const characters = useAllCharacters();

    const updateProfileMutation = useUpdateProfile();
    const uploadAvatarMutation = useUploadAvatar();
    const uploadBannerMutation = useUploadBanner();
    const saving = updateProfileMutation.isPending;
    const uploadingAvatar = uploadAvatarMutation.isPending;
    const uploadingBanner = uploadBannerMutation.isPending;

    const activeDraft: FormDraft =
        profile && draft.profileId === profile.id ? draft : { profileId: profile?.id ?? null };
    const baseGender = profile ? deriveGender(profile.gender) : { gender: "Prefer not to say", customGender: "" };
    const basePronouns = profile ? derivePronouns(profile) : { subject: "they", possessive: "their", isCustom: false };

    const displayName = activeDraft.display_name ?? profile?.display_name ?? "";
    const bio = activeDraft.bio ?? profile?.bio ?? "";
    const avatarUrl = activeDraft.avatar_url ?? profile?.avatar_url ?? "";
    const bannerUrl = activeDraft.banner_url ?? profile?.banner_url ?? "";
    const bannerPosition = activeDraft.banner_position ?? profile?.banner_position ?? 50;
    const favouriteCharacter = activeDraft.favourite_character ?? profile?.favourite_character ?? "";
    const gender = activeDraft.gender ?? baseGender.gender;
    const customGender = activeDraft.customGender ?? baseGender.customGender;
    const pronounSubject = activeDraft.pronoun_subject ?? basePronouns.subject;
    const pronounPossessive = activeDraft.pronoun_possessive ?? basePronouns.possessive;
    const customPronouns = activeDraft.customPronouns ?? basePronouns.isCustom;
    const socialTwitter = activeDraft.social_twitter ?? profile?.social_twitter ?? "";
    const socialDiscord = activeDraft.social_discord ?? profile?.social_discord ?? "";
    const socialWaifulist = activeDraft.social_waifulist ?? profile?.social_waifulist ?? "";
    const socialTumblr = activeDraft.social_tumblr ?? profile?.social_tumblr ?? "";
    const socialGithub = activeDraft.social_github ?? profile?.social_github ?? "";
    const website = activeDraft.website ?? profile?.website ?? "";
    const dmsEnabled = activeDraft.dms_enabled ?? profile?.dms_enabled ?? true;
    const episodeProgress = activeDraft.episode_progress ?? profile?.episode_progress ?? 0;
    const higurashiArcProgress = activeDraft.higurashi_arc_progress ?? profile?.higurashi_arc_progress ?? 0;
    const ciconiaChapterProgress = activeDraft.ciconia_chapter_progress ?? profile?.ciconia_chapter_progress ?? 0;
    const dob = activeDraft.dob ?? profile?.dob ?? "";
    const dobPublic = activeDraft.dob_public ?? profile?.dob_public ?? false;
    const email = activeDraft.email ?? profile?.email ?? "";
    const emailPublic = activeDraft.email_public ?? profile?.email_public ?? false;
    const emailNotifications = activeDraft.email_notifications ?? profile?.email_notifications ?? false;
    const playMessageSound = activeDraft.play_message_sound ?? profile?.play_message_sound ?? true;
    const playNotificationSound = activeDraft.play_notification_sound ?? profile?.play_notification_sound ?? true;
    const homePage = activeDraft.home_page ?? profile?.home_page ?? "landing";
    const gameBoardSort = activeDraft.game_board_sort ?? profile?.game_board_sort ?? "relevance";

    function patch(update: Partial<FormDraft>) {
        setDraft(prev => {
            const base = profile && prev.profileId === profile.id ? prev : { profileId: profile?.id ?? null };
            return { ...base, ...update };
        });
    }

    function setDisplayName(value: string) {
        patch({ display_name: value });
    }
    function setBio(value: string) {
        patch({ bio: value });
    }
    function setBannerPosition(value: number) {
        patch({ banner_position: value });
    }
    function setFavouriteCharacter(value: string) {
        patch({ favourite_character: value });
    }
    function setCustomGender(value: string) {
        patch({ customGender: value });
    }
    function setPronounSubject(value: string) {
        patch({ pronoun_subject: value });
    }
    function setPronounPossessive(value: string) {
        patch({ pronoun_possessive: value });
    }
    function setSocialTwitter(value: string) {
        patch({ social_twitter: value });
    }
    function setSocialDiscord(value: string) {
        patch({ social_discord: value });
    }
    function setSocialWaifulist(value: string) {
        patch({ social_waifulist: value });
    }
    function setSocialTumblr(value: string) {
        patch({ social_tumblr: value });
    }
    function setSocialGithub(value: string) {
        patch({ social_github: value });
    }
    function setWebsite(value: string) {
        patch({ website: value });
    }
    function setDmsEnabled(value: boolean) {
        patch({ dms_enabled: value });
    }
    function setEpisodeProgress(value: number) {
        patch({ episode_progress: value });
    }
    function setHigurashiArcProgress(value: number) {
        patch({ higurashi_arc_progress: value });
    }
    function setCiconiaChapterProgress(value: number) {
        patch({ ciconia_chapter_progress: value });
    }
    function setDob(value: string) {
        patch({ dob: value });
    }
    function setDobPublic(value: boolean) {
        patch({ dob_public: value });
    }
    function setEmail(value: string) {
        patch({ email: value });
    }
    function setEmailPublic(value: boolean) {
        patch({ email_public: value });
    }
    function setEmailNotifications(value: boolean) {
        patch({ email_notifications: value });
    }
    function setPlayMessageSound(value: boolean) {
        patch({ play_message_sound: value });
    }
    function setPlayNotificationSound(value: boolean) {
        patch({ play_notification_sound: value });
    }
    function setHomePage(value: string) {
        patch({ home_page: value });
    }
    function setGameBoardSort(value: string) {
        patch({ game_board_sort: value });
    }

    function handleGenderChange(newGender: string) {
        if (customPronouns) {
            patch({ gender: newGender });
        } else {
            const defaults = PRONOUN_DEFAULTS[newGender] ?? PRONOUN_DEFAULTS["Custom"];
            patch({
                gender: newGender,
                pronoun_subject: defaults.subject,
                pronoun_possessive: defaults.possessive,
            });
        }
    }

    function handleCustomPronounsToggle(checked: boolean) {
        if (checked) {
            patch({ customPronouns: true });
            return;
        }
        const defaults = PRONOUN_DEFAULTS[gender] ?? PRONOUN_DEFAULTS["Custom"];
        patch({
            customPronouns: false,
            pronoun_subject: defaults.subject,
            pronoun_possessive: defaults.possessive,
        });
    }

    async function handleAvatarChange(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        const sizeErr = validateFileSize(file, siteInfo.max_image_size, siteInfo.max_video_size);
        if (sizeErr) {
            setError(sizeErr);
            e.target.value = "";
            return;
        }
        setError("");
        try {
            const result = await uploadAvatarMutation.mutateAsync(file);
            patch({ avatar_url: result.avatar_url });
            if (user) {
                setUser({ ...user, avatar_url: result.avatar_url });
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to upload avatar.");
        } finally {
            e.target.value = "";
        }
    }

    async function handleBannerChange(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        const sizeErr = validateFileSize(file, siteInfo.max_image_size, siteInfo.max_video_size);
        if (sizeErr) {
            setError(sizeErr);
            return;
        }
        setError("");
        try {
            const result = await uploadBannerMutation.mutateAsync(file);
            patch({ banner_url: result.banner_url });
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to upload banner.");
        }
    }

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
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
            higurashi_arc_progress: higurashiArcProgress,
            ciconia_chapter_progress: ciconiaChapterProgress,
            dob,
            dob_public: dobPublic,
            email,
            email_public: emailPublic,
            email_notifications: emailNotifications,
            play_message_sound: playMessageSound,
            play_notification_sound: playNotificationSound,
            home_page: homePage,
            game_board_sort: gameBoardSort,
        };

        try {
            await updateProfileMutation.mutateAsync(payload);
            try {
                await qc.refetchQueries({ queryKey: ["auth", "me"] });
            } catch {
                return;
            }
            setSuccess("Profile updated successfully.");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to update profile.");
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
        higurashiArcProgress,
        setHigurashiArcProgress,
        ciconiaChapterProgress,
        setCiconiaChapterProgress,
        dob,
        setDob,
        dobPublic,
        setDobPublic,
        email,
        setEmail,
        emailPublic,
        setEmailPublic,
        emailNotifications,
        setEmailNotifications,
        playMessageSound,
        setPlayMessageSound,
        playNotificationSound,
        setPlayNotificationSound,
        homePage,
        setHomePage,
        gameBoardSort,
        setGameBoardSort,
        characters,

        handleSubmit,
        genderOptions: GENDER_OPTIONS,
    };
}
