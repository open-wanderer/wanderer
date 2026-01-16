package komoot

import "time"

type KomootIntegration struct {
	Active    bool   `json:"active"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Planned   bool   `json:"planned"`
	Completed bool   `json:"completed"`
	Privacy   string `json:"privacy"`
}

type LoginResponse struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	User     User   `json:"user"`
	Username string `json:"username"`
}

type Content struct {
	HasImage bool `json:"hasImage"`
}

type Fitness struct {
	Personalised bool `json:"personalised"`
}

type User struct {
	Content      Content `json:"content"`
	CreatedAt    string  `json:"createdAt"`
	Displayname  string  `json:"displayname"`
	Fitness      Fitness `json:"fitness"`
	ImageURL     string  `json:"imageUrl"`
	Locale       string  `json:"locale"`
	Metric       bool    `json:"metric"`
	Newsletter   bool    `json:"newsletter"`
	State        string  `json:"state"`
	Username     string  `json:"username"`
	WelcomeMails bool    `json:"welcomeMails"`
}

type KomootToursResponse struct {
	Embedded Embedded      `json:"_embedded"`
	Links    ResponseLinks `json:"_links"`
	Page     Page          `json:"page"`
}
type StartPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	Alt float64 `json:"alt"`
}
type Surfaces struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}
type WayTypes struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}
type Summary struct {
	Surfaces []Surfaces `json:"surfaces"`
	WayTypes []WayTypes `json:"way_types"`
}
type Difficulty struct {
	Grade                string `json:"grade"`
	ExplanationTechnical string `json:"explanation_technical"`
	ExplanationFitness   string `json:"explanation_fitness"`
}
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
type Path struct {
	Location    Location `json:"location"`
	Index       int      `json:"index"`
	Reference   string   `json:"reference,omitempty"`
	EndIndex    int      `json:"end_index,omitempty"`
	SegmentType string   `json:"segment_type,omitempty"`
}
type Segments struct {
	Type string `json:"type"`
	From int    `json:"from"`
	To   int    `json:"to"`
}
type MapImage struct {
	Src         string `json:"src"`
	Templated   bool   `json:"templated"`
	Type        string `json:"type"`
	Attribution string `json:"attribution"`
}
type MapImagePreview struct {
	Src         string `json:"src"`
	Templated   bool   `json:"templated"`
	Type        string `json:"type"`
	Attribution string `json:"attribution"`
}
type VectorMapImage struct {
	Src         string `json:"src"`
	Templated   bool   `json:"templated"`
	Type        string `json:"type"`
	Attribution string `json:"attribution"`
}
type VectorMapImagePreview struct {
	Src         string `json:"src"`
	Templated   bool   `json:"templated"`
	Type        string `json:"type"`
	Attribution string `json:"attribution"`
}

type Relation struct {
	Href      string `json:"href"`
	Templated bool   `json:"templated"`
}
type CreatorLinks struct {
	Relation Relation `json:"relation"`
}

type LinksEmbedded struct {
	Creator Creator `json:"creator"`
}
type LinksCreator struct {
	Href string `json:"href"`
}
type LinksCoordinates struct {
	Href string `json:"href"`
}
type LinksTourLine struct {
	Href string `json:"href"`
}
type LinksParticipants struct {
	Href string `json:"href"`
}
type LinksWayTypes struct {
	Href string `json:"href"`
}
type LinksSurfaces struct {
	Href string `json:"href"`
}
type LinksDirections struct {
	Href string `json:"href"`
}
type LinksTimeline struct {
	Href string `json:"href"`
}
type LinksTranslations struct {
	Href string `json:"href"`
}
type LinksCoverImages struct {
	Href string `json:"href"`
}
type LinksTourRating struct {
	Href string `json:"href"`
}
type TourLinks struct {
	Creator      LinksCreator      `json:"creator"`
	Coordinates  LinksCoordinates  `json:"coordinates"`
	TourLine     LinksTourLine     `json:"tour_line"`
	Participants LinksParticipants `json:"participants"`
	WayTypes     LinksWayTypes     `json:"way_types"`
	Surfaces     LinksSurfaces     `json:"surfaces"`
	Directions   LinksDirections   `json:"directions"`
	Timeline     LinksTimeline     `json:"timeline"`
	Translations LinksTranslations `json:"translations"`
	CoverImages  LinksCoverImages  `json:"cover_images"`
	TourRating   LinksTourRating   `json:"tour_rating"`
}
type KomootTour struct {
	ID                    int                   `json:"id"`
	Type                  string                `json:"type"`
	Name                  string                `json:"name"`
	Source                string                `json:"source"`
	RoutingVersion        string                `json:"routing_version"`
	Status                string                `json:"status"`
	Date                  time.Time             `json:"date"`
	KcalActive            int                   `json:"kcal_active"`
	KcalResting           int                   `json:"kcal_resting"`
	StartPoint            StartPoint            `json:"start_point"`
	Distance              float64               `json:"distance"`
	Duration              int                   `json:"duration"`
	ElevationUp           float64               `json:"elevation_up"`
	ElevationDown         float64               `json:"elevation_down"`
	Sport                 string                `json:"sport"`
	Query                 string                `json:"query"`
	Constitution          int                   `json:"constitution"`
	Summary               Summary               `json:"summary"`
	Difficulty            Difficulty            `json:"difficulty"`
	TourInformation       []any                 `json:"tour_information"`
	Path                  []Path                `json:"path"`
	Segments              []Segments            `json:"segments"`
	ChangedAt             time.Time             `json:"changed_at"`
	MapImage              MapImage              `json:"map_image"`
	MapImagePreview       MapImagePreview       `json:"map_image_preview"`
	VectorMapImage        VectorMapImage        `json:"vector_map_image"`
	VectorMapImagePreview VectorMapImagePreview `json:"vector_map_image_preview"`
	PotentialRouteUpdate  bool                  `json:"potential_route_update"`
	Embedded              Embedded              `json:"_embedded"`
	Links                 TourLinks             `json:"_links"`
}
type Embedded struct {
	Tours []KomootTour `json:"tours"`
}
type Next struct {
	Href string `json:"href"`
}
type ResponseLinks struct {
	Next Next `json:"next"`
}
type Page struct {
	Size          int `json:"size"`
	TotalElements int `json:"totalElements"`
	TotalPages    int `json:"totalPages"`
	Number        int `json:"number"`
}

type DetailedKomootTour struct {
	ID            int                  `json:"id"`
	Type          string               `json:"type"`
	Name          string               `json:"name"`
	Status        string               `json:"status"`
	Date          time.Time            `json:"date"`
	KcalActive    float64              `json:"kcal_active"`
	KcalResting   float64              `json:"kcal_resting"`
	StartPoint    StartPoint           `json:"start_point"`
	Distance      float64              `json:"distance"`
	Duration      int                  `json:"duration"`
	ElevationUp   float64              `json:"elevation_up"`
	ElevationDown float64              `json:"elevation_down"`
	Sport         string               `json:"sport"`
	MapImage      MapImage             `json:"map_image"`
	Difficulty    Difficulty           `json:"difficulty"`
	ChangedAt     time.Time            `json:"changed_at"`
	Embedded      DetailedTourEmbedded `json:"_embedded"`
}

type Items struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	Alt float64 `json:"alt"`
	T   int     `json:"t"`
}

type Coordinates struct {
	Items []Items `json:"items"`
}

type DetailedTourEmbedded struct {
	Coordinates Coordinates `json:"coordinates"`
	Timeline    Timeline    `json:"timeline"`
	CoverImages CoverImages `json:"cover_images"`
}

type CoverImages struct {
	Embedded CoverImagesEmbedded `json:"_embedded"`
	Links    Links               `json:"_links"`
	Page     Page                `json:"page"`
}

type CoverImagesEmbedded struct {
	Items []ImageItem `json:"items"`
}

type Timeline struct {
	Embedded TimelineEmbedded `json:"_embedded"`
	Links    Links            `json:"_links"`
	Page     Page             `json:"page"`
}

type TimelineEmbedded struct {
	Items []Item `json:"items"`
}

type Item struct {
	Index    int                  `json:"index"`
	Cover    int                  `json:"cover"`
	Type     string               `json:"type"`
	Embedded TimelineItemEmbedded `json:"_embedded"`
}

type TimelineItemEmbedded struct {
	Reference Reference `json:"reference"`
}

type Reference struct {
	ID            int         `json:"id"`
	Type          string      `json:"type"`
	BaseName      string      `json:"base_name"`
	Name          string      `json:"name"`
	CreatedAt     time.Time   `json:"created_at"`
	ChangedAt     time.Time   `json:"changed_at"`
	Sport         string      `json:"sport"`
	Routable      bool        `json:"routable"`
	StartPoint    Point       `json:"start_point"`
	MidPoint      Point       `json:"mid_point"`
	EndPoint      Point       `json:"end_point"`
	Distance      float64     `json:"distance"`
	ElevationUp   float64     `json:"elevation_up"`
	ElevationDown float64     `json:"elevation_down"`
	Score         float64     `json:"score"`
	WikiPOIID     string      `json:"wiki_poi_id"`
	PoorQuality   bool        `json:"poor_quality"`
	Categories    []string    `json:"categories"`
	Flagged       bool        `json:"flagged"`
	Links         Links       `json:"_links"`
	Embedded      SubEmbedded `json:"_embedded"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	Alt float64 `json:"alt"`
}

type Links struct {
	Self Link `json:"self"`
}

type Link struct {
	Href      string `json:"href"`
	Templated bool   `json:"templated,omitempty"`
}

type SubEmbedded struct {
	Creator Creator `json:"creator"`
	Images  Images  `json:"images"`
	Tips    Tips    `json:"tips"`
}

type Creator struct {
	Username    string `json:"username"`
	Avatar      Avatar `json:"avatar"`
	Status      string `json:"status"`
	Links       Links  `json:"_links"`
	DisplayName string `json:"display_name"`
	IsPremium   bool   `json:"is_premium"`
}

type Avatar struct {
	Src       string `json:"src"`
	Templated bool   `json:"templated"`
	Type      string `json:"type"`
}

type Images struct {
	Embedded ImagesEmbedded `json:"_embedded"`
	Links    Links          `json:"_links"`
	Page     Page           `json:"page"`
}

type ImagesEmbedded struct {
	Items []ImageItem `json:"items"`
}

type ImageItem struct {
	ID          int         `json:"id"`
	Src         string      `json:"src"`
	Rating      Rating      `json:"rating"`
	Templated   bool        `json:"templated"`
	HighlightID int         `json:"highlight_id"`
	ClientHash  string      `json:"client_hash,omitempty"`
	Location    Location    `json:"location"`
	Type        string      `json:"type"`
	Links       Links       `json:"_links"`
	Embedded    SubEmbedded `json:"_embedded"`
}

type Rating struct {
	Up   int `json:"up"`
	Down int `json:"down"`
}

type Tips struct {
	Embedded TipsEmbedded `json:"_embedded"`
	Links    Links        `json:"_links"`
	Page     Page         `json:"page"`
}

type TipsEmbedded struct {
	Items []TipItem `json:"items"`
}

type TipItem struct {
	ID                     int         `json:"id"`
	Text                   string      `json:"text"`
	Rating                 Rating      `json:"rating"`
	CreatedAt              time.Time   `json:"created_at"`
	TextLanguage           string      `json:"text_language"`
	TranslatedText         string      `json:"translated_text"`
	TranslatedTextLanguage string      `json:"translated_text_language"`
	Attribution            string      `json:"attribution"`
	HighlightID            int         `json:"highlight_id"`
	Links                  Links       `json:"_links"`
	Embedded               SubEmbedded `json:"_embedded"`
}
