import { Link } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { PieceTrigger } from "../../features/easterEgg";
import styles from "./LandingPage.module.css";

interface FeatureCard {
    to: string;
    title: string;
    description: string;
    tag: string;
}

const features: FeatureCard[] = [
    {
        to: "/game-board",
        title: "The Game Board",
        description:
            "The main parlour. Post, reply, and debate across Umineko, Higurashi, Ciconia, Higanbana, and Rose Guns Days.",
        tag: "Discuss",
    },
    {
        to: "/theories",
        title: "Theories",
        description:
            "Declare your solution to the epitaph as blue truth. Let others nail it with red, or shatter it outright.",
        tag: "Solve",
    },
    {
        to: "/mysteries",
        title: "Mysteries",
        description: "Fan-crafted puzzles with game masters in the wings. Attempt a closed room, earn your gold.",
        tag: "Challenge",
    },
    {
        to: "/ships",
        title: "Ships",
        description: "Declare your OTPs and crackships. Vote on pairings, defend them, and debate the merits of love.",
        tag: "Ship",
    },
    {
        to: "/fanfiction",
        title: "Fanfiction",
        description:
            "Spin your own fragments. Chapter-by-chapter stories written by the community, with comments and tags.",
        tag: "Create",
    },
    {
        to: "/gallery",
        title: "Gallery",
        description:
            "Fan art galleries from across the community. Upload your own, curate collections, and admire the witches.",
        tag: "Admire",
    },
    {
        to: "/journals",
        title: "Reading Journals",
        description:
            "Live-blog your first read-through. Log your reactions, predictions, and growing suspicions in real time.",
        tag: "Chronicle",
    },
    {
        to: "/rooms",
        title: "Chat Rooms",
        description: "Synchronous group chats for episode reactions, roleplay, book clubs, and late-night theorising.",
        tag: "Gather",
    },
];

export function LandingPage() {
    usePageTitle("Welcome");
    const { user } = useAuth();

    return (
        <div className={styles.page}>
            <RulesBox page="landing" />

            <section className={styles.hero}>
                <div className={styles.heroOrnament}>{"\u2666 \u2663 \u2665 \u2660"}</div>
                <h1 className={styles.heroTitle}>
                    Umineko City of Books <PieceTrigger pieceId="piece_01" />
                </h1>
                <p className={styles.heroTagline}>Without love, it cannot be seen.</p>
                <p className={styles.heroBlurb}>
                    Welcome, new witness. The game board is always open, the tea is always hot, and the catbox is
                    forever ajar. Here we gather to declare fan theories, nail them down in red, and shatter them when
                    we can.
                </p>
                <div className={styles.heroActions}>
                    {user ? (
                        <Link to="/game-board" className={`${styles.cta} ${styles.ctaPrimary}`}>
                            Enter the Game Board
                        </Link>
                    ) : (
                        <>
                            <Link to="/login" className={`${styles.cta} ${styles.ctaPrimary}`}>
                                Sign in to Play
                            </Link>
                            <Link to="/game-board" className={`${styles.cta} ${styles.ctaGhost}`}>
                                Peek at the Board
                            </Link>
                        </>
                    )}
                </div>
            </section>

            <section className={styles.truths}>
                <h2 className={styles.sectionTitle}>The Colours of Truth</h2>
                <p className={styles.sectionIntro}>
                    Every post, theory, comment, and chat message on this site supports the four colours. Highlight some
                    text, click a colour, and your words take on their full weight.
                </p>
                <div className={styles.truthGrid}>
                    <div className={styles.truthCard}>
                        <span className="red-truth">RED TRUTH</span>
                        <p>Absolute fact. What is written in red cannot be denied.</p>
                    </div>
                    <div className={styles.truthCard}>
                        <span className="blue-truth">BLUE TRUTH</span>
                        <p>A theory. Stands until someone nails it down in red.</p>
                    </div>
                    <div className={styles.truthCard}>
                        <span className="gold-truth">GOLD TRUTH</span>
                        <p>A fragment of truth only the game master may wield.</p>
                    </div>
                    <div className={styles.truthCard}>
                        <span className="purple-truth">PURPLE TRUTH</span>
                        <p>The words of the characters themselves. No lies here.</p>
                    </div>
                </div>
            </section>

            <section className={styles.features}>
                <h2 className={styles.sectionTitle}>Choose Your Seat at the Table</h2>
                <div className={styles.featureGrid}>
                    {features.map(f => (
                        <Link key={f.to} to={f.to} className={styles.featureCard}>
                            <span className={styles.featureTag}>{f.tag}</span>
                            <h3 className={styles.featureTitle}>{f.title}</h3>
                            <p className={styles.featureDescription}>{f.description}</p>
                        </Link>
                    ))}
                </div>
            </section>

            <section className={styles.meta}>
                <blockquote className={styles.quote}>
                    "A mystery novel in which the writer reveals all the tricks at the end is heresy. A true mystery
                    must be solved by the reader."
                </blockquote>
                <p className={styles.metaCaption}>Read slowly. Theorise boldly. Be kind to fellow witches.</p>
            </section>

            {!user && (
                <section className={styles.bottomCta}>
                    <h2 className={styles.sectionTitle}>Ready to sit at the table?</h2>
                    <p className={styles.sectionIntro}>
                        Create an account to post theories, join the chat, upload art, and keep your own reading
                        journal. Everyone starts as a witness; prove yourself and the golden butterflies may notice.
                    </p>
                    <div className={styles.heroActions}>
                        <Link to="/login" className={`${styles.cta} ${styles.ctaPrimary}`}>
                            Sign in or Register
                        </Link>
                    </div>
                </section>
            )}
        </div>
    );
}
