/// A stable, index-independent identity for a `RouteSegment`, derived from
/// its bounding anchor ids rather than array position — indices drift under
/// insert/delete/reorder (RESEARCH.md Pitfall 3), but an anchor-id pair does
/// not. Both the routing engine's in-flight/generation race-guard maps
/// (`route_anchor_provider.dart`) and 19-03's GeoJSON feature properties key
/// off this value.
String segmentKey(String beforeAnchorId, String afterAnchorId) =>
    '${beforeAnchorId}_$afterAnchorId';
