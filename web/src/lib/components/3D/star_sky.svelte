<script lang="ts">
    import starLayers from "./star_postions.json";
</script>

<div class="stars-wrapper">
    {#each starLayers as layer, i}
        <svg
            class="stars layer-{i}"
            width="100%"
            height="100%"
            preserveAspectRatio="none"
        >
            {#each layer as star}
                <circle cx="{star.x}%" cy="{star.y}%" r={star.r} fill="white" />
            {/each}
        </svg>
    {/each}

    <svg class="extras" width="100%" height="100%" preserveAspectRatio="none">
        <defs>
            <radialGradient id="comet-gradient" cx="0" cy=".5" r="0.5">
                <stop offset="0%" stop-color="rgba(255,255,255,.8)" />
                <stop offset="100%" stop-color="rgba(255,255,255,0)" />
            </radialGradient>
        </defs>

        <g transform="rotate(-135)">
            <ellipse
                class="comet comet-a"
                fill="url(#comet-gradient)"
                cx="0"
                cy="0"
                rx="150"
                ry="2"
            />
        </g>
        <g transform="rotate(20)">
            <ellipse
                class="comet comet-b"
                fill="url(#comet-gradient)"
                cx="100%"
                cy="0"
                rx="150"
                ry="2"
            />
        </g>
        <g transform="rotate(300)">
            <ellipse
                class="comet comet-c"
                fill="url(#comet-gradient)"
                cx="40%"
                cy="100%"
                rx="150"
                ry="2"
            />
        </g>
    </svg>
</div>

<style>
    .stars-wrapper {
        position: absolute;
        inset: 0;
        pointer-events: none;
        z-index: 0;
        transition:
            opacity 1.5s ease-in-out,
            visibility 1.5s;
        will-change: opacity;
    }

    .visible {
        opacity: 1;
        visibility: visible;
    }

    .hidden {
        opacity: 0;
        visibility: hidden;
    }

    .stars {
        position: absolute;
        inset: 0;
        /* Force GPU to handle the transparency fade */
        transform: translateZ(0);
    }

    /* Your working twinkle animation from before */
    .layer-0 {
        animation: twinkle 4s infinite -0s;
    }
    .layer-1 {
        animation: twinkle 4s infinite -1.3s;
    }
    .layer-2 {
        animation: twinkle 4s infinite -2.6s;
    }

    @keyframes twinkle {
        0%,
        100% {
            opacity: 1;
        }
        50% {
            opacity: 0.3;
        }
    }

    .comet {
        animation: comet 10s linear infinite;
    }

    @keyframes comet {
        0% {
            transform: translateX(0);
            opacity: 0;
        }
        1% {
            opacity: 1;
        }
        20% {
            transform: translateX(-2000px);
            opacity: 0;
        }
        100% {
            transform: translateX(-2000px);
            opacity: 0;
        }
    }

    .comet-b {
        animation-delay: -3.3s;
    }

    .comet-c {
        animation-delay: -5s;
    }
</style>
