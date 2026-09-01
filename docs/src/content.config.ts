import { defineCollection } from 'astro:content';
import { docsLoader } from "@astrojs/starlight/loaders";
import { docsSchema } from "@astrojs/starlight/schema";
import { z } from 'astro/zod';

const specMetadataSchema = z.object({
	id: z.string(),
	kind: z.enum([
		'overview',
		'shared',
		'capability',
		'delivery',
		'work-item',
		'contract',
		'decision',
		'evidence',
	]),
	status: z.enum(['stub', 'draft', 'reviewable', 'accepted', 'superseded']),
	deliveryStatus: z
		.enum(['candidate', 'blocked', 'ready', 'in-progress', 'validating', 'released', 'withdrawn'])
		.optional(),
	capability: z.enum(['FOUNDATION', 'DERIVED', 'CONTEXT', 'DIALOG']).optional(),
	productSlice: z.string().optional(),
	exposure: z.enum(['internal', 'shadow', 'operational', 'user-visible', 'cutover']).optional(),
	implementationDependsOn: z.array(z.string()).default([]),
	implementationConditions: z.array(z.string()).default([]),
	releaseGates: z.array(z.string()).default([]),
	normativeSources: z.array(z.string()).default([]),
	implementationPrs: z.array(z.string()).default([]),
	trackingIssue: z.string().optional(),
	lastReviewed: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
}).strict();

export const collections = {
	docs: defineCollection({
		loader: docsLoader(),
		schema: docsSchema({
			extend: z.object({
				spec: specMetadataSchema.optional(),
			}),
		}),
	}),
};
