import {useNavigate, useParams} from "react-router";
import {useProfile} from "../hooks/useProfile";
import {useTheoryFeed} from "../hooks/useTheoryFeed";
import {TheoryCard} from "../components/theory/TheoryCard";
import {Pagination} from "../components/common/Pagination";

const SOCIAL_LABELS: Record<string, string> = {
    social_twitter: "Twitter / X",
    social_discord: "Discord",
    social_waifulist: "WaifuList",
    social_tumblr: "Tumblr",
    social_github: "GitHub",
};

function formatDate(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleDateString("en-GB", { year: "numeric", month: "long", day: "numeric" });
}

function socialUrl(key: string, value: string): string {
    if (value.startsWith("http://") || value.startsWith("https://")) {
        return value;
    }
    switch (key) {
        case "social_twitter":
            return `https://x.com/${value}`;
        case "social_github":
            return `https://github.com/${value}`;
        case "social_tumblr":
            return `https://${value}.tumblr.com`;
        case "social_waifulist":
            return `https://waifulist.moe/user/${value}`;
        default:
            return value;
    }
}

export function ProfilePage() {
    const { username } = useParams<{ username: string }>();
    const navigate = useNavigate();
    const { profile, loading } = useProfile(username ?? "");
    const {
        theories,
        total,
        loading: theoriesLoading,
        offset,
        limit,
        goNext,
        goPrev,
        hasNext,
        hasPrev,
    } = useTheoryFeed("new", 0, profile?.id);

    if (loading) {
        return <div className="loading">Consulting the game board...</div>;
    }

    if (!profile) {
        return (
            <div className="empty-state">
                Player not found on the game board.
                <br />
                <button className="nav-btn" onClick={() => navigate("/")} style={{ marginTop: "1rem" }}>
                    Return to Feed
                </button>
            </div>
        );
    }

    const socialEntries = Object.entries(SOCIAL_LABELS)
        .map(([key, label]) => ({
            key,
            label,
            value: profile[key as keyof typeof profile] as string,
        }))
        .filter(entry => entry.value);

    if (profile.website) {
        socialEntries.push({
            key: "website",
            label: "Website",
            value: profile.website,
        });
    }

    return (
        <div className="profile-page">
            <div className="profile-header">
                <div className="profile-avatar">
                    {profile.avatar_url ? (
                        <img src={profile.avatar_url} alt={profile.display_name} />
                    ) : (
                        <div className="avatar-placeholder">{profile.display_name.charAt(0).toUpperCase()}</div>
                    )}
                </div>
                <div className="profile-info">
                    <h1 className="profile-display-name">{profile.display_name}</h1>
                    <span className="profile-username">@{profile.username}</span>
                    <span className="profile-join-date">Joined {formatDate(profile.created_at)}</span>
                </div>
            </div>

            <div className="profile-bio">{profile.bio || "This player has not written a bio yet."}</div>

            {profile.favourite_character && (
                <div className="profile-favourite">
                    <span className="profile-favourite-label">Favourite Character</span>
                    <span className="profile-favourite-value">{profile.favourite_character}</span>
                </div>
            )}

            <div className="profile-stats">
                <div className="profile-stat-box">
                    <span className="profile-stat-number">{profile.stats.theory_count}</span>
                    <span className="profile-stat-label">Theories</span>
                </div>
                <div className="profile-stat-box">
                    <span className="profile-stat-number">{profile.stats.response_count}</span>
                    <span className="profile-stat-label">Responses</span>
                </div>
                <div className="profile-stat-box">
                    <span className="profile-stat-number">{profile.stats.votes_received}</span>
                    <span className="profile-stat-label">Votes Received</span>
                </div>
            </div>

            {socialEntries.length > 0 && (
                <div className="profile-social-links">
                    <h3 className="section-title">Links</h3>
                    <ul>
                        {socialEntries.map(entry => (
                            <li key={entry.key}>
                                <span className="social-link-label">{entry.label}</span>
                                {entry.key === "social_discord" ? (
                                    <span className="social-link-value">{entry.value}</span>
                                ) : (
                                    <a
                                        className="social-link-value"
                                        href={
                                            entry.key === "website"
                                                ? entry.value.startsWith("http")
                                                    ? entry.value
                                                    : `https://${entry.value}`
                                                : socialUrl(entry.key, entry.value)
                                        }
                                        target="_blank"
                                        rel="noopener noreferrer"
                                    >
                                        {entry.value}
                                    </a>
                                )}
                            </li>
                        ))}
                    </ul>
                </div>
            )}

            <div className="profile-theories">
                <h3 className="section-title">Theories</h3>

                {theoriesLoading && <div className="loading">Loading theories...</div>}

                {!theoriesLoading && theories.length === 0 && (
                    <div className="empty-state">This player has not declared any theories yet.</div>
                )}

                {!theoriesLoading && theories.map(theory => <TheoryCard key={theory.id} theory={theory} />)}

                {!theoriesLoading && total > 0 && (
                    <Pagination
                        offset={offset}
                        limit={limit}
                        total={total}
                        hasNext={hasNext}
                        hasPrev={hasPrev}
                        onNext={goNext}
                        onPrev={goPrev}
                    />
                )}
            </div>
        </div>
    );
}
