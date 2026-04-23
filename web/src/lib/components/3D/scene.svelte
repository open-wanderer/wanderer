<script lang="ts">
    import { theme } from "$lib/stores/theme_store";
    import { T, useTask } from "@threlte/core";
    import { backInOut } from "svelte/easing";
    import { Tween } from "svelte/motion";
    import Marker from "./marker.svelte";
    import Moon from "./moon.svelte";
    import Sun from "./sun.svelte";
    import { useViewport } from "@threlte/extras";
    import { innerHeight, innerWidth } from "svelte/reactivity/window";

    const viewport = useViewport();

    const canvasHeight = $derived((innerHeight.current ?? 0) - 112);
    const canvasWidth = $derived((innerWidth.current ?? 0) * 0.6);
    const dioramaImageMaxHeight = 0.75;

    const imgWidth = $derived(
        Math.min(
            ((dioramaImageMaxHeight * canvasHeight) / 9) * 16,
            canvasWidth,
        ),
    );
    const imgHeight = $derived((imgWidth / 16) * 9);

    const imgWorldWidth = $derived((imgWidth / canvasWidth) * $viewport.width);
    const imgWorldHeight = $derived(
        (imgHeight / canvasHeight) * $viewport.height,
    );

    const peak1PositionRelativeToImage = [-0.195, 0.7];
    const peak1Position = $derived([
        peak1PositionRelativeToImage[0] * (imgWorldWidth / 2),
        peak1PositionRelativeToImage[1] * (imgWorldHeight / 2),
        20,
    ]);

    const peak2PositionRelativeToImage = [0.05, 1];
    const peak2Position = $derived([
        peak2PositionRelativeToImage[0] * (imgWorldWidth / 2),
        peak2PositionRelativeToImage[1] * (imgWorldHeight / 2),
        20,
    ]);

    const peak3PositionRelativeToImage = [0.55, 0.37];
    const peak3Position = $derived([
        peak3PositionRelativeToImage[0] * (imgWorldWidth / 2),
        peak3PositionRelativeToImage[1] * (imgWorldHeight / 2),
        20,
    ]);

    const sunUpPosition = $derived(($viewport.height / 2) * 0.6);

    let rotation = $state(0);
    useTask((delta) => {
        rotation += delta / 2;
    });

    const sunZPosition = new Tween($theme == "light" ? sunUpPosition : 0, {
        duration: 1000,
        easing: backInOut,
    });
    $effect(() => {
        sunZPosition.set($theme == "light" ? sunUpPosition : 0);
    });

    const sunLightIntensity = new Tween($theme == "light" ? 4 : 0, {
        duration: 400,
    });
    $effect(() => {
        sunLightIntensity.set($theme == "light" ? 4 : 0);
    });

    const moonZPosition = new Tween($theme == "dark" ? sunUpPosition : 0, {
        duration: 1000,
        easing: backInOut,
    });
    $effect(() => {
        moonZPosition.set($theme == "dark" ? sunUpPosition : 0);
    });
</script>

<T.OrthographicCamera
    makeDefault
    zoom={14}
    position={[0, 0, 30]}
    oncreate={(c) => {
        c.lookAt(0, 0, 0);
    }}
/>

<!-- <Earth {rotation}></Earth> -->

<!-- <Mountain {rotation}></Mountain> -->

<Sun
    position={[7.5, sunZPosition.current, 0]}
    rotation={[0, 0, 0]}
    scale={0.11}
    lightIntesity={sunLightIntensity.current}
/>
<Moon
    position={[-5, moonZPosition.current, 0]}
    rotation={[0, -Math.PI / 2, 0]}
    scale={0.53}
></Moon>

<Marker position={peak1Position} rotation={[0, rotation, 0]} scale={1}></Marker>

<Marker position={peak2Position} rotation={[0, rotation - 1, 0]} scale={1}
></Marker>

<Marker
    position={peak3Position}
    rotation={[0, rotation + 1, 0]}
    scale={1}
></Marker>
