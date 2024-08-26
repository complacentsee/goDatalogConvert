package libFTH

/*
#cgo CFLAGS: -w
#cgo LDFLAGS: -L"C:/Program Files/Rockwell Software/FactoryTalk Historian/PIPC/bin" -lpiapi

#include <stdint.h>
#include <stdlib.h>

// C function prototypes
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
	"time"
	"unsafe"

	"github.com/complacentsee/goDatalogConvert/libPI"
)

func Connect(serverName string) error {
	cServerName := C.CString(serverName)
	defer C.free(unsafe.Pointer(cServerName))

	err := C.piut_setservernode(cServerName)
	if err != 0 {
		return fmt.Errorf("piut_setservernode returned error %d", err)
	}
	return nil
}

func SetProcessName(processName string) {
	cProcessName := C.CString(processName)
	defer C.free(unsafe.Pointer(cProcessName))
	C.piut_setprocname(cProcessName)
}

func Disconnect() error {
	err := C.piut_disconnect()
	if err != 0 {
		return fmt.Errorf("piut_setservernode returned error %d", err)
	}
	return nil
}

func GetPointNumber(ptName string) (int32, error) {
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
	return int32(pointNumber), nil
}

// GetPointType retrieves the point type for a given point ID
func GetPointType(ptId int32) (libPI.PointType, error) {
	var cType C.char

	// Call the C function to get the point type
	res := C.pipt_pointtype(C.int32_t(ptId), &cType)
	if res > 0 {
		return libPI.PointTypeUnknown, fmt.Errorf("system error occurred, error code: %d", res)
	} else if res == -1 {
		return libPI.PointTypeUnknown, fmt.Errorf("point does not exist, point ID: %d", ptId)
	}

	// Map the returned type character to the PointType enum
	switch cType {
	case 'R':
		return libPI.PointTypeReal, nil
	case 'I':
		return libPI.PointTypeInteger, nil
	case 'D':
		return libPI.PointTypeDigital, nil
	default:
		return libPI.PointTypeUnknown, fmt.Errorf("unknown point type: %c", cType)
	}
}

func PutSnapshot(ptId int32, v float64, dt time.Time) error {
	ival := C.int32_t(0)
	bsize := C.uint32_t(0)
	istat := C.int32_t(0)
	flags := C.int16_t(0)
	ts := libPI.NewPITIMESTAMP(dt)

	err := C.pisn_putsnapshotx(C.int32_t(ptId), (*C.double)(&v), &ival, nil, &bsize, &istat, &flags, (*C.struct_PITIMESTAMP)(unsafe.Pointer(&ts)))
	if err != 0 {
		return fmt.Errorf("pisn_putsnapshotx returned error %d", err)
	}
	return nil
}

func PutSnapshots(count int32, ptids []int32, vs []float64, ts []libPI.PITIMESTAMP) error {
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
			if errors[i] != 0 {
				return fmt.Errorf("pisn_putsnapshotsx returned error %d, item %d, ts %v, err %d", err, i, ts[i], errors[i])
			}
		}
	}
	return nil
}

func AddToPIPointCache(datalogName string, datalogID int, datalogType int, piPointName *string) *libPI.PointCache {
	if piPointName == nil {
		piPointName = &datalogName
	}

	slog.Debug(fmt.Sprintf("Looking up PI Point %s", *piPointName))
	PIPointID, err := GetPointNumber(*piPointName)
	if err != nil {
		return &libPI.PointCache{
			DatalogName: datalogName,
			DataLogID:   datalogID,
			DataLogType: 0,
			Process:     false,
			PIName:      piPointName,
		}
	}

	// TODO: Confirm that types are compatible. EG: Don't assume all data is float/real
	slog.Debug(fmt.Sprintf("Looking up PI Type for PI ID %d", PIPointID))
	PIPointType, err := GetPointType(PIPointID)
	if err != nil {
		return &libPI.PointCache{
			DatalogName: datalogName,
			DataLogID:   datalogID,
			DataLogType: datalogType,
			Process:     false,
			PIName:      piPointName,
			PIId:        &PIPointID,
			PiType:      &PIPointType,
		}
	}

	return &libPI.PointCache{
		DatalogName: datalogName,
		DataLogID:   datalogID,
		DataLogType: 0,
		Process:     true,
		PIName:      piPointName,
		PIId:        &PIPointID,
		PiType:      &PIPointType,
	}
}
