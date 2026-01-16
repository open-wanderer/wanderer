package strava

import "time"

type TokenRequest struct {
	ClientID     int32  `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
	GrantType    string `json:"grant_type"`
}

type RefreshTokenRequest struct {
	ClientID     int32  `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	GrantType    string `json:"grant_type"`
}
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}
type StravaIntegration struct {
	Active       bool   `json:"active"`
	Routes       bool   `json:"routes"`
	Activities   bool   `json:"activities"`
	ClientID     int32  `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"`
	Privacy      string `json:"privacy"`
	After        string `json:"after,omitempty"`
}
type StravaRoute struct {
	Athlete             Athlete     `json:"athlete"`
	Description         string      `json:"description"`
	Distance            float32     `json:"distance"`
	ElevationGain       float32     `json:"elevation_gain"`
	ID                  int64       `json:"id"`
	IDStr               string      `json:"id_str"`
	Map                 Map         `json:"map"`
	Name                string      `json:"name"`
	Private             bool        `json:"private"`
	Starred             bool        `json:"starred"`
	Timestamp           int         `json:"timestamp"`
	Type                int         `json:"type"`
	SubType             int         `json:"sub_type"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
	EstimatedMovingTime int         `json:"estimated_moving_time"`
	Segments            []Segments  `json:"segments"`
	Waypoints           []Waypoints `json:"waypoints"`
}

type Athlete struct {
	ID            int64     `json:"id"`
	ResourceState int       `json:"resource_state"`
	Firstname     string    `json:"firstname"`
	Lastname      string    `json:"lastname"`
	ProfileMedium string    `json:"profile_medium"`
	Profile       string    `json:"profile"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	Country       string    `json:"country"`
	Sex           string    `json:"sex"`
	Premium       bool      `json:"premium"`
	Summit        bool      `json:"summit"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Map struct {
	ID              string `json:"id"`
	Polyline        string `json:"polyline"`
	SummaryPolyline string `json:"summary_polyline"`
}

type AthletePrEffort struct {
	PrActivityID  int64     `json:"pr_activity_id"`
	PrElapsedTime int       `json:"pr_elapsed_time"`
	PrDate        time.Time `json:"pr_date"`
	EffortCount   int       `json:"effort_count"`
}

type AthleteSegmentStats struct {
	ID             int       `json:"id"`
	ActivityID     int       `json:"activity_id"`
	ElapsedTime    int       `json:"elapsed_time"`
	StartDate      time.Time `json:"start_date"`
	StartDateLocal time.Time `json:"start_date_local"`
	Distance       float32   `json:"distance"`
	IsKom          bool      `json:"is_kom"`
}

type Segments struct {
	ID                  int64               `json:"id"`
	Name                string              `json:"name"`
	ActivityType        string              `json:"activity_type"`
	Distance            float32             `json:"distance"`
	AverageGrade        float32             `json:"average_grade"`
	MaximumGrade        float32             `json:"maximum_grade"`
	ElevationHigh       float32             `json:"elevation_high"`
	ElevationLow        float32             `json:"elevation_low"`
	StartLatlng         []float32           `json:"start_latlng"`
	EndLatlng           []float32           `json:"end_latlng"`
	ClimbCategory       int                 `json:"climb_category"`
	City                string              `json:"city"`
	State               string              `json:"state"`
	Country             string              `json:"country"`
	Private             bool                `json:"private"`
	AthletePrEffort     AthletePrEffort     `json:"athlete_pr_effort"`
	AthleteSegmentStats AthleteSegmentStats `json:"athlete_segment_stats"`
}

type Waypoints struct {
	Latlng            []float32 `json:"latlng"`
	TargetLatlng      []float32 `json:"target_latlng"`
	Categories        []string  `json:"categories"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	DistanceIntoRoute float64   `json:"distance_into_route"`
}

type StravaActivity struct {
	ResourceState        int       `json:"resource_state"`
	Athlete              Athlete   `json:"athlete"`
	Name                 string    `json:"name"`
	Distance             float64   `json:"distance"`
	MovingTime           int       `json:"moving_time"`
	ElapsedTime          int       `json:"elapsed_time"`
	TotalElevationGain   float64   `json:"total_elevation_gain"`
	Type                 string    `json:"type"`
	SportType            string    `json:"sport_type"`
	WorkoutType          any       `json:"workout_type"`
	ID                   int64     `json:"id"`
	ExternalID           string    `json:"external_id"`
	UploadID             int64     `json:"upload_id"`
	StartDate            time.Time `json:"start_date"`
	StartDateLocal       time.Time `json:"start_date_local"`
	Timezone             string    `json:"timezone"`
	StartLatlng          any       `json:"start_latlng"`
	EndLatlng            any       `json:"end_latlng"`
	LocationCity         any       `json:"location_city"`
	LocationState        any       `json:"location_state"`
	LocationCountry      string    `json:"location_country"`
	AchievementCount     int       `json:"achievement_count"`
	KudosCount           int       `json:"kudos_count"`
	CommentCount         int       `json:"comment_count"`
	AthleteCount         int       `json:"athlete_count"`
	PhotoCount           int       `json:"photo_count"`
	Map                  Map       `json:"map"`
	Trainer              bool      `json:"trainer"`
	Commute              bool      `json:"commute"`
	Manual               bool      `json:"manual"`
	Private              bool      `json:"private"`
	Flagged              bool      `json:"flagged"`
	GearID               string    `json:"gear_id"`
	FromAcceptedTag      bool      `json:"from_accepted_tag"`
	AverageSpeed         float64   `json:"average_speed"`
	MaxSpeed             float64   `json:"max_speed"`
	AverageCadence       float64   `json:"average_cadence"`
	AverageWatts         float64   `json:"average_watts"`
	WeightedAverageWatts int       `json:"weighted_average_watts"`
	Kilojoules           float64   `json:"kilojoules"`
	DeviceWatts          bool      `json:"device_watts"`
	HasHeartrate         bool      `json:"has_heartrate"`
	AverageHeartrate     float64   `json:"average_heartrate"`
	MaxHeartrate         float64   `json:"max_heartrate"`
	MaxWatts             int       `json:"max_watts"`
	PrCount              int       `json:"pr_count"`
	TotalPhotoCount      int       `json:"total_photo_count"`
	HasKudoed            bool      `json:"has_kudoed"`
	SufferScore          float64   `json:"suffer_score"`
}

type DetailedStravaActivity struct {
	ID                       int64                 `json:"id"`
	ResourceState            int                   `json:"resource_state"`
	ExternalID               string                `json:"external_id"`
	UploadID                 int64                 `json:"upload_id"`
	Athlete                  Athlete               `json:"athlete"`
	Name                     string                `json:"name"`
	Distance                 float64               `json:"distance"`
	MovingTime               int                   `json:"moving_time"`
	ElapsedTime              int                   `json:"elapsed_time"`
	TotalElevationGain       float64               `json:"total_elevation_gain"`
	Type                     string                `json:"type"`
	SportType                string                `json:"sport_type"`
	StartDate                time.Time             `json:"start_date"`
	StartDateLocal           time.Time             `json:"start_date_local"`
	Timezone                 string                `json:"timezone"`
	StartLatlng              []float64             `json:"start_latlng"`
	EndLatlng                []float64             `json:"end_latlng"`
	AchievementCount         int                   `json:"achievement_count"`
	KudosCount               int                   `json:"kudos_count"`
	CommentCount             int                   `json:"comment_count"`
	AthleteCount             int                   `json:"athlete_count"`
	PhotoCount               int                   `json:"photo_count"`
	Map                      Map                   `json:"map"`
	Trainer                  bool                  `json:"trainer"`
	Commute                  bool                  `json:"commute"`
	Manual                   bool                  `json:"manual"`
	Private                  bool                  `json:"private"`
	Flagged                  bool                  `json:"flagged"`
	GearID                   string                `json:"gear_id"`
	FromAcceptedTag          bool                  `json:"from_accepted_tag"`
	AverageSpeed             float64               `json:"average_speed"`
	MaxSpeed                 float64               `json:"max_speed"`
	AverageCadence           float64               `json:"average_cadence"`
	AverageTemp              int                   `json:"average_temp"`
	AverageWatts             float64               `json:"average_watts"`
	WeightedAverageWatts     int                   `json:"weighted_average_watts"`
	Kilojoules               float64               `json:"kilojoules"`
	DeviceWatts              bool                  `json:"device_watts"`
	HasHeartrate             bool                  `json:"has_heartrate"`
	MaxWatts                 int                   `json:"max_watts"`
	ElevHigh                 float64               `json:"elev_high"`
	ElevLow                  float64               `json:"elev_low"`
	PrCount                  int                   `json:"pr_count"`
	TotalPhotoCount          int                   `json:"total_photo_count"`
	HasKudoed                bool                  `json:"has_kudoed"`
	WorkoutType              int                   `json:"workout_type"`
	SufferScore              float64               `json:"suffer_score"`
	Description              string                `json:"description"`
	Calories                 float64               `json:"calories"`
	SegmentEfforts           []SegmentEfforts      `json:"segment_efforts"`
	SplitsMetric             []SplitsMetric        `json:"splits_metric"`
	Laps                     []Laps                `json:"laps"`
	Gear                     Gear                  `json:"gear"`
	PartnerBrandTag          any                   `json:"partner_brand_tag"`
	Photos                   Photos                `json:"photos"`
	HighlightedKudosers      []HighlightedKudosers `json:"highlighted_kudosers"`
	HideFromHome             bool                  `json:"hide_from_home"`
	DeviceName               string                `json:"device_name"`
	EmbedToken               string                `json:"embed_token"`
	SegmentLeaderboardOptOut bool                  `json:"segment_leaderboard_opt_out"`
	LeaderboardOptOut        bool                  `json:"leaderboard_opt_out"`
}

type SegmentActivity struct {
	ID            int64 `json:"id"`
	ResourceState int   `json:"resource_state"`
}

type Segment struct {
	ID            int64     `json:"id"`
	ResourceState int       `json:"resource_state"`
	Name          string    `json:"name"`
	ActivityType  string    `json:"activity_type"`
	Distance      float64   `json:"distance"`
	AverageGrade  float64   `json:"average_grade"`
	MaximumGrade  float64   `json:"maximum_grade"`
	ElevationHigh float64   `json:"elevation_high"`
	ElevationLow  float64   `json:"elevation_low"`
	StartLatlng   []float64 `json:"start_latlng"`
	EndLatlng     []float64 `json:"end_latlng"`
	ClimbCategory int       `json:"climb_category"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	Country       string    `json:"country"`
	Private       bool      `json:"private"`
	Hazardous     bool      `json:"hazardous"`
	Starred       bool      `json:"starred"`
}

type SegmentEfforts struct {
	ID             int64           `json:"id"`
	ResourceState  int             `json:"resource_state"`
	Name           string          `json:"name"`
	Activity       SegmentActivity `json:"activity"`
	Athlete        Athlete         `json:"athlete"`
	ElapsedTime    int             `json:"elapsed_time"`
	MovingTime     int             `json:"moving_time"`
	StartDate      time.Time       `json:"start_date"`
	StartDateLocal time.Time       `json:"start_date_local"`
	Distance       float64         `json:"distance"`
	StartIndex     int             `json:"start_index"`
	EndIndex       int             `json:"end_index"`
	AverageCadence float64         `json:"average_cadence"`
	DeviceWatts    bool            `json:"device_watts"`
	AverageWatts   float64         `json:"average_watts"`
	Segment        Segment         `json:"segment"`
	KomRank        any             `json:"kom_rank"`
	PrRank         any             `json:"pr_rank"`
	Achievements   []any           `json:"achievements"`
	Hidden         bool            `json:"hidden"`
}

type SplitsMetric struct {
	Distance            float64 `json:"distance"`
	ElapsedTime         int     `json:"elapsed_time"`
	ElevationDifference float64 `json:"elevation_difference"`
	MovingTime          int     `json:"moving_time"`
	Split               int     `json:"split"`
	AverageSpeed        float64 `json:"average_speed"`
	PaceZone            int     `json:"pace_zone"`
}

type Laps struct {
	ID                 int64           `json:"id"`
	ResourceState      int             `json:"resource_state"`
	Name               string          `json:"name"`
	Activity           SegmentActivity `json:"activity"`
	Athlete            Athlete         `json:"athlete"`
	ElapsedTime        int             `json:"elapsed_time"`
	MovingTime         int             `json:"moving_time"`
	StartDate          time.Time       `json:"start_date"`
	StartDateLocal     time.Time       `json:"start_date_local"`
	Distance           float64         `json:"distance"`
	StartIndex         int             `json:"start_index"`
	EndIndex           int             `json:"end_index"`
	TotalElevationGain float64         `json:"total_elevation_gain"`
	AverageSpeed       float64         `json:"average_speed"`
	MaxSpeed           float64         `json:"max_speed"`
	AverageCadence     float64         `json:"average_cadence"`
	DeviceWatts        bool            `json:"device_watts"`
	AverageWatts       float64         `json:"average_watts"`
	LapIndex           int             `json:"lap_index"`
	Split              int             `json:"split"`
}

type Gear struct {
	ID            string `json:"id"`
	Primary       bool   `json:"primary"`
	Name          string `json:"name"`
	ResourceState int    `json:"resource_state"`
	Distance      int    `json:"distance"`
}

type Urls struct {
	Num100 string `json:"100"`
	Num600 string `json:"600"`
}

type Primary struct {
	ID       any    `json:"id"`
	UniqueID string `json:"unique_id"`
	Urls     Urls   `json:"urls"`
	Source   int    `json:"source"`
}

type Photos struct {
	Primary         Primary `json:"primary"`
	UsePrimaryPhoto bool    `json:"use_primary_photo"`
	Count           int     `json:"count"`
}

type HighlightedKudosers struct {
	DestinationURL string `json:"destination_url"`
	DisplayName    string `json:"display_name"`
	AvatarURL      string `json:"avatar_url"`
	ShowName       bool   `json:"show_name"`
}

type ActivityStreamResponse struct {
	LatLng   LatLngStream   `json:"latlng"`
	Altitude AltitudeStream `json:"altitude"`
	Time     TimeStream     `json:"time"`
}

type ActivityStream struct {
	OriginalSize int    `json:"original_size"`
	Resolution   string `json:"resolution"`
	SeriesType   string `json:"series_type"`
}

type TimeStream struct {
	ActivityStream
	Data []int `json:"data"`
}

type LatLngStream struct {
	ActivityStream
	Data [][]float64 `json:"data"`
}

type AltitudeStream struct {
	ActivityStream
	Data []float64 `json:"data"`
}
