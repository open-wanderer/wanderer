/**
 * @swagger
 * /api/v1/capabilities:
 *   get:
 *     summary: Discover supported client capabilities
 *     tags: [Server]
 *     responses:
 *       200:
 *         description: >
 *           Explicit client capabilities. Only a 200 response without assetApi or a 404 means the marker is absent and the multipart photo API applies. Other non-200 responses leave capability support unknown; retry discovery.
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               required: [assetApi]
 *               properties:
 *                 assetApi:
 *                   type: boolean
 *                   enum: [true]
 */
export function GET() {
 return Response.json({assetApi:true});
}
