import { assetResponseJson as json } from "#lib/server/asset_response_json.ts";
import { RecordListOptionsSchema } from '#lib/models/api/base_schema.ts';
import { TrailCreateSchema } from '#lib/models/api/trail_schema.ts';
import type { Trail } from '#lib/models/trail.ts';
import { withTrailPreferencePocketBaseFilter } from '#lib/server/category_preference_filter.ts';
import { completeTrailAssetExpands } from '#lib/server/trail_asset_pagination.ts';
import { completeOwnerExportChildren } from '#lib/server/trail_export_response.ts';
import { enrichTrailListResponse, enrichTrailResponse, withTrailAssetExpands } from '#lib/server/trail_response_util.ts';
import { Collection, create, handleError } from '#lib/util/api_util.ts';
import { type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail:
 *   get:
 *     summary: List trails
 *     description: Retrieves a paginated list of trails with optional filtering and sorting
 *     tags:
 *       - Trails
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *       - in: query
 *         name: sort
 *         schema:
 *           type: string
 *       - in: query
 *         name: filter
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *       - in: query
 *         name: export
 *         description: Export the authenticated owner's trails, including hidden discovery categories. Uses ascending ID order and up to 100 records per page.
 *         schema:
 *           type: string
 *           enum: [owned]
 *       - in: query
 *         name: photos
 *         description: Set false to omit photo asset expansion, for example for a metadata-only account export.
 *         schema:
 *           type: boolean
 *           default: true
 *     responses:
 *       200:
 *         description: ListResult<Trail>
 *       401:
 *         description: Authentication required for an owned export
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const searchParams = Object.fromEntries(event.url.searchParams);
        const safeSearchParams = RecordListOptionsSchema.parse(searchParams);
        const { perPage, page, ...opts } = safeSearchParams;
        const ownerExport = searchParams.export === "owned";
        if (ownerExport && !event.locals.user?.actor) return json({ message: "Authentication and a local actor are required" }, { status: 401 });
        const filter = ownerExport ? event.locals.pb.filter("author = {:actor}", { actor: event.locals.user!.actor }) : await withTrailPreferencePocketBaseFilter(
            event,
            safeSearchParams.filter,
        );
        const includePhotos = searchParams.photos !== "false";
        if (ownerExport) {
            opts.sort = "id";
            opts.expand = "author,category,subcategory,subcategory.category,waypoints_via_trail,summit_logs_via_trail,tags";
        }
        const listOptions = includePhotos ? withTrailAssetExpands({ ...opts, filter }) : { ...opts, filter };
        const r = ownerExport ? await event.locals.pb.collection(Collection.trails).getList<Trail>(Math.max(1, page ?? 1), Math.max(1, Math.min(100, perPage ?? 100)), listOptions) : (perPage ?? 0) < 0
            ? {
                  items: await event.locals.pb
                      .collection(Collection.trails)
                      .getFullList<Trail>(listOptions),
                  perPage: -1,
                  page: 1,
                  totalItems: 0,
                  totalPages: 1,
              }
            : await event.locals.pb
                  .collection(Collection.trails)
                  .getList<Trail>(page, perPage, listOptions);

        if (!ownerExport && perPage && perPage < 0) {
            r.totalItems = r.items.length;
        }

        if (ownerExport) await completeOwnerExportChildren(event, r.items);
        if (includePhotos) await completeTrailAssetExpands(event, r.items);
        return json(enrichTrailListResponse(r))
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/trail:
 *   put:
 *     summary: Create trail
 *     tags:
 *       - Trails
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TrailCreateInput'
 *     responses:
 *       201:
 *         description: Trail created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Invalid input or `trail_photos_form_field_removed` when photo modifier keys are present. A plain `photos` key is ignored. Save photos through PUT /api/v1/assets.
 *       500:
 *         description: Internal Server Error
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
export async function PUT(event: RequestEvent) {
    try {        
        const r = await create<Trail>(event, TrailCreateSchema, Collection.trails)
        enrichRecord(r);
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}

function enrichRecord(r: Trail) {
    enrichTrailResponse(r);
}
