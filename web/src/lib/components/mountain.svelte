<script lang="ts">
  
  import { theme } from "#lib/stores/theme_store.ts";
  import { T } from "@threlte/core";
  import { useDraco, useGltf } from "@threlte/extras";
  import { backInOut } from "svelte/easing";
  import { Tween } from "svelte/motion";
  import { Group, Mesh } from "three";

  import { interactivity } from "@threlte/extras";
    import type { Snippet } from "svelte";
  interactivity();

  export const ref = new Group();
  interface Props {
    rotation?: number;
    fallback?: Snippet;
    errorSlot?: Snippet<[any]>;
    [key: string]: any
  }

  let { rotation = 0, fallback, errorSlot, ...rest }: Props = $props();

  let dog: Mesh | undefined = $state();

  const dracoLoader = useDraco()
  const gltf = useGltf("/models/mountain.glb", { dracoLoader });

  const bark = $state(new Audio("/audio/sophie.mp3"));
  bark.volume = 0.5

  const sunZPosition = new Tween($theme == "light" ? 11.7 : -20, {
    duration: 1000,
    easing: backInOut,
  });
  $effect(() => {
    sunZPosition.set($theme == "light" ? 11.7 : -20);
  });

  const moonZPosition = new Tween($theme == "dark" ? 14 : -20, {
    duration: 1000,
    easing: backInOut,
  });
  $effect(() => {
    moonZPosition.set($theme == "dark" ? 14 : -20);
  });

  const campfireLightIntensity = new Tween($theme == "dark" ? 2 : 0, {
    duration: 500,
  });
  $effect(() => {
    campfireLightIntensity.set($theme == "dark" ? 2 : 0);
  });

  const cabinLightIntensity = new Tween($theme == "dark" ? 4 : 0, {
    duration: 400,
  });
  $effect(() => {
    cabinLightIntensity.set($theme == "dark" ? 4 : 0);
  });

  const sunLightIntensity = new Tween($theme == "light" ? 5 : 0, {
    duration: 500,
  });
  $effect(() => {
    sunLightIntensity.set($theme == "light" ? 5 : 0);
  });

  const moonLightIntensity = new Tween($theme == "dark" ? 0.8 : 0, {
    duration: 500,
  });
  $effect(() => {
    moonLightIntensity.set($theme == "dark" ? 0.8 : 0);
  });
</script>

<T is={ref} dispose={false} {...rest}>
  {#await gltf}
    {@render fallback?.()}
  {:then gltf}
    <T.Group
      position={[3.98, 4.61, 3.15]}
      rotation={[0, -0.83, 0]}
      scale={0.59}
    >
      <T.Group position={[0.35, 0, -0.1]} rotation={[0, 0.19, 0]} scale={77.1}>
        <T.Mesh
          geometry={gltf.nodes.Sign6_1.geometry}
          material={gltf.materials["Dark Wood"]}
        />
        <T.Mesh
          geometry={gltf.nodes.Sign6_2.geometry}
          material={gltf.materials["Light Wood"]}
        />
        <T.Mesh
          geometry={gltf.nodes.Sign6_3.geometry}
          material={gltf.materials.Herbs}
        />
      </T.Group>
      <T.Group
        position={[0.2, 1.3, 13.73]}
        rotation={[Math.PI, -1.43, Math.PI]}
        scale={78.24}
      >
        <T.Mesh
          geometry={gltf.nodes.Sign6_1.geometry}
          material={gltf.materials["Dark Wood"]}
        />
        <T.Mesh
          geometry={gltf.nodes.Sign6_2.geometry}
          material={gltf.materials["Light Wood"]}
        />
        <T.Mesh
          geometry={gltf.nodes.Sign6_3.geometry}
          material={gltf.materials.Herbs}
        />
      </T.Group>
      <T.Group
        position={[0.2, 1.3, 13.73]}
        rotation={[Math.PI, -1.43, Math.PI]}
        scale={78.24}
      >
        <T.Mesh
          geometry={gltf.nodes.Sign6_1.geometry}
          material={gltf.materials["Dark Wood"]}
        />
        <T.Mesh
          geometry={gltf.nodes.Sign6_2.geometry}
          material={gltf.materials["Light Wood"]}
        />
        <T.Mesh
          geometry={gltf.nodes.Sign6_3.geometry}
          material={gltf.materials.Herbs}
        />
      </T.Group>
    </T.Group>
    <T.Group
      position={[15.43, 3.37, 5.09]}
      rotation={[Math.PI, -0.44, Math.PI]}
    >
      <T.Mesh
        geometry={gltf.nodes["Node-Mesh"].geometry}
        material={gltf.materials.mat9}
      />
      <T.Mesh
        geometry={gltf.nodes["Node-Mesh_1"].geometry}
        material={gltf.materials.mat20}
      />
    </T.Group>
    <T.PointLight
      position={[14.37, 3.8, 4.11]}
      color="#C98A76"
      intensity={campfireLightIntensity.current}
    ></T.PointLight>
    <T.Mesh
      geometry={gltf.nodes.campfire.geometry}
      material={gltf.materials.None}
      position={[14.37, 3.26, 4.11]}
      scale={0.28}
    />
    <T.Mesh
      geometry={gltf.nodes.PUSHILIN_Kayak_Circle003.geometry}
      material={gltf.materials.kayak}
      position={[13.84, 2.64, 8.83]}
      scale={0.42}
    />

    <T.Mesh
      geometry={gltf.nodes.Moon.geometry}
      material={gltf.materials.DD9944}
      position={[-23.68, moonZPosition.current, -6.37]}
      rotation={[0, -0.81, 0]}
      scale={0.53}
      ><T.PointLight position={[0.5, 5, 0.5]} color="#a2cbf5" intensity={4}
      ></T.PointLight>
      <T.DirectionalLight
        position={[20, 20, 40]}
        color="#a2cbf5"
        intensity={moonLightIntensity.current}
        castShadow
      ></T.DirectionalLight></T.Mesh
    >
    <T.Group
      position={[-5.61, sunZPosition.current, -26.34]}
      rotation={[0, 0.75, 0]}
      scale={0.11}
    >
      <T.Mesh
        geometry={gltf.nodes.GeoSphere003_1.geometry}
        material={gltf.materials._Sol02___Default}
      />
      <T.Mesh
        geometry={gltf.nodes.GeoSphere003_1_1.geometry}
        material={gltf.materials._Sol01___Default}
      >
        <T.PointLight position={[0, 20, 45]} color="#ffffff" intensity={20}
        ></T.PointLight>
        <T.PointLight position={[-50, 40, 40]} color="#ffffff" intensity={20}
        ></T.PointLight>
        <T.PointLight position={[0, 60, 50]} color="#ffffff" intensity={20}
        ></T.PointLight>
        <T.PointLight position={[50, 40, 40]} color="#ffffff" intensity={20}
        ></T.PointLight>
        <T.DirectionalLight
          position={[20, 40, 20]}
          color="#FCF9D9"
          intensity={sunLightIntensity.current}
          castShadow
        /></T.Mesh
      >
    </T.Group>
    <T.SpotLight
      position={[-4.58, 4.85, 13.62]}
      angle={Math.PI / 5}
      color="#ffffff"
      intensity={campfireLightIntensity.current * 1.5}
      target={dog}
    ></T.SpotLight>
    <T.Mesh
      geometry={gltf.nodes.Hiker.geometry}
      material={gltf.materials["Material.004"]}
      position={[-4.58, 4.6, 13.62]}
      rotation={[-1.53, -0.02, 2.52]}
      scale={1.5}
    />
    <T.Mesh
      geometry={gltf.nodes.Geo_Beagle.geometry}
      material={gltf.materials.lambert2SG}
      position={[-4.27, 5, 12.68]}
      rotation={[-2.94, 0.25, Math.PI]}
      scale={1.5}
      bind:ref={dog}
      onclick={() => bark.play()}
    />
    <T.Mesh
      geometry={gltf.nodes.ground.geometry}
      material={gltf.materials["Color texture"]}
    />
    <T.Mesh
      geometry={gltf.nodes.bridge.geometry}
      material={gltf.materials["Color texture"]}
      position={[0.06, 5.76, 6.04]}
    />
    <T.Mesh
      geometry={gltf.nodes.water.geometry}
      material={gltf.materials["Color texture"]}
      position={[0, -0.08, 0]}
    />
    <T.Mesh
      geometry={gltf.nodes.path.geometry}
      material={gltf.materials["Color texture"]}
      position={[1.41, 5.48, 4.97]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10030.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.4, 2.87, 15.96]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10011.geometry}
      material={gltf.materials["Color texture"]}
      position={[4.68, 4.33, 2.46]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10012.geometry}
      material={gltf.materials["Color texture"]}
      position={[3.55, 5.46, -2.92]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10013.geometry}
      material={gltf.materials["Color texture"]}
      position={[9.51, 3.22, 4.3]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10034.geometry}
      material={gltf.materials["Color texture"]}
      position={[16.77, 2.96, 7.14]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10045.geometry}
      material={gltf.materials["Color texture"]}
      position={[16.86, 3.15, -5.23]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10036.geometry}
      material={gltf.materials["Color texture"]}
      position={[4.45, 4.95, -1.88]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10017.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.45, 3.12, -10.44]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10018.geometry}
      material={gltf.materials["Color texture"]}
      position={[15.47, 2.83, 12]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10039.geometry}
      material={gltf.materials["Color texture"]}
      position={[-12.71, 3.79, 16.97]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100410.geometry}
      material={gltf.materials["Color texture"]}
      position={[16.16, 3.15, -10.68]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100311.geometry}
      material={gltf.materials["Color texture"]}
      position={[4.74, 4.38, 0.15]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100112.geometry}
      material={gltf.materials["Color texture"]}
      position={[15.07, 3.39, -4.98]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100113.geometry}
      material={gltf.materials["Color texture"]}
      position={[14.37, 3.55, -7.1]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100114.geometry}
      material={gltf.materials["Color texture"]}
      position={[1.66, 3.08, 16.7]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100215.geometry}
      material={gltf.materials["Color texture"]}
      position={[-5.65, 3.46, 16.8]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100116.geometry}
      material={gltf.materials["Color texture"]}
      position={[12.95, 3.41, -0.97]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100217.geometry}
      material={gltf.materials["Color texture"]}
      position={[3, 3.35, 14.8]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100318.geometry}
      material={gltf.materials["Color texture"]}
      position={[1.91, 3.36, 14.63]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100219.geometry}
      material={gltf.materials["Color texture"]}
      position={[-6.1, 4.61, 13.18]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100420.geometry}
      material={gltf.materials["Color texture"]}
      position={[10.78, 3.82, -0.66]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100421.geometry}
      material={gltf.materials["Color texture"]}
      position={[9.66, 3.03, 15.08]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100322.geometry}
      material={gltf.materials["Color texture"]}
      position={[16.75, 3.17, -6.47]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100223.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.83, 3.5, -8.86]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100224.geometry}
      material={gltf.materials["Color texture"]}
      position={[16.53, 2.88, 14.09]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100125.geometry}
      material={gltf.materials["Color texture"]}
      position={[3.82, 3.17, 15.72]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100426.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.45, 2.9, 12.72]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100327.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.64, 3.22, -13.76]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100328.geometry}
      material={gltf.materials["Color texture"]}
      position={[12.88, 3.86, -2.52]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100429.geometry}
      material={gltf.materials["Color texture"]}
      position={[14.17, 3.17, -17.21]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100330.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.37, 2.97, 5.36]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100131.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.05, 3.05, -1.78]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100232.geometry}
      material={gltf.materials["Color texture"]}
      position={[-14.28, 4.17, 15.24]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100333.geometry}
      material={gltf.materials["Color texture"]}
      position={[-4.24, 3.47, 16.61]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100134.geometry}
      material={gltf.materials["Color texture"]}
      position={[12.92, 2.93, 14.56]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100235.geometry}
      material={gltf.materials["Color texture"]}
      position={[0.35, 3.14, 16.35]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100436.geometry}
      material={gltf.materials["Color texture"]}
      position={[11.9, 3.1, 2.41]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100437.geometry}
      material={gltf.materials["Color texture"]}
      position={[14.9, 2.9, 15.83]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100438.geometry}
      material={gltf.materials["Color texture"]}
      position={[-1.44, 3.56, 15.21]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100339.geometry}
      material={gltf.materials["Color texture"]}
      position={[-10.26, 3.83, 16.79]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100140.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.38, 3.12, -12.7]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100241.geometry}
      material={gltf.materials["Color texture"]}
      position={[16.25, 3.12, -14.39]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100142.geometry}
      material={gltf.materials["Color texture"]}
      position={[15.46, 3.14, -17.36]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100343.geometry}
      material={gltf.materials["Color texture"]}
      position={[15.41, 3.02, 2.76]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100244.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.78, 3.08, 0.9]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100245.geometry}
      material={gltf.materials["Color texture"]}
      position={[10.86, 3.08, 8.7]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100246.geometry}
      material={gltf.materials["Color texture"]}
      position={[5.86, 5.14, -2.97]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100347.geometry}
      material={gltf.materials["Color texture"]}
      position={[5.22, 5.37, -4.18]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100248.geometry}
      material={gltf.materials["Color texture"]}
      position={[7.39, 3.62, 1.64]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100249.geometry}
      material={gltf.materials["Color texture"]}
      position={[5.86, 5.67, -5.15]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100250.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.55, 2.99, 6.92]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100151.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.96, 2.81, 13.48]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100152.geometry}
      material={gltf.materials["Color texture"]}
      position={[10.58, 2.96, 15.6]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100453.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.33, 2.86, 17.45]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100254.geometry}
      material={gltf.materials["Color texture"]}
      position={[-6.15, 8.16, -3.25]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100155.geometry}
      material={gltf.materials["Color texture"]}
      position={[2.56, 3.14, 15.84]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100356.geometry}
      material={gltf.materials["Color texture"]}
      position={[8.6, 3.36, 9.53]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100157.geometry}
      material={gltf.materials["Color texture"]}
      position={[5.4, 3.1, 16.55]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100258.geometry}
      material={gltf.materials["Color texture"]}
      position={[10.51, 3.19, 2.55]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100159.geometry}
      material={gltf.materials["Color texture"]}
      position={[3.37, 5.64, -4.24]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100360.geometry}
      material={gltf.materials["Color texture"]}
      position={[11.3, 3.37, -17.5]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100161.geometry}
      material={gltf.materials["Color texture"]}
      position={[8.89, 3.31, 3.16]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100362.geometry}
      material={gltf.materials["Color texture"]}
      position={[6.8, 3.01, 17.55]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100263.geometry}
      material={gltf.materials["Color texture"]}
      position={[2.8, 4.06, 10.67]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100164.geometry}
      material={gltf.materials["Color texture"]}
      position={[-2.56, 4.71, 11.74]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100365.geometry}
      material={gltf.materials["Color texture"]}
      position={[-1.83, 8.28, -4.63]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100366.geometry}
      material={gltf.materials["Color texture"]}
      position={[-16.74, 3.3, 17.04]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100467.geometry}
      material={gltf.materials["Color texture"]}
      position={[0.45, 5.65, 3.61]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100268.geometry}
      material={gltf.materials["Color texture"]}
      position={[-0.49, 4.01, 13.29]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100369.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.24, 3.04, -0.52]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100370.geometry}
      material={gltf.materials["Color texture"]}
      position={[16.62, 2.99, 3.64]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100371.geometry}
      material={gltf.materials["Color texture"]}
      position={[-1.28, 7.92, -2.47]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100472.geometry}
      material={gltf.materials["Color texture"]}
      position={[-5.23, 5.09, 10.83]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100473.geometry}
      material={gltf.materials["Color texture"]}
      position={[-0.74, 6.99, 0.69]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100274.geometry}
      material={gltf.materials["Color texture"]}
      position={[12.28, 2.9, 8.59]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100475.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.14, 3.07, 2.87]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100276.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.71, 3.12, -0.16]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100377.geometry}
      material={gltf.materials["Color texture"]}
      position={[12.64, 3.13, 0.76]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100378.geometry}
      material={gltf.materials["Color texture"]}
      position={[14.1, 3.04, 3.15]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100279.geometry}
      material={gltf.materials["Color texture"]}
      position={[14.99, 3.06, 0.59]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100180.geometry}
      material={gltf.materials["Color texture"]}
      position={[14.35, 3, 5.89]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100481.geometry}
      material={gltf.materials["Color texture"]}
      position={[1.48, 4.34, 10.28]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100182.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.4, 2.93, 9.52]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100283.geometry}
      material={gltf.materials["Color texture"]}
      position={[-2.56, 7.97, -3.12]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100184.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.72, 3.32, -10.94]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100485.geometry}
      material={gltf.materials["Color texture"]}
      position={[17.26, 3.12, -15.43]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100186.geometry}
      material={gltf.materials["Color texture"]}
      position={[15.79, 3.03, 1.32]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10020.geometry}
      material={gltf.materials["Color texture"]}
      position={[1.44, 5.77, 1.71]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir1001004.geometry}
      material={gltf.materials["Color texture"]}
      position={[1.43, 6.77, -4.55]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10022.geometry}
      material={gltf.materials["Color texture"]}
      position={[-4.05, 5.75, 8.02]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10023.geometry}
      material={gltf.materials["Color texture"]}
      position={[-8.23, 7.58, 0.81]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir1003001.geometry}
      material={gltf.materials["Color texture"]}
      position={[-4.45, 6.58, 3.08]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir1004001.geometry}
      material={gltf.materials["Color texture"]}
      position={[3.81, 4.9, 6.19]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10016.geometry}
      material={gltf.materials["Color texture"]}
      position={[7.02, 3.78, 6.78]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10037.geometry}
      material={gltf.materials["Color texture"]}
      position={[4.37, 3.77, 12.16]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir1001005.geometry}
      material={gltf.materials["Color texture"]}
      position={[-10.67, 8.67, 10.89]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir10019.geometry}
      material={gltf.materials["Color texture"]}
      position={[-12.43, 9.19, 8.42]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100310.geometry}
      material={gltf.materials["Color texture"]}
      position={[-1.81, 8.56, -6.94]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100111.geometry}
      material={gltf.materials["Color texture"]}
      position={[-2.23, 8.4, -11.75]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100212.geometry}
      material={gltf.materials["Color texture"]}
      position={[1.43, 8.8, -10.28]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir1001010.geometry}
      material={gltf.materials["Color texture"]}
      position={[3.08, 8.52, -9.16]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100214.geometry}
      material={gltf.materials["Color texture"]}
      position={[8.33, 6.92, -7.79]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100415.geometry}
      material={gltf.materials["Color texture"]}
      position={[8.19, 6.47, -5.82]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100316.geometry}
      material={gltf.materials["Color texture"]}
      position={[11.42, 4.99, -6.31]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100117.geometry}
      material={gltf.materials["Color texture"]}
      position={[8.33, 4.77, -1.69]}
    />
    <T.Mesh
      geometry={gltf.nodes.Fir100418.geometry}
      material={gltf.materials["Color texture"]}
      position={[13.01, 4.2, -4.84]}
    />
    <T.PointLight
      position={[-8.6, 9.3, -1]}
      color="#FCF9D9"
      intensity={cabinLightIntensity.current}
    ></T.PointLight>
    <T.Mesh
      geometry={gltf.nodes.Cabin.geometry}
      material={gltf.materials.lambert3SG}
      position={[-9.4, 8.1, -2.43]}
      rotation={[0, 0.54, 0]}
      scale={0.003}
    />
    <T.Mesh
      geometry={gltf.nodes.Door_1.geometry}
      material={gltf.materials.lambert3SG}
      position={[-9.4, 8.1, -2.43]}
      rotation={[0, 0.54, 0]}
      scale={0.003}
    />
    <T.Mesh
      geometry={gltf.nodes.Door_2.geometry}
      material={gltf.materials.lambert3SG}
      position={[-9.4, 8.1, -2.43]}
      rotation={[0, 0.54, 0]}
      scale={0.003}
    />
    <T.Group position={[7.32, 13.07, -12.47]} rotation={[0, rotation, 0]}>
      <T.Mesh
        geometry={gltf.nodes["Node-Mesh001"].geometry}
        material={gltf.materials.mat8}
      />
      <T.Mesh
        geometry={gltf.nodes["Node-Mesh001_1"].geometry}
        material={gltf.materials.mat23}
      >
        <T.PointLight
          position={[-1, 0.8, 0]}
          color="#ffffff"
          intensity={campfireLightIntensity.current}
        ></T.PointLight>
        <T.PointLight
          position={[1, 0.8, 0]}
          color="#ffffff"
          intensity={campfireLightIntensity.current}
        ></T.PointLight></T.Mesh
      >
    </T.Group>
    <T.Group
      position={[-13.32, 18.54, -9.77]}
      rotation={[0, rotation + 0.5, 0]}
    >
      <T.Mesh
        geometry={gltf.nodes["Node-Mesh001"].geometry}
        material={gltf.materials.mat8}
      />
      <T.Mesh
        geometry={gltf.nodes["Node-Mesh001_1"].geometry}
        material={gltf.materials.mat23}
      >
        <T.PointLight
          position={[-1, 0.8, 0]}
          color="#ffffff"
          intensity={campfireLightIntensity.current}
        ></T.PointLight>
        <T.PointLight
          position={[1, 0.8, 0]}
          color="#ffffff"
          intensity={campfireLightIntensity.current}
        ></T.PointLight></T.Mesh
      >
    </T.Group>
    <T.Group position={[-9.02, 14.74, 6.22]} rotation={[0, rotation - 0.5, 0]}>
      <T.Mesh
        geometry={gltf.nodes["Node-Mesh001"].geometry}
        material={gltf.materials.mat8}
      />
      <T.Mesh
        geometry={gltf.nodes["Node-Mesh001_1"].geometry}
        material={gltf.materials.mat23}
      >
        <T.PointLight
          position={[-1, 0.8, 0]}
          color="#ffffff"
          intensity={campfireLightIntensity.current}
        ></T.PointLight>
        <T.PointLight
          position={[1, 0.8, 0]}
          color="#ffffff"
          intensity={campfireLightIntensity.current}
        ></T.PointLight></T.Mesh
      >
    </T.Group>
  {:catch error}
    {@render errorSlot?.({ error, })}
  {/await}
</T>
