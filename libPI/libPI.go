package libPI

// #include <stdint.h>
import "C"
import "time"

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
