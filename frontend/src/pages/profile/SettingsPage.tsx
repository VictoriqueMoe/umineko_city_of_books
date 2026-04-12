import React, { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useSettingsForm } from "../../hooks/useSettingsForm";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { TextArea } from "../../components/TextArea/TextArea";
import { Select } from "../../components/Select/Select";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { BlockedUsersSection } from "./BlockedUsersSection";
import { ChangePasswordSection } from "./ChangePasswordSection";
import { DangerZoneSection } from "./DangerZoneSection";
import styles from "./SettingsPage.module.css";

const SPECIAL_CHARACTERS: string[] = ["Goldsmith"];

function BannerSection({ form }: { form: ReturnType<typeof useSettingsForm> }) {
    const containerRef = useRef<HTMLDivElement>(null);
    const [dragging, setDragging] = useState(false);
    const dragStartY = useRef(0);
    const dragStartPos = useRef(0);

    const handlePointerDown = useCallback(
        (e: React.MouseEvent | React.TouchEvent) => {
            if (!form.bannerUrl) {
                return;
            }
            dragStartY.current = "touches" in e ? e.touches[0].clientY : e.clientY;
            dragStartPos.current = form.bannerPosition;
            setDragging(true);
        },
        [form.bannerUrl, form.bannerPosition],
    );

    const handlePointerMove = useCallback(
        (e: MouseEvent | TouchEvent) => {
            if (!dragging || !containerRef.current) {
                return;
            }
            const clientY = "touches" in e ? e.touches[0].clientY : e.clientY;
            const containerHeight = containerRef.current.getBoundingClientRect().height;
            const deltaPercent = ((clientY - dragStartY.current) / containerHeight) * 100;
            const newPos = Math.min(100, Math.max(0, dragStartPos.current - deltaPercent));
            form.setBannerPosition(newPos);
        },
        [dragging, form],
    );

    const handlePointerUp = useCallback(() => {
        setDragging(false);
    }, []);

    useEffect(() => {
        if (dragging) {
            document.addEventListener("mousemove", handlePointerMove);
            document.addEventListener("mouseup", handlePointerUp);
            document.addEventListener("touchmove", handlePointerMove);
            document.addEventListener("touchend", handlePointerUp);
        }
        return () => {
            document.removeEventListener("mousemove", handlePointerMove);
            document.removeEventListener("mouseup", handlePointerUp);
            document.removeEventListener("touchmove", handlePointerMove);
            document.removeEventListener("touchend", handlePointerUp);
        };
    }, [dragging, handlePointerMove, handlePointerUp]);

    return (
        <div className={styles.section}>
            <h3 className={styles.sectionTitle}>Banner</h3>
            <div className={styles.bannerSection}>
                <div
                    ref={containerRef}
                    className={styles.bannerPreview}
                    style={{ cursor: form.bannerUrl ? "grab" : undefined, userSelect: dragging ? "none" : undefined }}
                    onMouseDown={handlePointerDown}
                    onTouchStart={handlePointerDown}
                >
                    {form.bannerUrl ? (
                        <>
                            <img
                                src={form.bannerUrl}
                                alt="Banner"
                                draggable={false}
                                style={{ objectPosition: `center ${form.bannerPosition}%` }}
                            />
                            <div className={styles.bannerDragHint}>Drag to reposition</div>
                        </>
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
    );
}

export function SettingsPage() {
    usePageTitle("Settings");
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

    const uminekoEntries = Object.entries(form.characters.umineko).sort((a, b) => a[1].localeCompare(b[1]));
    const higurashiEntries = Object.entries(form.characters.higurashi).sort((a, b) => a[1].localeCompare(b[1]));

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>Settings</h2>

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

                    <BannerSection form={form} />

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
                                    <optgroup label="Umineko">
                                        {uminekoEntries.map(([id, name]) => (
                                            <option key={`umineko-${id}`} value={name}>
                                                {name}
                                            </option>
                                        ))}
                                    </optgroup>
                                    <optgroup label="Higurashi">
                                        {higurashiEntries.map(([id, name]) => (
                                            <option key={`higurashi-${id}`} value={name}>
                                                {name}
                                            </option>
                                        ))}
                                    </optgroup>
                                    <optgroup label="Special / Alias">
                                        {SPECIAL_CHARACTERS.map(name => (
                                            <option key={`special-${name}`} value={name}>
                                                {name}
                                            </option>
                                        ))}
                                    </optgroup>
                                </Select>
                            </label>
                            <label className={styles.label}>
                                VN Progress
                                <Select
                                    value={String(form.episodeProgress)}
                                    onChange={e =>
                                        form.setEpisodeProgress(Number((e.target as HTMLSelectElement).value))
                                    }
                                >
                                    <option value="0">I've read everything</option>
                                    {[1, 2, 3, 4, 5, 6, 7, 8].map(ep => (
                                        <option key={ep} value={String(ep)}>
                                            Episode {ep}
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
                        <ToggleSwitch
                            enabled={form.dmsEnabled}
                            onChange={form.setDmsEnabled}
                            label="Direct Messages"
                            description="Allow other users to send you direct messages"
                        />
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
                        <h3 className={styles.sectionTitle}>Email</h3>
                        <label className={styles.label}>
                            Email Address
                            <Input
                                type="email"
                                fullWidth
                                value={form.email}
                                onChange={e => form.setEmail(e.target.value)}
                                placeholder="your@email.com"
                            />
                        </label>
                        <ToggleSwitch
                            enabled={form.emailPublic}
                            onChange={form.setEmailPublic}
                            label="Public Email"
                            description="Show your email address on your profile"
                        />
                    </div>

                    <div className={`${styles.section} ${styles.gridFull}`}>
                        <h3 className={styles.sectionTitle}>Notifications</h3>
                        <ToggleSwitch
                            enabled={form.emailNotifications}
                            onChange={form.setEmailNotifications}
                            label="Email Notifications"
                            description="Receive email notifications for replies and upvotes on your posts"
                        />
                    </div>

                    <div className={`${styles.section} ${styles.gridFull}`}>
                        <h3 className={styles.sectionTitle}>Preferences</h3>
                        <label className={styles.label}>
                            Home Page
                            <Select value={form.homePage} onChange={e => form.setHomePage(e.target.value)}>
                                <option value="theories">Theories (Umineko)</option>
                                <option value="theories_higurashi">Theories (Higurashi)</option>
                                <option value="game_board">Game Board (General)</option>
                                <option value="game_board_umineko">Game Board (Umineko)</option>
                                <option value="game_board_higurashi">Game Board (Higurashi)</option>
                                <option value="game_board_ciconia">Game Board (Ciconia)</option>
                                <option value="game_board_higanbana">Game Board (Higanbana)</option>
                                <option value="game_board_roseguns">Game Board (Rose Guns Days)</option>
                                <option value="gallery">Gallery (General)</option>
                                <option value="gallery_umineko">Gallery (Umineko)</option>
                                <option value="gallery_higurashi">Gallery (Higurashi)</option>
                                <option value="gallery_ciconia">Gallery (Ciconia)</option>
                                <option value="quotes">Quotes</option>
                                <option value="mysteries">Mysteries</option>
                                <option value="ships">Ships</option>
                                <option value="fanfiction">Fanfiction</option>
                                <option value="journals">Reading Journals</option>
                            </Select>
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
                {form.error && <div className={styles.error}>{form.error}</div>}
                {form.success && <div className={styles.success}>{form.success}</div>}
            </form>

            <div className={styles.grid} style={{ marginTop: "1.5rem" }}>
                <BlockedUsersSection />
                <ChangePasswordSection />
                <DangerZoneSection />
            </div>
        </div>
    );
}
