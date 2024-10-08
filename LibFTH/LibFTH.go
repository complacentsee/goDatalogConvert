package LibFTH

/*
#cgo LDFLAGS: -L"./" -lpiapi

#include <stdint.h>
#include <stdlib.h>

// C function prototypes
struct PITIMESTAMP {
    int16_t year;
    int16_t month;
    int16_t day;
    int16_t hour;
    int16_t minute;
    int16_t second;
    int32_t subsecond;
};
extern int32_t piut_setservernode(const char* name);
extern int32_t piut_disconnect();
extern void piut_setprocname(const char* name);
extern int32_t pipt_findpoint(const char* name, int32_t* pointNumber);
extern int32_t pipt_pointtype(int32_t ptnum, char* type);
extern int32_t pisn_putsnapshotx(int32_t ptnum, double* drval, int32_t* ival, uint8_t* bval, uint32_t* bsize,
                                int32_t* istat, int16_t* flags, struct PITIMESTAMP* timestamp);
extern int32_t pisn_putsnapshotsx(int32_t count, int32_t* ptnum, double* drval, int32_t* ival, uint8_t* bval,
                                 uint32_t* bsize, int32_t* istat, int16_t* flags, struct PITIMESTAMP* timestamp, int32_t* errors);
*/
import "C"
import (
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unsafe"

	"github.com/complacentsee/goDatalogConvert/LibDAT"
	"github.com/complacentsee/goDatalogConvert/LibPI"
)

var piapidll sync.Mutex
var mu sync.Mutex
var historianCache = make(map[string]LibPI.HistorianPoint)

func Connect(serverName string) error {
	piapidll.Lock()
	defer piapidll.Unlock()
	cServerName := C.CString(serverName)
	defer C.free(unsafe.Pointer(cServerName))

	err := C.piut_setservernode(cServerName)
	if err != 0 {
		return fmt.Errorf("piut_setservernode returned error %d", err)
	}
	return nil
}

func SetProcessName(processName string) {
	piapidll.Lock()
	defer piapidll.Unlock()
	cProcessName := C.CString(processName)
	defer C.free(unsafe.Pointer(cProcessName))
	C.piut_setprocname(cProcessName)
}

func Disconnect() error {
	piapidll.Lock()
	defer piapidll.Unlock()
	err := C.piut_disconnect()
	if err != 0 {
		return fmt.Errorf("piut_setservernode returned error %d", err)
	}
	return nil
}

func GetPointNumber(ptName string) (int32, error) {
	mu.Lock()
	if point, ok := historianCache[ptName]; ok {
		mu.Unlock()
		return point.PIId, nil
	}
	mu.Unlock()

	piapidll.Lock()
	defer piapidll.Unlock()
	if len(ptName) > 80 {
		return 0, fmt.Errorf("historian point name %s > 80 characters not supported", ptName)
	}

	cPtName := C.CString(ptName)
	defer C.free(unsafe.Pointer(cPtName))

	var pointNumber C.int32_t
	err := C.pipt_findpoint(cPtName, &pointNumber)
	if err != 0 {
		return 0, fmt.Errorf("error finding historian point %s, pipt_findpoint returned error %d", ptName, err)
	}

	ptNumber := int32(pointNumber)

	mu.Lock()
	historianCache[ptName] = LibPI.HistorianPoint{PIId: ptNumber}
	mu.Unlock()
	return ptNumber, nil
}

func PutSnapshots(count int32, ptids []int32, vs []float64, ts []LibPI.PITIMESTAMP) (time.Duration, error) {
	start := time.Now()
	piapidll.Lock()
	waitDuration := time.Since(start)
	defer piapidll.Unlock()
	ivals := make([]C.int32_t, count)
	bsizes := make([]C.uint32_t, count)
	istats := make([]C.int32_t, count)
	flags := make([]C.int16_t, count)
	errors := make([]C.int32_t, count)

	cPtids := (*C.int32_t)(unsafe.Pointer(&ptids[0]))
	cVs := (*C.double)(unsafe.Pointer(&vs[0]))
	cTs := (*C.struct_PITIMESTAMP)(unsafe.Pointer(&ts[0]))

	err := C.pisn_putsnapshotsx(C.int32_t(count), cPtids, cVs, &ivals[0], nil, &bsizes[0], &istats[0], &flags[0], cTs, &errors[0])
	if err != 0 {
		for i := 0; i < int(count); i++ {
			if errors[i] != 0 && errors[i] != -109 {
				return waitDuration, fmt.Errorf("pisn_putsnapshotsx returned error %d, item %d, ts %v, err %d", err, i, ts[i], errors[i])
			}
		}
	}
	return waitDuration, nil
}

func AddToPIPointCache(datalogName string, datalogID int, datalogType int, piPointName string) *LibPI.PointCache {
	slog.Debug(fmt.Sprintf("Looking up PI Point %s", piPointName))
	PIPointID, err := GetPointNumber(piPointName)
	if err != nil {
		return &LibPI.PointCache{
			DatalogName: datalogName,
			DataLogID:   datalogID,
			DataLogType: 0,
			Process:     false,
			PIName:      piPointName,
		}
	}

	// TODO: Confirm that types are compatible. EG: Don't assume all data is float/real
	// slog.Debug(fmt.Sprintf("Looking up PI Type for PI ID %d", PIPointID))
	// if err != nil {
	// 	return &LibPI.PointCache{
	// 		DatalogName: datalogName,
	// 		DataLogID:   datalogID,
	// 		DataLogType: datalogType,
	// 		Process:     false,
	// 		PIName:      piPointName,
	// 		PIId:        &PIPointID,
	// 	}
	// }

	return &LibPI.PointCache{
		DatalogName: datalogName,
		DataLogID:   datalogID,
		DataLogType: 0,
		Process:     true,
		PIName:      piPointName,
		PIId:        &PIPointID,
	}
}

func ConvertDatFloatRecordsToPutSnapshots(records []*LibDAT.DatFloatRecord, pointLookup *LibPI.PointLookup) error {
	// Prepare slices for PutSnapshots inputs
	ptids := make([]int32, 0, len(records))
	vs := make([]float64, 0, len(records))
	ts := make([]LibPI.PITIMESTAMP, 0, len(records))
	var count int32 = 0
	start := time.Now()

	for _, record := range records {
		if record == nil {
			continue // Skip nil records to avoid dereferencing nil pointers
		}

		// Use the point lookup to get the PI Point ID
		piPointID, exists := pointLookup.GetPointIDByDataLogID(record.TagID)
		if !exists || piPointID == nil {
			continue
		}

		piTimestamp := LibPI.NewPITIMESTAMP(record.TimeStamp)

		// Append the mapped values to the slices
		ptids = append(ptids, *piPointID)
		vs = append(vs, record.Val)
		ts = append(ts, piTimestamp)
		count++
	}

	if count < 1 {
		return fmt.Errorf("no valid entries to push to historian")
	}

	waitingDuration, err := PutSnapshots(count, ptids, vs, ts)
	duration := time.Since(start)

	slog.Info(fmt.Sprintf("Pushed %d records to historian in %.2f seconds, waited %.2f seconds", count, duration.Seconds()-waitingDuration.Seconds(), waitingDuration.Seconds()))

	return err
}

// func ConvertDatFloatRecordsToPutSnapshots(records []*LibDAT.DatFloatRecord, pointLookup *LibPI.PointLookup) error {
// 	// Prepare slices for PutSnapshots inputs
// 	var ptids []int32
// 	var vs []float64
// 	var ts []LibPI.PITIMESTAMP
// 	var count int32 = 0

// 	for i, record := range records {

// 		// Use the point lookup to get the PI Point ID
// 		piPointID, exists := pointLookup.GetPointIDByDataLogID(record.TagID)
// 		if !exists {
// 			continue
// 			//return fmt.Errorf("point ID not found for DataLogID %d", record.TagID)
// 		}
// 		if piPointID == nil {
// 			continue
// 		}

// 		piTimestamp := LibPI.NewPITIMESTAMP(record.TimeStamp)

// 		// Append the mapped values to the slices
// 		ptids = append(ptids, *piPointID)
// 		vs = append(vs, record.Val)
// 		ts = append(ts, piTimestamp)
// 		count++
// 	}
// 	slog.Info(fmt.Sprintf("Pushing %d records to historian", count))
// 	if count < 1 {
// 		return fmt.Errorf("no Valid entries to push to historian")
// 	}

// 	// Call the PutSnapshots function with the prepared data
// 	return PutSnapshots(count, ptids, vs, ts)
// }
