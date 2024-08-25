package libFTH

/*
#cgo CFLAGS: -I${SRCDIR}/include
#cgo LDFLAGS: -L"C:/Program Files/Rockwell Software/FactoryTalk Historian/PIPC/bin" -lpiapi

#include <stdint.h>
#include <stdlib.h>

// C function prototypes
extern int32_t piut_setservernode(const char* name);
extern int32_t piut_disconnect();
extern void piut_setprocname(const char* name);
extern int32_t pipt_findpoint(const char* name, int32_t* pointNumber);
extern int32_t pisn_putsnapshotx(int32_t ptnum, double* drval, int32_t* ival, uint8_t* bval, uint32_t* bsize,
                                int32_t* istat, int16_t* flags, struct PITIMESTAMP* timestamp);
extern int32_t pisn_putsnapshotsx(int32_t count, int32_t* ptnum, double* drval, int32_t* ival, uint8_t* bval,
                                 uint32_t* bsize, int32_t* istat, int16_t* flags, struct PITIMESTAMP* timestamp, int32_t* errors);
*/
import "C"
import (
	"fmt"
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
