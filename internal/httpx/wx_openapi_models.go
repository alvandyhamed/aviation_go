package httpx

import (
	"time"
)

// ---------- METAR ----------

type MetarCloud struct {
	Cover string `json:"cover"` // e.g. "FEW"
	Base  int    `json:"base"`  // e.g. 25000 (ft AGL)
}

type MetarDTO struct {
	ICAOId      string       `json:"icaoId"`      // e.g. "KJFK"
	ReceiptTime time.Time    `json:"receiptTime"` // ISO8601
	ObsTime     int64        `json:"obsTime"`     // epoch seconds
	ReportTime  time.Time    `json:"reportTime"`  // ISO8601
	Temp        float64      `json:"temp"`        // °C
	Dewp        float64      `json:"dewp"`        // °C
	WDir        int          `json:"wdir"`        // degrees
	WSpd        int          `json:"wspd"`        // kt
	Visib       string       `json:"visib"`       // e.g. "10+"
	Altim       float64      `json:"altim"`       // hPa (per sample)
	SLP         float64      `json:"slp"`         // hPa
	QCField     int          `json:"qcField"`
	MetarType   string       `json:"metarType"` // "METAR"/"SPECI"
	RawOb       string       `json:"rawOb"`     // raw METAR text
	Lat         float64      `json:"lat"`
	Lon         float64      `json:"lon"`
	Elev        int          `json:"elev"`            // meters (per sample)
	Name        string       `json:"name"`            // station full name
	Cover       string       `json:"cover,omitempty"` // summary cloud cover if present
	Clouds      []MetarCloud `json:"clouds,omitempty"`
	FltCat      string       `json:"fltCat,omitempty"`   // VFR/IFR/MVFR/LIFR
	PresTend    *float64     `json:"presTend,omitempty"` // optional in some reports
	MaxT        *float64     `json:"maxT,omitempty"`
	MinT        *float64     `json:"minT,omitempty"`
}

// ---------- TAF ----------

// NOTE: wdir در fcsts می‌تواند عدد یا "VRB" باشد.
// برای Swagger آن را به صورت string مستند می‌کنیم تا سازگار بماند.
type TafCloud struct {
	Cover string  `json:"cover"`          // "FEW","SCT","BKN","OVC"
	Base  int     `json:"base"`           // ft AGL
	Type  *string `json:"type,omitempty"` // optional
}

type TafForecast struct {
	TimeFrom    int64   `json:"timeFrom"`             // epoch seconds
	TimeTo      int64   `json:"timeTo"`               // epoch seconds
	TimeBec     *int64  `json:"timeBec,omitempty"`    // epoch seconds (nullable)
	FcstChange  *string `json:"fcstChange,omitempty"` // e.g. "FM","BECMG","PROB30"
	Probability *int    `json:"probability,omitempty"`
	// swagger:ignore
	WDirRaw any `json:"wdir,omitempty"` // واقعی upstream می‌تواند int یا "VRB" باشد
	// برای داکیومنتیشن به‌صورت رشته نشان بده:
	WDirDoc    string     `json:"-" swaggertype:"string" example:"VRB"` // فقط برای schema
	WSpd       *int       `json:"wspd,omitempty"`
	WGst       *int       `json:"wgst,omitempty"`
	WShearHgt  *int       `json:"wshearHgt,omitempty"`
	WShearDir  *int       `json:"wshearDir,omitempty"`
	WShearSpd  *int       `json:"wshearSpd,omitempty"`
	Visib      *string    `json:"visib,omitempty"` // e.g. "6+"
	Altim      *float64   `json:"altim,omitempty"`
	VertVis    *int       `json:"vertVis,omitempty"`
	WxString   *string    `json:"wxString,omitempty"`
	NotDecoded *string    `json:"notDecoded,omitempty"`
	Clouds     []TafCloud `json:"clouds,omitempty"`
	// icgTurb, temp: آرایه‌های خالی/اختیاری
	ICGTurb []any `json:"icgTurb,omitempty"`
	Temp    []any `json:"temp,omitempty"`
}

type TafDTO struct {
	ICAOId        string        `json:"icaoId"`
	DBPopTime     time.Time     `json:"dbPopTime"`     // ISO8601
	BulletinTime  time.Time     `json:"bulletinTime"`  // ISO8601
	IssueTime     time.Time     `json:"issueTime"`     // ISO8601
	ValidTimeFrom int64         `json:"validTimeFrom"` // epoch seconds
	ValidTimeTo   int64         `json:"validTimeTo"`   // epoch seconds
	RawTAF        string        `json:"rawTAF"`
	MostRecent    int           `json:"mostRecent"`
	Remarks       string        `json:"remarks"`
	Lat           float64       `json:"lat"`
	Lon           float64       `json:"lon"`
	Elev          int           `json:"elev"`
	Prior         int           `json:"prior"`
	Name          string        `json:"name"`
	FCSTS         []TafForecast `json:"fcsts"`
}
