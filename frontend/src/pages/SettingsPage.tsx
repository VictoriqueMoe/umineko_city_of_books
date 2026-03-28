import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../hooks/useAuth";
import { useProfile } from "../hooks/useProfile";
import { getCharacters, updateProfile, uploadAvatar } from "../api/endpoints";
import type { UpdateProfilePayload } from "../types/api";

export function SettingsPage() {
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const { profile, loading: profileLoading } = useProfile(user?.username ?? "");

    const [displayName, setDisplayName] = useState("");
    const [bio, setBio] = useState("");
    const [avatarUrl, setAvatarUrl] = useState("");
    const [favouriteCharacter, setFavouriteCharacter] = useState("");
    const [socialTwitter, setSocialTwitter] = useState("");
    const [socialDiscord, setSocialDiscord] = useState("");
    const [socialWaifulist, setSocialWaifulist] = useState("");
    const [socialTumblr, setSocialTumblr] = useState("");
    const [socialGithub, setSocialGithub] = useState("");
    const [website, setWebsite] = useState("");

    const [characters, setCharacters] = useState<Record<string, string>>({});
    const [saving, setSaving] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    useEffect(() => {
        if (profile) {
            setDisplayName(profile.display_name);
            setBio(profile.bio);
            setAvatarUrl(profile.avatar_url);
            setFavouriteCharacter(profile.favourite_character);
            setSocialTwitter(profile.social_twitter);
            setSocialDiscord(profile.social_discord);
            setSocialWaifulist(profile.social_waifulist);
            setSocialTumblr(profile.social_tumblr);
            setSocialGithub(profile.social_github);
            setWebsite(profile.website);
        }
    }, [profile]);

    useEffect(() => {
        getCharacters()
            .then(setCharacters)
            .catch(() => setCharacters({}));
    }, []);

    if (!user) {
        return null;
    }

    if (profileLoading) {
        return <div className="loading">Loading settings...</div>;
    }

    async function handleAvatarChange(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }
        setUploading(true);
        setError("");
        try {
            const result = await uploadAvatar(file);
            setAvatarUrl(result.avatar_url);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to upload avatar.");
        } finally {
            setUploading(false);
        }
    }

    async function handleSubmit(e: React.SubmitEvent) {
        e.preventDefault();
        setSaving(true);
        setError("");
        setSuccess("");

        const payload: UpdateProfilePayload = {
            display_name: displayName,
            bio,
            avatar_url: avatarUrl,
            favourite_character: favouriteCharacter,
            social_twitter: socialTwitter,
            social_discord: socialDiscord,
            social_waifulist: socialWaifulist,
            social_tumblr: socialTumblr,
            social_github: socialGithub,
            website,
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

    const characterEntries = Object.entries(characters).sort((a, b) => a[1].localeCompare(b[1]));

    return (
        <div className="settings-page">
            <h2 className="logic-title">Settings</h2>

            {error && <div className="login-error">{error}</div>}
            {success && <div className="settings-success">{success}</div>}

            <form onSubmit={handleSubmit}>
                <div className="settings-section">
                    <h3 className="section-title">Avatar</h3>
                    <div className="settings-avatar-section">
                        <div className="settings-avatar-preview">
                            {avatarUrl ? (
                                <img src={avatarUrl} alt="Avatar" />
                            ) : (
                                <div className="avatar-placeholder">
                                    {displayName ? displayName.charAt(0).toUpperCase() : "?"}
                                </div>
                            )}
                        </div>
                        <label className="nav-btn settings-upload-btn">
                            {uploading ? "Uploading..." : "Upload Avatar"}
                            <input
                                type="file"
                                accept="image/*"
                                onChange={handleAvatarChange}
                                style={{ display: "none" }}
                                disabled={uploading}
                            />
                        </label>
                    </div>
                </div>

                <div className="settings-section">
                    <h3 className="section-title">Profile</h3>
                    <label className="settings-label">
                        Display Name
                        <input
                            type="text"
                            className="login-input"
                            value={displayName}
                            onChange={e => setDisplayName(e.target.value)}
                        />
                    </label>
                    <label className="settings-label">
                        Bio
                        <textarea
                            className="login-input settings-textarea"
                            value={bio}
                            onChange={e => setBio(e.target.value)}
                            rows={4}
                            placeholder="Tell others about yourself on the game board..."
                        />
                    </label>
                    <label className="settings-label">
                        Favourite Character
                        <select
                            className="filter-select settings-select"
                            value={favouriteCharacter}
                            onChange={e => setFavouriteCharacter(e.target.value)}
                        >
                            <option value="">None</option>
                            {characterEntries.map(([id, name]) => (
                                <option key={id} value={name}>
                                    {name}
                                </option>
                            ))}
                        </select>
                    </label>
                </div>

                <div className="settings-section">
                    <h3 className="section-title">Social Links</h3>
                    <label className="settings-label">
                        Twitter / X
                        <input
                            type="text"
                            className="login-input"
                            value={socialTwitter}
                            onChange={e => setSocialTwitter(e.target.value)}
                            placeholder="username"
                        />
                    </label>
                    <label className="settings-label">
                        Discord
                        <input
                            type="text"
                            className="login-input"
                            value={socialDiscord}
                            onChange={e => setSocialDiscord(e.target.value)}
                            placeholder="username#0000"
                        />
                    </label>
                    <label className="settings-label">
                        WaifuList
                        <input
                            type="text"
                            className="login-input"
                            value={socialWaifulist}
                            onChange={e => setSocialWaifulist(e.target.value)}
                            placeholder="username"
                        />
                    </label>
                    <label className="settings-label">
                        Tumblr
                        <input
                            type="text"
                            className="login-input"
                            value={socialTumblr}
                            onChange={e => setSocialTumblr(e.target.value)}
                            placeholder="username"
                        />
                    </label>
                    <label className="settings-label">
                        GitHub
                        <input
                            type="text"
                            className="login-input"
                            value={socialGithub}
                            onChange={e => setSocialGithub(e.target.value)}
                            placeholder="username"
                        />
                    </label>
                    <label className="settings-label">
                        Website
                        <input
                            type="text"
                            className="login-input"
                            value={website}
                            onChange={e => setWebsite(e.target.value)}
                            placeholder="https://example.com"
                        />
                    </label>
                </div>

                <button className="login-submit settings-save" type="submit" disabled={saving}>
                    {saving ? "Saving..." : "Save Changes"}
                </button>
            </form>
        </div>
    );
}
