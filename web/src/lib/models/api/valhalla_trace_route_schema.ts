import { z } from "zod";

const TraceRouteRequestSchema = z.object({
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

export type TraceRouteRequest = z.infer<typeof TraceRouteRequestSchema>;

export type TraceRouteShapePoint = {
  lat: number;
  lon: number;
};

export type TraceRouteResponse = {
  shape: TraceRouteShapePoint[];
};

export { TraceRouteRequestSchema };
