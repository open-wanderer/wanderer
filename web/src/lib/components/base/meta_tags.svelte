<script lang="ts">
    /**
     * Component adapted from: https://github.com/oekazuma/svelte-meta-tags/tree/main
     * Original author: @oekazuma
     * License: MIT
     */

    interface MobileAlternate {
        media: string;
        href: string;
    }

    interface LanguageAlternate {
        hrefLang: string;
        href: string;
    }

    interface AdditionalRobotsProps {
        nosnippet?: boolean;
        maxSnippet?: number;
        maxImagePreview?: "none" | "standard" | "large";
        maxVideoPreview?: number;
        noarchive?: boolean;
        unavailableAfter?: string;
        noimageindex?: boolean;
        notranslate?: boolean;
    }

    interface OpenGraph {
        url?: string;
        type?: string;
        title?: string;
        description?: string;
        images?: ReadonlyArray<OpenGraphImage>;
        videos?: ReadonlyArray<OpenGraphVideos>;
        audio?: ReadonlyArray<OpenGraphAudio>;
        locale?: string;
        siteName?: string;
        profile?: OpenGraphProfile;
        book?: OpenGraphBook;
        article?: OpenGraphArticle;
        video?: OpenGraphVideo;
    }

    interface OpenGraphImage {
        url: string;
        secureUrl?: string;
        type?: string;
        width?: number;
        height?: number;
        alt?: string;
    }

    interface OpenGraphVideos {
        url: string;
        secureUrl?: string;
        type?: string;
        width?: number;
        height?: number;
    }

    interface OpenGraphAudio {
        url: string;
        secureUrl?: string;
        type?: string;
    }

    interface OpenGraphProfile {
        firstName?: string;
        lastName?: string;
        username?: string;
        gender?: string;
    }

    interface OpenGraphBook {
        authors?: ReadonlyArray<string>;
        isbn?: string;
        releaseDate?: string;
        tags?: ReadonlyArray<string>;
    }

    interface OpenGraphArticle {
        publishedTime?: string;
        modifiedTime?: string;
        expirationTime?: string;
        authors?: ReadonlyArray<string>;
        section?: string;
        tags?: ReadonlyArray<string>;
    }

    interface OpenGraphVideo {
        actors?: ReadonlyArray<OpenGraphVideoActors>;
        directors?: ReadonlyArray<string>;
        writers?: ReadonlyArray<string>;
        duration?: number;
        releaseDate?: string;
        tags?: ReadonlyArray<string>;
        series?: string;
    }

    interface OpenGraphVideoActors {
        profile: string;
        role?: string;
    }

    interface BaseMetaTag {
        content: string;
    }

    interface HTML5MetaTag extends BaseMetaTag {
        name: string;
        property?: undefined;
        httpEquiv?: undefined;
    }

    interface RDFaMetaTag extends BaseMetaTag {
        property: string;
        name?: undefined;
        httpEquiv?: undefined;
    }

    interface HTTPEquivMetaTag extends BaseMetaTag {
        httpEquiv:
            | "content-security-policy"
            | "content-type"
            | "default-style"
            | "x-ua-compatible"
            | "refresh";
        name?: undefined;
        property?: undefined;
    }

    type MetaTag = HTML5MetaTag | RDFaMetaTag | HTTPEquivMetaTag;

    interface LinkTag {
        rel: string;
        href: string;
        hrefLang?: string;
        title?: string;
        media?: string;
        sizes?: string;
        type?: string;
        color?: string;
        imagesrcset?: string;
        imagesizes?: string;
        integrity?: string;
        as?:
            | "fetch"
            | "audio"
            | "audioworklet"
            | "document"
            | "embed"
            | "font"
            | "frame"
            | "iframe"
            | "image"
            | "json"
            | "manifest"
            | "object"
            | "paintworklet"
            | "report"
            | "script"
            | "serviceworker"
            | "sharedworker"
            | "style"
            | "track"
            | "video"
            | "webidentity"
            | "worker"
            | "xslt";
        crossOrigin?: "anonymous" | "use-credentials";
        referrerPolicy?: ReferrerPolicy;
    }

    interface MetaTagsProps {
        title?: string;
        titleTemplate?: string;
        robots?: string | boolean;
        additionalRobotsProps?: AdditionalRobotsProps;
        description?: string;
        canonical?: string;
        mobileAlternate?: MobileAlternate;
        languageAlternates?: ReadonlyArray<LanguageAlternate>;
        openGraph?: OpenGraph;
        additionalMetaTags?: ReadonlyArray<MetaTag>;
        additionalLinkTags?: ReadonlyArray<LinkTag>;
        keywords?: ReadonlyArray<string>;
    }

    let {
        title = undefined,
        titleTemplate = undefined,
        robots = "index,follow",
        additionalRobotsProps = undefined,
        description = undefined,
        mobileAlternate = undefined,
        languageAlternates = undefined,
        openGraph = undefined,
        canonical = undefined,
        keywords = undefined,
        additionalMetaTags = undefined,
        additionalLinkTags = undefined,
    }: Partial<MetaTagsProps> = $props();

    let updatedTitle = $derived(
        titleTemplate
            ? title
                ? titleTemplate.replace(/%s/g, title)
                : title
            : title,
    );

    let robotsParams = $derived.by(() => {
        if (!additionalRobotsProps) {
            return "";
        }
        const {
            nosnippet,
            maxSnippet,
            maxImagePreview,
            maxVideoPreview,
            noarchive,
            noimageindex,
            notranslate,
            unavailableAfter,
        } = additionalRobotsProps;

        return `${nosnippet ? ",nosnippet" : ""}${maxSnippet ? `,max-snippet:${maxSnippet}` : ""}${
            maxImagePreview ? `,max-image-preview:${maxImagePreview}` : ""
        }${noarchive ? ",noarchive" : ""}${unavailableAfter ? `,unavailable_after:${unavailableAfter}` : ""}${
            noimageindex ? ",noimageindex" : ""
        }${maxVideoPreview ? `,max-video-preview:${maxVideoPreview}` : ""}${notranslate ? ",notranslate" : ""}`;
    });

    $effect(() => {
        if (!robots && additionalRobotsProps) {
            console.warn(
                "additionalRobotsProps cannot be used when robots is set to false",
            );
        }
    });
</script>

<svelte:head>
    {#if updatedTitle}
        <title>{updatedTitle}</title>
    {/if}

    {#if robots !== false}
        <meta name="robots" content="{robots}{robotsParams}" />
    {/if}

    {#if description}
        <meta name="description" content={description} />
    {/if}

    {#if canonical}
        <link rel="canonical" href={canonical} />
    {/if}

    {#if keywords?.length}
        <meta name="keywords" content={keywords.join(", ")} />
    {/if}

    {#if mobileAlternate}
        <link
            rel="alternate"
            media={mobileAlternate.media}
            href={mobileAlternate.href}
        />
    {/if}

    {#if languageAlternates && languageAlternates.length > 0}
        {#each languageAlternates as languageAlternate (languageAlternate)}
            <link
                rel="alternate"
                hrefLang={languageAlternate.hrefLang}
                href={languageAlternate.href}
            />
        {/each}
    {/if}

    {#if openGraph}
        {#if openGraph.url || canonical}
            <meta property="og:url" content={openGraph.url || canonical} />
        {/if}

        {#if openGraph.type}
            <meta property="og:type" content={openGraph.type.toLowerCase()} />
            {#if openGraph.type.toLowerCase() === "profile" && openGraph.profile}
                {#if openGraph.profile.firstName}
                    <meta
                        property="profile:first_name"
                        content={openGraph.profile.firstName}
                    />
                {/if}

                {#if openGraph.profile.lastName}
                    <meta
                        property="profile:last_name"
                        content={openGraph.profile.lastName}
                    />
                {/if}

                {#if openGraph.profile.username}
                    <meta
                        property="profile:username"
                        content={openGraph.profile.username}
                    />
                {/if}

                {#if openGraph.profile.gender}
                    <meta
                        property="profile:gender"
                        content={openGraph.profile.gender}
                    />
                {/if}
            {:else if openGraph.type.toLowerCase() === "book" && openGraph.book}
                {#if openGraph.book.authors && openGraph.book.authors.length}
                    {#each openGraph.book.authors as author (author)}
                        <meta property="book:author" content={author} />
                    {/each}
                {/if}

                {#if openGraph.book.isbn}
                    <meta property="book:isbn" content={openGraph.book.isbn} />
                {/if}

                {#if openGraph.book.releaseDate}
                    <meta
                        property="book:release_date"
                        content={openGraph.book.releaseDate}
                    />
                {/if}

                {#if openGraph.book.tags && openGraph.book.tags.length}
                    {#each openGraph.book.tags as tag (tag)}
                        <meta property="book:tag" content={tag} />
                    {/each}
                {/if}
            {:else if openGraph.type.toLowerCase() === "article" && openGraph.article}
                {#if openGraph.article.publishedTime}
                    <meta
                        property="article:published_time"
                        content={openGraph.article.publishedTime}
                    />
                {/if}

                {#if openGraph.article.modifiedTime}
                    <meta
                        property="article:modified_time"
                        content={openGraph.article.modifiedTime}
                    />
                {/if}

                {#if openGraph.article.expirationTime}
                    <meta
                        property="article:expiration_time"
                        content={openGraph.article.expirationTime}
                    />
                {/if}

                {#if openGraph.article.authors && openGraph.article.authors.length}
                    {#each openGraph.article.authors as author (author)}
                        <meta property="article:author" content={author} />
                    {/each}
                {/if}

                {#if openGraph.article.section}
                    <meta
                        property="article:section"
                        content={openGraph.article.section}
                    />
                {/if}

                {#if openGraph.article.tags && openGraph.article.tags.length}
                    {#each openGraph.article.tags as tag (tag)}
                        <meta property="article:tag" content={tag} />
                    {/each}
                {/if}
            {:else if openGraph.type.toLowerCase() === "video.movie" || openGraph.type.toLowerCase() === "video.episode" || openGraph.type.toLowerCase() === "video.tv_show" || (openGraph.type.toLowerCase() === "video.other" && openGraph.video)}
                {#if openGraph.video?.actors && openGraph.video.actors.length}
                    {#each openGraph.video.actors as actor (actor)}
                        {#if actor.profile}
                            <meta
                                property="video:actor"
                                content={actor.profile}
                            />
                        {/if}
                        {#if actor.role}
                            <meta
                                property="video:actor:role"
                                content={actor.role}
                            />
                        {/if}
                    {/each}
                {/if}

                {#if openGraph.video?.directors && openGraph.video.directors.length}
                    {#each openGraph.video.directors as director (director)}
                        <meta property="video:director" content={director} />
                    {/each}
                {/if}

                {#if openGraph.video?.writers && openGraph.video.writers.length}
                    {#each openGraph.video.writers as writer (writer)}
                        <meta property="video:writer" content={writer} />
                    {/each}
                {/if}

                {#if openGraph.video?.duration}
                    <meta
                        property="video:duration"
                        content={openGraph.video.duration.toString()}
                    />
                {/if}

                {#if openGraph.video?.releaseDate}
                    <meta
                        property="video:release_date"
                        content={openGraph.video.releaseDate}
                    />
                {/if}

                {#if openGraph.video?.tags && openGraph.video.tags.length}
                    {#each openGraph.video.tags as tag (tag)}
                        <meta property="video:tag" content={tag} />
                    {/each}
                {/if}

                {#if openGraph.video?.series}
                    <meta
                        property="video:series"
                        content={openGraph.video.series}
                    />
                {/if}
            {/if}
        {/if}

        {#if openGraph.title || updatedTitle}
            <meta
                property="og:title"
                content={openGraph.title || updatedTitle}
            />
        {/if}

        {#if openGraph.description || description}
            <meta
                property="og:description"
                content={openGraph.description || description}
            />
        {/if}

        {#if openGraph.images && openGraph.images.length}
            {#each openGraph.images as image (image)}
                <meta property="og:image" content={image.url} />
                {#if image.alt}
                    <meta property="og:image:alt" content={image.alt} />
                {/if}
                {#if image.width}
                    <meta
                        property="og:image:width"
                        content={image.width.toString()}
                    />
                {/if}
                {#if image.height}
                    <meta
                        property="og:image:height"
                        content={image.height.toString()}
                    />
                {/if}
                {#if image.secureUrl}
                    <meta
                        property="og:image:secure_url"
                        content={image.secureUrl.toString()}
                    />
                {/if}
                {#if image.type}
                    <meta
                        property="og:image:type"
                        content={image.type.toString()}
                    />
                {/if}
            {/each}
        {/if}

        {#if openGraph.videos && openGraph.videos.length}
            {#each openGraph.videos as video (video)}
                <meta property="og:video" content={video.url} />
                {#if video.width}
                    <meta
                        property="og:video:width"
                        content={video.width.toString()}
                    />
                {/if}
                {#if video.height}
                    <meta
                        property="og:video:height"
                        content={video.height.toString()}
                    />
                {/if}
                {#if video.secureUrl}
                    <meta
                        property="og:video:secure_url"
                        content={video.secureUrl.toString()}
                    />
                {/if}
                {#if video.type}
                    <meta
                        property="og:video:type"
                        content={video.type.toString()}
                    />
                {/if}
            {/each}
        {/if}

        {#if openGraph.audio && openGraph.audio.length}
            {#each openGraph.audio as audio (audio)}
                <meta property="og:audio" content={audio.url} />
                {#if audio.secureUrl}
                    <meta
                        property="og:audio:secure_url"
                        content={audio.secureUrl.toString()}
                    />
                {/if}
                {#if audio.type}
                    <meta
                        property="og:audio:type"
                        content={audio.type.toString()}
                    />
                {/if}
            {/each}
        {/if}

        {#if openGraph.locale}
            <meta property="og:locale" content={openGraph.locale} />
        {/if}

        {#if openGraph.siteName}
            <meta property="og:site_name" content={openGraph.siteName} />
        {/if}
    {/if}

    {#if additionalMetaTags && Array.isArray(additionalMetaTags)}
        {#each additionalMetaTags as tag (tag)}
            <meta
                {...tag.httpEquiv
                    ? { ...tag, "http-equiv": tag.httpEquiv }
                    : tag}
            />
        {/each}
    {/if}

    {#if additionalLinkTags?.length}
        {#each additionalLinkTags as tag (tag)}
            <link {...tag} />
        {/each}
    {/if}
</svelte:head>
