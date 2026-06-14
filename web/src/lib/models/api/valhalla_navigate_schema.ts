import { z } from "zod";

const NavigateRequestSchema = z.object({
  shape: z
    .array(
      z.object({
        lat: z.number().min(-90).max(90),
        lon: z.number().min(-180).max(180),
      })
    )
    .min(2, "at_least_two_shape_points")
    .max(500),
  costing: z.enum(["pedestrian", "bicycle"]).default("pedestrian"),
});

type NavigateRequest = z.infer<typeof NavigateRequestSchema>;

type NavigateShapePoint = {
  lat: number;
  lon: number;
};

type NavigateManeuver = {
  instruction: string;
  length: number;
  begin_shape_index: number;
  bearing: number;
  type: number;
};

type NavigateResponse = {
  maneuvers: NavigateManeuver[];
  shape: [number, number][];
};

export { NavigateRequestSchema };
export type { NavigateRequest, NavigateShapePoint, NavigateManeuver, NavigateResponse };
