package libPI

// #include <stdint.h>
import "C"
import (
	"log/slog"
	"sync"
	"time"
)

type PITIMESTAMP struct {
	Month  C.int
	Year   C.int
	Day    C.int
	Hour   C.int
	Minute C.int
	Tzinfo C.int
	Second C.double
}

func NewPITIMESTAMP(dt time.Time) PITIMESTAMP {
	return PITIMESTAMP{
		Month:  C.int(dt.Month()),
		Year:   C.int(dt.Year()),
		Day:    C.int(dt.Day()),
		Hour:   C.int(dt.Hour()),
		Minute: C.int(dt.Minute()),
		Second: C.double(dt.Second()) + C.double(dt.Nanosecond())/1e9,
		Tzinfo: C.int(0), // Timezone is 0 in datalogs
	}
}

type FloatSnapshot struct {
	Ptid int32
	Val  float64
	Dt   time.Time
}

// PointType represents the different point types
type PointType int

const (
	PointTypeUnknown PointType = iota
	PointTypeReal
	PointTypeInteger
	PointTypeDigital
)

// String provides a string representation of the PointType
func (pt PointType) String() string {
	switch pt {
	case PointTypeReal:
		return "Real"
	case PointTypeInteger:
		return "Integer"
	case PointTypeDigital:
		return "Digital"
	default:
		return "Unknown"
	}
}

type PointCache struct {
	DatalogName string
	DataLogID   int
	DataLogType int
	Process     bool
	PIName      *string
	PIId        *int32
	PiType      *PointType
}

// Add a method to print out the contents of the PointCache
func (pc *PointCache) Print() {
	slog.Info("PointCache details",
		"DatalogName", pc.DatalogName,
		"DataLogID", pc.DataLogID,
		"DataLogType", pc.DataLogType,
		"Process", pc.Process,
		"PIName", *pc.PIName,
		"PIId", pc.PIId,
		"PiType", pc.PiType,
	)
}

type PointLookup struct {
	mu     sync.RWMutex
	points map[int]*PointCache
}

// PrintAll method to print all PointCache instances in the map
func (pl *PointLookup) PrintAll() {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	for _, point := range pl.points {
		point.Print()
	}
}

func NewPointLookup() *PointLookup {
	return &PointLookup{
		points: make(map[int]*PointCache),
	}
}

// AddPoint adds or updates a point in the lookup
func (pl *PointLookup) AddPoint(point *PointCache) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.points[point.DataLogID] = point
}

// GetPoint retrieves a point based on DataLogID
func (pl *PointLookup) GetPointByDataLogID(dataLogID int) (*PointCache, bool) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	point, exists := pl.points[dataLogID]
	return point, exists
}

// GetPointByName retrieves a point based on DatalogName
func (pl *PointLookup) GetPointByDataLogName(datalogName string) (*PointCache, bool) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	for _, p := range pl.points {
		if p.DatalogName == datalogName {
			return p, true
		}
	}
	return nil, false
}
