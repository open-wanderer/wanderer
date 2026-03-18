package hammerhead

import (
	"time"
)

type HammerheadToursResponse struct {
	TotalItems  int                      `json:"totalItems"`
	TotalPages  int                      `json:"totalPages"`
	PerPage     int                      `json:"perPage"`
	CurrentPage int                      `json:"currentPage"`
	Data        []HammerheadTourResponse `json:"data"`
}
type HammerheadTourResponse struct {
	StartLocationName string    `json:"startLocationName"`
	IsAutoImported    bool      `json:"isAutoImported"`
	SummaryPolyline   string    `json:"summaryPolyline"`
	IsStarred         bool      `json:"isStarred"`
	IsPublic          bool      `json:"isPublic"`
	Collections       any       `json:"collections"`
	Gain              int       `json:"gain"`
	Distance          float64   `json:"distance"`
	Name              string    `json:"name"`
	RoutingType       string    `json:"routingType"`
	ID                string    `json:"id"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	Source            string    `json:"source"`
}

type HammerheadTourElevation struct {
	Gain     float64 `json:"gain"`
	Loss     float64 `json:"loss"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Source   string  `json:"source"`
	Polyline string  `json:"polyline"`
}
type HammerheadLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
type HammerheadWaypoint struct {
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	WaypointType  string  `json:"waypointType"`
	PolylineIndex int     `json:"polylineIndex"`
}

type HammerheadTour struct {
	ID                string                  `json:"id"`
	CreatedAt         time.Time               `json:"createdAt"`
	Name              string                  `json:"name"`
	Distance          float64                 `json:"distance"`
	Elevation         HammerheadTourElevation `json:"elevation"`
	IsStarred         bool                    `json:"isStarred"`
	StartLocationName string                  `json:"startLocationName"`
	EndLocationName   string                  `json:"endLocationName"`
	StartLocation     HammerheadLocation      `json:"startLocation"`
	EndLocation       HammerheadLocation      `json:"endLocation"`
	Waypoints         []HammerheadWaypoint    `json:"waypoints"`
	Collections       []string                `json:"collections"`
	RoutePolyline     string                  `json:"routePolyline"`
	SummaryPolyline   string                  `json:"summaryPolyline"`
	Source            string                  `json:"source"`
	SourceID          string                  `json:"sourceId"`
	IsPublic          bool                    `json:"isPublic"`
	ImageVersion      string                  `json:"imageVersion"`
	IsAutoImported    bool                    `json:"isAutoImported"`
	UpdatedAt         time.Time               `json:"updatedAt"`
	Bounds            []HammerheadLocation    `json:"bounds"`
}

type HammerheadIntegration struct {
	Active    bool   `json:"active"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Planned   bool   `json:"planned"`
	Completed bool   `json:"completed"`
	After     string `json:"after,omitempty"`
}

type LoginResponse struct {
	Token   string `json:"access_token"`
	Type    string `json:"token_type"`
	Expires int    `json:"expires_in"`
}

type HammerheadActivitiesResponse struct {
	Items   int                          `json:"totalItems"`
	Pages   int                          `json:"totalPages"`
	PerPage int                          `json:"perPage"`
	Tours   []HammerheadActivityResponse `json:"data"`
}

type HammerheadActivityResponse struct {
	ID           string                 `json:"id"`
	CreatedAt    time.Time              `json:"createdAt"`
	Name         string                 `json:"name"`
	Client       string                 `json:"client"`
	ActiveTime   int                    `json:"activeTime"`
	Duration     HammerheadTourDuration `json:"duration"`
	Sync         HammerheadSync         `json:"partners"`
	ActivityInfo []HammerheadInfo       `json:"activityInfo"`
}
type HammerheadInfoValue struct {
	Format string  `json:"format"`
	Value  float64 `json:"value"`
}
type HammerheadInfo struct {
	Key   string              `json:"key"`
	Value HammerheadInfoValue `json:"value"`
}
type HammerheadPartner struct {
	Partner     string    `json:"partner"`
	NeedsUpload bool      `json:"needsUpload"`
	ExternalID  string    `json:"externalId"`
	Attempts    int       `json:"attempts"`
	UploadedAt  time.Time `json:"uploadedAt"`
}
type HammerheadSync struct {
	Description string              `json:"description"`
	Tags        []any               `json:"tags"`
	Synced      bool                `json:"synced"`
	Partners    []HammerheadPartner `json:"partners"`
}
type HammerheadTourDuration struct {
	ElapsedTime int       `json:"elapsedTime"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
}

type HammerheadActivity struct {
	ActivityData      HammerheadActivityData      `json:"activityData"`
	SessionData       HammerheadSessionData       `json:"sessionData"`
	RecordData        HammerheadRecordData        `json:"recordData"`
	ShiftData         HammerheadShiftData         `json:"shiftData"`
	LapData           HammerheadLapData           `json:"lapData"`
	DeviceBatteryData HammerheadDeviceBatteryData `json:"deviceBatteryData"`
}
type HammerheadDuration struct {
	ElapsedTime int       `json:"elapsedTime"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
}
type HammerheadLapDetail struct {
	ActiveTime int                  `json:"activeTime"`
	Duration   HammerheadDuration   `json:"duration"`
	LapNumber  int                  `json:"lapNumber"`
	Pauses     []HammerheadDuration `json:"pauses"`
	LapInfo    []HammerheadInfo     `json:"lapInfo"`
	Trigger    string               `json:"trigger"`
}
type HammerheadActivityData struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	BikeID       string                `json:"bikeId"`
	Client       string                `json:"client"`
	ActiveTime   int                   `json:"activeTime"`
	Duration     HammerheadDuration    `json:"duration"`
	ActivityInfo []HammerheadInfo      `json:"activityInfo"`
	Laps         []HammerheadLapDetail `json:"laps"`
	Polyline     string                `json:"polyline"`
	Sync         HammerheadSync        `json:"sync"`
	ActivityType string                `json:"activityType"`
	Climbs       []HammerheadClimb     `json:"climbs"`
	CreatedAt    time.Time             `json:"createdAt"`
	UpdatedAt    time.Time             `json:"updatedAt"`
}
type HammerheadClimb struct {
	StartDistance float64 `json:"startDistance"`
	EndDistance   float64 `json:"endDistance"`
	Distance      float64 `json:"distance"`
}
type HammerheadSessionData struct {
	ThresholdPower int   `json:"thresholdPower"`
	FrontGears     []int `json:"frontGears"`
	RearGears      []int `json:"rearGears"`
}
type HammerheadRecordData struct {
	Distance    []float64 `json:"distance"`
	Timestamp   []int     `json:"timestamp"`
	Elevation   []float64 `json:"elevation"`
	Grade       []float64 `json:"grade"`
	Lat         []float64 `json:"lat"`
	Lng         []float64 `json:"lng"`
	Speed       []float64 `json:"speed"`
	Power       []any     `json:"power"`
	Temperature []int     `json:"temperature"`
}
type HammerheadShiftData struct {
	Timestamp    []int  `json:"timestamp"`
	FrontChange  []bool `json:"frontChange"`
	FrontGear    []int  `json:"frontGear"`
	RearGear     []int  `json:"rearGear"`
	FrontGearNum []int  `json:"frontGearNum"`
	RearGearNum  []int  `json:"rearGearNum"`
}
type HammerheadLapData struct {
	Timestamp []int    `json:"timestamp"`
	Trigger   []string `json:"trigger"`
}
type HammerheadDeviceBatteryData struct {
	Timestamp     []int `json:"timestamp"`
	DeviceBattery []int `json:"deviceBattery"`
}
