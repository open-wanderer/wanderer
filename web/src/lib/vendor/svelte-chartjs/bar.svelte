<script lang="ts">
  import {
    BarController,
    BarElement,
    Chart,
    LineController,
    LineElement,
    PointElement,
    Tooltip,
    type ChartData,
    type ChartOptions,
  } from 'chart.js';
  import type { HTMLCanvasAttributes } from 'svelte/elements';


  interface Props extends HTMLCanvasAttributes {
    data: ChartData<'bar' | 'line', number[], string>;
    options: ChartOptions<'bar' | 'line'>;
  }

  const { data, options, ...rest }: Props = $props();

  Chart.register(
    Tooltip,
    BarController,
    BarElement,
    LineController,
    LineElement,
    PointElement,
  );

  let canvasElem: HTMLCanvasElement;
  let chart: Chart<'bar' | 'line', number[], string>;

  $effect(() => {
    chart = new Chart(canvasElem, {
      type: 'bar',
      data,
      options,
    });

    return () => {
      chart.destroy();
    };
  });

  $effect(() => {
    if (chart) {
      chart.data = data;
      chart.update();
    }
  });
</script>

<canvas bind:this={canvasElem} {...rest}></canvas>
