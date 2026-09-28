package main

import (
	"encoding/json"
	"testing"
)

func TestTourImportDifficulty(t *testing.T) {
	for _, test := range []struct {
		name  string
		grade string
		want  string
	}{
		{name: "easy", grade: "easy", want: "easy"},
		{name: "moderate", grade: "moderate", want: "moderate"},
		{name: "difficult", grade: "difficult", want: "difficult"},
		{name: "missing"},
		{name: "unrecognized", grade: "extreme"},
		{name: "technical grade", grade: "T3"},
		{name: "display label", grade: "hard"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tour := &detailedTour{
				ID:         123,
				Name:       "Test tour",
				Difficulty: difficulty{Grade: test.grade},
				Embedded: detailedTourEmbedded{
					Coordinates: coordinates{Items: []coordinate{{Lat: 47, Lng: 8}}},
				},
			}
			item, err := tourImport(tour, nil)
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(detailOutput{Item: item})
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				Item map[string]json.RawMessage `json:"item"`
			}
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			if test.want == "" {
				if value, exists := wire.Item["difficulty"]; exists {
					t.Fatalf("unknown difficulty must be omitted, got %s", value)
				}
			} else if string(wire.Item["difficulty"]) != `"`+test.want+`"` {
				t.Fatalf("wire difficulty = %s, want %q", wire.Item["difficulty"], test.want)
			}
			var metadata map[string]any
			if err := json.Unmarshal(wire.Item["metadata"], &metadata); err != nil {
				t.Fatal(err)
			}
			if metadata["difficulty"] != test.grade {
				t.Fatalf("raw grade = %v, want %q", metadata["difficulty"], test.grade)
			}
		})
	}
}

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
	if points[0].Photos[0].Filename != "komoot-waypoint-48446190.jpg" {
		t.Fatalf("unexpected waypoint photo filename: %q", points[0].Photos[0].Filename)
	}
}

func TestTourPhotoFilenamesIdentifySource(t *testing.T) {
	t.Run("cover with id", func(t *testing.T) {
		tour := &detailedTour{Embedded: detailedTourEmbedded{CoverImages: coverImages{Embedded: imagesEmbedded{Items: []imageItem{{
			ID:  flexibleID("123"),
			Src: "https://example.test/cover.jpg",
		}}}}}}
		got := photos(tour, nil)
		if len(got) != 1 || got[0].Filename != "komoot-cover-123.jpg" {
			t.Fatalf("unexpected cover photos: %#v", got)
		}
	})

	t.Run("cover without id", func(t *testing.T) {
		tour := &detailedTour{Embedded: detailedTourEmbedded{CoverImages: coverImages{Embedded: imagesEmbedded{Items: []imageItem{{
			Src: "https://example.test/cover.jpg",
		}}}}}}
		got := photos(tour, nil)
		if len(got) != 1 || got[0].Filename != "komoot-cover.jpg" {
			t.Fatalf("unexpected cover photos: %#v", got)
		}
	})

	t.Run("map image", func(t *testing.T) {
		tour := &detailedTour{MapImage: mapImage{Src: "https://example.test/map.jpg"}}
		got := photos(tour, nil)
		if len(got) != 1 || got[0].Filename != "komoot-map.jpg" {
			t.Fatalf("unexpected map photos: %#v", got)
		}
	})
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
