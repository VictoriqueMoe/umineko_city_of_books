import {useEffect} from "react";
import {useNavigate} from "react-router";
import {useAuth} from "../../hooks/useAuth";
import {useSettingsForm} from "../../hooks/useSettingsForm";
import {Button} from "../../components/Button/Button";
import {Input} from "../../components/Input/Input";
import {TextArea} from "../../components/TextArea/TextArea";
import {Select} from "../../components/Select/Select";
import {ToggleSwitch} from "../../components/ToggleSwitch/ToggleSwitch";
import {ChangePasswordSection} from "./ChangePasswordSection";
import {DangerZoneSection} from "./DangerZoneSection";
import styles from "./SettingsPage.module.css";

export function SettingsPage() {
    const navigate = useNavigate();
    const { user, loading: authLoading } = useAuth();
    const form = useSettingsForm();

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (!user) {
        return null;
    }

    if (form.profileLoading) {
        return <div className="loading">Loading settings...</div>;
    }

    const characterEntries = Object.entries(form.characters).sort((a, b) => a[1].localeCompare(b[1]));

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>Settings</h2>

            {form.error && <div className={styles.error}>{form.error}</div>}
            {form.success && <div className={styles.success}>{form.success}</div>}

            <form onSubmit={form.handleSubmit}>
                <div className={styles.grid}>
                    <div className={styles.section}>
                        <h3 className={styles.sectionTitle}>Avatar</h3>
                        <div className={styles.avatarSection}>
                            <div className={styles.avatarPreview}>
                                {form.avatarUrl ? (
                                    <img src={form.avatarUrl} alt="Avatar" />
                                ) : (
                                    <div className={styles.avatarPlaceholder}>
                                        {form.displayName ? form.displayName.charAt(0).toUpperCase() : "?"}
                                    </div>
                                )}
                            </div>
                            <label className={styles.uploadBtn}>
                                {form.uploadingAvatar ? "Uploading..." : "Upload Avatar"}
                                <input
                                    type="file"
                                    accept="image/*"
                                    onChange={form.handleAvatarChange}
                                    style={{ display: "none" }}
                                    disabled={form.uploadingAvatar}
                                />
                            </label>
                        </div>
                    </div>

                    <div className={styles.section}>
                        <h3 className={styles.sectionTitle}>Banner</h3>
                        <div className={styles.bannerSection}>
                            <div className={styles.bannerPreview}>
                                {form.bannerUrl ? (
                                    <img src={form.bannerUrl} alt="Banner" />
                                ) : (
                                    <div className={styles.bannerPlaceholder}>No banner set</div>
                                )}
                            </div>
                            <label className={styles.uploadBtn}>
                                {form.uploadingBanner ? "Uploading..." : "Upload Banner"}
                                <input
                                    type="file"
                                    accept="image/*"
                                    onChange={form.handleBannerChange}
                                    style={{ display: "none" }}
                                    disabled={form.uploadingBanner}
                                />
                            </label>
                        </div>
                    </div>

                    <div className={`${styles.section} ${styles.gridFull}`}>
                        <h3 className={styles.sectionTitle}>Profile</h3>
                        <div className={styles.twoCol}>
                            <label className={styles.label}>
                                Display Name
                                <Input
                                    type="text"
                                    fullWidth
                                    value={form.displayName}
                                    onChange={e => form.setDisplayName(e.target.value)}
                                />
                            </label>
                            <label className={styles.label}>
                                Favourite Character
                                <Select
                                    value={form.favouriteCharacter}
                                    onChange={e => form.setFavouriteCharacter((e.target as HTMLSelectElement).value)}
                                >
                                    <option value="">None</option>
                                    {characterEntries.map(([id, name]) => (
                                        <option key={id} value={name}>
                                            {name}
                                        </option>
                                    ))}
                                </Select>
                            </label>
                        </div>
                        <div>
                            <label className={styles.label}>
                                Gender
                                <Select
                                    value={form.gender}
                                    onChange={e => form.handleGenderChange((e.target as HTMLSelectElement).value)}
                                >
                                    {form.genderOptions.map(opt => (
                                        <option key={opt} value={opt}>
                                            {opt}
                                        </option>
                                    ))}
                                </Select>
                            </label>
                            {form.gender === "Custom" && (
                                <label className={styles.label}>
                                    Custom Gender
                                    <Input
                                        type="text"
                                        fullWidth
                                        value={form.customGender}
                                        onChange={e => form.setCustomGender(e.target.value)}
                                        placeholder="Enter your gender"
                                    />
                                </label>
                            )}
                            <div className={styles.pronounRow}>
                                <span className={styles.pronounPreview}>
                                    Pronouns: {form.pronounSubject}/{form.pronounPossessive}
                                </span>
                                <ToggleSwitch
                                    enabled={form.customPronouns}
                                    onChange={form.handleCustomPronounsToggle}
                                    label="Custom pronouns"
                                />
                            </div>
                            {form.customPronouns && (
                                <div className={styles.twoCol}>
                                    <label className={styles.label}>
                                        Subject (e.g. she, he, they)
                                        <Input
                                            type="text"
                                            fullWidth
                                            value={form.pronounSubject}
                                            onChange={e => form.setPronounSubject(e.target.value)}
                                            placeholder="they"
                                        />
                                    </label>
                                    <label className={styles.label}>
                                        Possessive (e.g. her, his, their)
                                        <Input
                                            type="text"
                                            fullWidth
                                            value={form.pronounPossessive}
                                            onChange={e => form.setPronounPossessive(e.target.value)}
                                            placeholder="their"
                                        />
                                    </label>
                                </div>
                            )}
                        </div>
                        <label className={styles.label}>
                            Bio
                            <TextArea
                                value={form.bio}
                                onChange={e => form.setBio(e.target.value)}
                                rows={3}
                                placeholder="Tell others about yourself on the game board..."
                            />
                        </label>
                    </div>

                    <div className={`${styles.section} ${styles.gridFull}`}>
                        <h3 className={styles.sectionTitle}>Social Links</h3>
                        <div className={styles.twoCol}>
                            <label className={styles.label}>
                                Twitter / X
                                <Input
                                    type="text"
                                    fullWidth
                                    value={form.socialTwitter}
                                    onChange={e => form.setSocialTwitter(e.target.value)}
                                    placeholder="username"
                                />
                            </label>
                            <label className={styles.label}>
                                Discord
                                <Input
                                    type="text"
                                    fullWidth
                                    value={form.socialDiscord}
                                    onChange={e => form.setSocialDiscord(e.target.value)}
                                    placeholder="username#0000"
                                />
                            </label>
                            <label className={styles.label}>
                                WaifuList
                                <Input
                                    type="text"
                                    fullWidth
                                    value={form.socialWaifulist}
                                    onChange={e => form.setSocialWaifulist(e.target.value)}
                                    placeholder="https://waifulist.moe/list/..."
                                />
                            </label>
                            <label className={styles.label}>
                                Tumblr
                                <Input
                                    type="text"
                                    fullWidth
                                    value={form.socialTumblr}
                                    onChange={e => form.setSocialTumblr(e.target.value)}
                                    placeholder="username"
                                />
                            </label>
                            <label className={styles.label}>
                                GitHub
                                <Input
                                    type="text"
                                    fullWidth
                                    value={form.socialGithub}
                                    onChange={e => form.setSocialGithub(e.target.value)}
                                    placeholder="username"
                                />
                            </label>
                            <label className={styles.label}>
                                Website
                                <Input
                                    type="text"
                                    fullWidth
                                    value={form.website}
                                    onChange={e => form.setWebsite(e.target.value)}
                                    placeholder="https://example.com"
                                />
                            </label>
                        </div>
                    </div>
                </div>

                <Button variant="primary" type="submit" disabled={form.saving} style={{ width: "100%" }}>
                    {form.saving ? "Saving..." : "Save Changes"}
                </Button>
            </form>

            <div className={styles.grid} style={{ marginTop: "1.5rem" }}>
                <ChangePasswordSection />
                <DangerZoneSection />
            </div>
        </div>
    );
}
