import { useState } from "react";
import { Link } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { ageOn, type CalendarDate } from "../../utils/time";
import styles from "./AboutPage.module.css";

interface Favourite {
    to: string;
    title: string;
    description: string;
}

interface Social {
    href?: string;
    platform: string;
    handle: string;
}

const BIRTH_DATE: CalendarDate = { year: 1994, month: 10, day: 4 };

const SOCIALS: Social[] = [
    { href: "https://x.com/VictoriqueM", platform: "X", handle: "@VictoriqueM" },
    { href: "https://x.com/FeatherineFAA", platform: "X", handle: "@FeatherineFAA" },
    { href: "https://github.com/VictoriqueMoe", platform: "GitHub", handle: "VictoriqueMoe" },
    { platform: "Discord", handle: "victorique.moe" },
];

const FAVOURITES: Favourite[] = [
    {
        to: "/rooms",
        title: "Chat rooms",
        description: "The heart of the place: people who love the same stories just talking to each other.",
    },
    {
        to: "/games/live",
        title: "Games",
        description: "Sitting down to a game with someone, or watching one that is already under way.",
    },
    {
        to: "/live",
        title: "Live streaming from OBS",
        description: "Anyone here can go live straight from OBS, and the rest of us get to watch together.",
    },
    {
        to: "/gallery",
        title: "The art people upload",
        description: "The drawings, cosplay and figures everyone shares. It never gets old.",
    },
    {
        to: "/mysteries",
        title: "Mysteries",
        description: "Puzzles told in the colours of truth. They are simply fun.",
    },
    {
        to: "/game-board",
        title: "The Game Board",
        description: "Where everyone posts and chats about anything, with a corner for each of Ryukishi's worlds.",
    },
];

export function AboutPage() {
    usePageTitle("About");
    const [age] = useState(() => ageOn(BIRTH_DATE, new Date()));

    return (
        <div className={styles.page}>
            <header className={styles.heroText}>
                <h1 dir="auto" className={styles.title}>
                    Hi, I'm Victoria
                </h1>
                <p dir="auto" className={styles.tagline}>
                    I made this place.
                </p>
                <p dir="auto" className={styles.provenance}>
                    Coded by hand, no AI.{" "}
                    <a
                        href="https://github.com/VictoriqueMoe/umineko_city_of_books/blob/main/PROVENANCE.md"
                        target="_blank"
                        rel="noopener noreferrer"
                        className={styles.externalLink}
                    >
                        Here's how
                    </a>
                </p>
            </header>

            <div className={styles.story}>
                <figure className={styles.photoFigure}>
                    <img
                        src="/about-creator.webp"
                        alt="Victoria"
                        width={900}
                        height={1200}
                        loading="eager"
                        className={styles.photo}
                    />
                    <figcaption dir="auto" className={styles.photoCaption}>
                        My Erika Furudo cosplay
                    </figcaption>
                </figure>
                <section className={styles.storySection} aria-labelledby="about-who">
                    <h2 id="about-who" dir="auto" className={`${styles.sectionTitle} ${styles.storyTitle}`}>
                        Who I am
                    </h2>
                    <p dir="auto" className={styles.storyProse}>
                        I'm a German girl living in the UK, born on 4 October 1994 ({age}).
                    </p>
                    <p dir="auto" className={styles.storyProse}>
                        By day I work in cybersecurity for a fintech company in London. The rest of the time I write
                        code because I love it, and I cosplay.
                    </p>
                    <p dir="auto" className={styles.storyProse}>
                        I love Featherine and Victorique de Blois so much that you can always catch me in my Victorique
                        cosplay as my usual daily attire!.
                    </p>
                </section>

                <section className={styles.storySection} aria-labelledby="about-why">
                    <h2 id="about-why" dir="auto" className={`${styles.sectionTitle} ${styles.storyTitle}`}>
                        Why I built it
                    </h2>
                    <p dir="auto" className={styles.storyProse}>
                        I've been a 07th Expansion fan for a long time. I was already into Higurashi when Umineko came
                        out, and it was a big deal. I fell in love with the wider When They Cry universe and its
                        ontological framework.
                    </p>
                    <p dir="auto" className={styles.storyProse}>
                        What I never found was a home for it. There was no central place for WTC fans to share theories,
                        art and fanfiction and just talk to each other, and Twitter is cringe. So I built one.
                    </p>
                </section>
            </div>

            <section className={styles.section} aria-labelledby="about-how">
                <h2 id="about-how" dir="auto" className={styles.sectionTitle}>
                    How it was built
                </h2>
                <div className={styles.card}>
                    <p dir="auto" className={styles.prose}>
                        I started it in December 2025, around the same time I was building{" "}
                        <a
                            href="https://quotes.auaurora.moe/"
                            target="_blank"
                            rel="noopener noreferrer"
                            className={styles.externalLink}
                        >
                            quotes.auaurora.moe
                        </a>
                        , which is where the quotes on this site come from.
                    </p>
                    <p dir="auto" className={styles.prose}>
                        I built all of it myself. It's a Go backend on Postgres with a React frontend, LiveKit carries
                        the live streams, and there's an Android app too.
                    </p>
                </div>
            </section>

            <section className={styles.section} aria-labelledby="about-favourites">
                <h2 id="about-favourites" dir="auto" className={styles.sectionTitle}>
                    What I love here
                </h2>
                <ul className={styles.favouriteGrid}>
                    {FAVOURITES.map(favourite => (
                        <li key={favourite.to} className={styles.favouriteItem}>
                            <Link to={favourite.to} className={styles.favouriteCard}>
                                <span dir="auto" className={styles.favouriteTitle}>
                                    {favourite.title}
                                </span>
                                <span dir="auto" className={styles.favouriteDescription}>
                                    {favourite.description}
                                </span>
                            </Link>
                        </li>
                    ))}
                </ul>
            </section>

            <section className={styles.section} aria-labelledby="about-socials">
                <h2 id="about-socials" dir="auto" className={styles.sectionTitle}>
                    Find me elsewhere
                </h2>
                <ul className={styles.socialRow} aria-label="My socials">
                    {SOCIALS.map(social => {
                        const content = (
                            <>
                                <span className={styles.socialPlatform}>{social.platform}</span>{" "}
                                <span dir="auto" className={styles.socialHandle}>
                                    {social.handle}
                                </span>
                            </>
                        );

                        return (
                            <li key={`${social.platform}:${social.handle}`} className={styles.socialItem}>
                                {social.href ? (
                                    <a
                                        href={social.href}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className={styles.socialChip}
                                    >
                                        {content}
                                    </a>
                                ) : (
                                    <span className={styles.socialChip}>{content}</span>
                                )}
                            </li>
                        );
                    })}
                </ul>
            </section>
        </div>
    );
}
