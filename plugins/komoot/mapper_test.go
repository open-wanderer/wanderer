package main

import "testing"

func TestWaypointsFromEmbeddedWayPoints(t *testing.T) {
	tour := &detailedTour{
		Embedded: detailedTourEmbedded{
			WayPoints: timeline{
				Embedded: timelineEmbedded{
					Items: []timelineItem{{
						Embedded: timelineItemEmbedded{
							Reference: waypointReference{
								ID:   flexibleID("2355158"),
								Name: "Ruedertaler Hofglace Rastplatz",
								Location: point{
									Lat: 47.280262,
									Lng: 8.046906,
									Alt: 476.7,
								},
								StartPoint: point{
									Lat: 47.280262,
									Lng: 8.046906,
									Alt: 476.7,
								},
							},
						},
					}},
				},
			},
		},
	}

	points := waypoints(tour)
	if len(points) != 1 {
		t.Fatalf("expected 1 waypoint, got %d", len(points))
	}
	if points[0].ExternalID != "2355158" || points[0].Name != "Ruedertaler Hofglace Rastplatz" {
		t.Fatalf("unexpected waypoint identity: %#v", points[0])
	}
	if points[0].Lat != 47.280262 || points[0].Lon != 8.046906 || points[0].Ele == nil || *points[0].Ele != 476.7 {
		t.Fatalf("unexpected waypoint coordinates: %#v", points[0])
	}
}

func TestWaypointsDeduplicateWayPointsAndTimeline(t *testing.T) {
	item := timelineItem{
		Embedded: timelineItemEmbedded{
			Reference: waypointReference{
				ID:         flexibleID("8277503"),
				Name:       "Aarebruecke bei Aarburg",
				StartPoint: point{Lat: 47.320204, Lng: 7.897589},
			},
		},
	}
	tour := &detailedTour{
		Embedded: detailedTourEmbedded{
			WayPoints: timeline{Embedded: timelineEmbedded{Items: []timelineItem{item}}},
			Timeline:  timeline{Embedded: timelineEmbedded{Items: []timelineItem{item}}},
		},
	}

	points := waypoints(tour)
	if len(points) != 1 {
		t.Fatalf("expected duplicate waypoint to be collapsed, got %d", len(points))
	}
}

func TestWaypointsIncludeFrontImage(t *testing.T) {
	tour := &detailedTour{
		Embedded: detailedTourEmbedded{
			WayPoints: timeline{
				Embedded: timelineEmbedded{
					Items: []timelineItem{{
						Embedded: timelineItemEmbedded{
							Reference: waypointReference{
								ID:         flexibleID("4266004"),
								Name:       "Blick auf die Solothurner Altstadt und die St.-Ursen-Kathedrale",
								StartPoint: point{Lat: 47.205925, Lng: 7.535326, Alt: 424.6},
								Embedded: waypointSubEmbedded{
									FrontImage: imageItem{
										ID:   flexibleID("48446190"),
										Src:  "https://example.test/image.jpg",
										Type: "image/*",
									},
								},
							},
						},
					}},
				},
			},
		},
	}

	points := waypoints(tour)
	if len(points) != 1 {
		t.Fatalf("expected 1 waypoint, got %d", len(points))
	}
	if len(points[0].Photos) != 1 {
		t.Fatalf("expected 1 waypoint photo, got %d", len(points[0].Photos))
	}
	if points[0].Photos[0].ExternalID != "48446190" || points[0].Photos[0].Source.URL != "https://example.test/image.jpg" {
		t.Fatalf("unexpected waypoint photo: %#v", points[0].Photos[0])
	}
}

func TestWaypointPhotosDeduplicateFrontImage(t *testing.T) {
	item := timelineItem{
		Embedded: timelineItemEmbedded{
			Reference: waypointReference{
				Embedded: waypointSubEmbedded{
					FrontImage: imageItem{
						ID:   flexibleID("48446190"),
						Src:  "https://example.test/front.jpg",
						Type: "image/*",
					},
					Images: coverImages{
						Embedded: imagesEmbedded{
							Items: []imageItem{{
								ID:   flexibleID("48446190"),
								Src:  "https://example.test/front.jpg",
								Type: "image/*",
							}},
						},
					},
				},
			},
		},
	}

	photos := waypointPhotos(item)
	if len(photos) != 1 {
		t.Fatalf("expected duplicate front image to be collapsed, got %d", len(photos))
	}
}
