<script lang="ts">
    import { theme } from "$lib/stores/theme_store";
    import { T, useTask } from "@threlte/core";
    import { onMount } from "svelte";
    import { Tween } from "svelte/motion";
    import * as THREE from "three";
    import Mountain from "./mountain.svelte";

    let rotation = $state(0);
    useTask((delta) => {
        rotation += delta / 2;
    });

    const ambientIntensity = new Tween($theme == "light" ? 3 : 0, {
        duration: 500,
    });

    let ambientColor = $derived($theme == "light" ? "#ffffff" : "#4c7fe6");
    $effect(() => {
        ambientIntensity.set($theme == "light" ? 3 : 0.1);
    });

    let starsGeometry: THREE.BufferGeometry | undefined = $state();
    let starsMaterial: THREE.PointsMaterial | undefined = $state();
    onMount(() => {
        const stars = new Array(0);
        for (var i = 0; i < 500; i++) {
            let x = THREE.MathUtils.randFloatSpread(500);
            let y = THREE.MathUtils.randFloatSpread(500);
            let z = THREE.MathUtils.randFloatSpread(500);
            stars.push(x, y, z);
        }
        starsGeometry = new THREE.BufferGeometry();
        starsGeometry.setAttribute(
            "position",
            new THREE.Float32BufferAttribute(stars, 3),
        );
        starsMaterial = new THREE.PointsMaterial({ color: 0xffffff });
    });
</script>

<T.PerspectiveCamera
    makeDefault
    position={[35.5, 28, 35.5]}
    oncreate={(c) => {
        c.lookAt(-17, 0, 12);
    }}
/>

{#if starsGeometry && starsMaterial}
    <T.Points geometry={starsGeometry} material={starsMaterial}></T.Points>
{/if}

<!-- <Earth {rotation}></Earth> -->
<Mountain {rotation}></Mountain>

<T.HemisphereLight
    skyColor={ambientColor}
    groundColor="#242734"
    intensity={ambientIntensity.current}
/>
