package LibDAT

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DatFloatRecord struct {
	TimeStamp time.Time
	TagID     int
	Val       float64
	Status    byte
	Marker    byte
	IsValid   bool
}

type Status struct {
	Good               bool
	CommunicationError bool
	Disabled           bool
	Stale              bool
	Uninitialized      bool
}

type Marker struct {
	Began bool
	Ended bool
}

func (dr *DatReader) ReadFloatFileHeader(filename string) (*int32, error) {

	// Open the float file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open float file: %v", err)
	}
	defer file.Close()

	// Create a binary reader
	br := binaryReader(file)

	// Read the version byte
	_, err = br.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read version byte: %v", err)
	}

	// Read date parts
	_, err = br.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read year byte: %v", err)
	}

	_, err = br.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read month byte: %v", err)
	}

	_, err = br.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read day byte: %v", err)
	}

	// Read the number of rows
	rowCount, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("failed to read row count: %v", err)
	}
	slog.Info(fmt.Sprintf("%d values", rowCount))

	// Seek to the starting position for reading float records
	if _, err := file.Seek(0x121, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to float records: %v", err)
	}

	return &rowCount, nil
}

func (dr *DatReader) ReadFloatFileRecords(filename string, rowCount int32) ([]*DatFloatRecord, error) {
	var records []*DatFloatRecord

	// Open the float file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open float file: %v", err)
	}
	defer file.Close()

	// Create a binary reader
	br := binaryReader(file)

	// Seek to the starting position for reading float records
	if _, err := file.Seek(0x121, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to float records: %v", err)
	}

	records = make([]*DatFloatRecord, rowCount)

	// Read the float records
	for i := 0; i < int(rowCount); i++ {
		rec, err := readNextDatFloatRecord(br)
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading record: %v", err))
			continue
		}
		records[i] = rec
	}

	return records, nil
}

// ReadFloatFile reads the float file and returns a slice of DatFloatRecord
func (dr *DatReader) ReadFloatFile(filename string) ([]*DatFloatRecord, error) {
	count, err := dr.ReadFloatFileHeader(filename)
	if err != nil {
		return nil, err
	}

	records, err := dr.ReadFloatFileRecords(filename, *count)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func readNextDatFloatRecord(r io.Reader) (*DatFloatRecord, error) {
	// Allocate a single buffer for all the data we need to read
	buffer := make([]byte, 39)

	// Read all 31 bytes into the buffer
	if _, err := r.Read(buffer); err != nil {
		return nil, err
	}

	timeSec := string(buffer[1:17])
	datetime, err := time.Parse("2006010215:04:05", timeSec)
	if err != nil {
		return &DatFloatRecord{IsValid: false}, err
	}

	milli, err := strconv.Atoi(strings.TrimSpace(string(buffer[17:20])))
	if err != nil {
		slog.Error("failed to set milli bytes")
		return &DatFloatRecord{IsValid: false}, err
	}
	datetime = datetime.Add(time.Duration(milli) * time.Millisecond)

	tagID, err := strconv.Atoi(strings.TrimSpace(string(buffer[20:25])))
	if err != nil {
		slog.Error("failed to create TagID")
		return &DatFloatRecord{IsValid: false}, err
	}

	return &DatFloatRecord{
		TimeStamp: datetime,
		TagID:     tagID,
		Val:       math.Float64frombits(binary.LittleEndian.Uint64(buffer[25:33])),
		Status:    buffer[33],
		Marker:    buffer[34],
		IsValid:   true,
	}, nil
}

type DatTagRecord struct {
	Name  string
	ID    int
	Type  int
	Dtype int
}

func NewDatTagRecord(r io.Reader) (*DatTagRecord, error) {
	var err error
	// Skip 1 byte
	if _, err = r.Read(make([]byte, 1)); err != nil {
		slog.Error("error skipping byte")
		return nil, err
	}

	nameBytes := make([]byte, 255)
	if _, err = r.Read(nameBytes); err != nil {
		slog.Error("error creating name bytes")
		return nil, err
	}
	name := strings.TrimSpace(string(nameBytes))

	idBytes := make([]byte, 5)
	if _, err = r.Read(idBytes); err != nil {
		slog.Error("error reading idBytes")
		return nil, err
	}
	id, err := strconv.Atoi(strings.TrimSpace(string(idBytes)))
	if err != nil {
		slog.Error("error converting id")
		return nil, err
	}

	typeBytes := make([]byte, 1)
	if _, err = r.Read(typeBytes); err != nil {
		slog.Error("error reading typeBytes")
		return nil, err
	}
	typ, err := strconv.Atoi(string(typeBytes))
	if err != nil {
		slog.Error("error converting type bytes")
		return nil, err
	}

	dtypeBytes := make([]byte, 2)
	if _, err = r.Read(dtypeBytes); err != nil {
		return nil, err
	}
	dtype, err := strconv.Atoi(strings.TrimSpace(string(dtypeBytes)))
	if err != nil {
		return nil, err
	}

	return &DatTagRecord{Name: name, ID: id, Type: typ, Dtype: dtype}, nil
}

// PrintTagRecord prints the details of a DatTagRecord in a formatted way
func PrintTagRecord(tag *DatTagRecord) {
	slog.Debug(fmt.Sprintf("Tag Name: %-100s | Tag ID: %-5d | Type: %-3d | Dtype: %-3d",
		tag.Name, tag.ID, tag.Type, tag.Dtype))
}

// ReadTagFile reads the tag file associated with a float file and returns the DatTagRecord instances
func (dr *DatReader) ReadTagFile(floatfileName string) ([]*DatTagRecord, error) {

	count, _, err := dr.ReadTagFileHeader(floatfileName)
	if err != nil {
		return nil, err
	}

	records, err := dr.ReadTagRecordsFile(floatfileName, int(*count))
	if err != nil {
		return nil, err
	}

	return records, nil
}

// ReadTagFile reads the tag file associated with a float file and returns the DatTagRecord instances
func (dr *DatReader) ReadTagFileHeader(floatfileName string) (*int32, *string, error) {
	// Replace " (Float)" with " (Tagname)" to get the tag file name
	tagfileName := strings.Replace(floatfileName, " (Float)", " (Tagname)", 1)

	// Open the tag file
	file, err := os.Open(tagfileName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open tag file: %v", err)
	}
	defer file.Close()

	// Create a binary reader
	br := binaryReader(file)

	// Read the version byte
	_, err = br.ReadByte()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read version byte: %v", err)
	}

	// Read date parts
	year, err := br.ReadByte()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read year byte: %v", err)
	}

	month, err := br.ReadByte()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read month byte: %v", err)
	}

	day, err := br.ReadByte()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read day byte: %v", err)
	}

	yearInt := int(year) + 1900
	monthInt := int(month)
	dayInt := int(day)

	// Format as year-month-day
	dateString := fmt.Sprintf("%04d-%02d-%02d", yearInt, monthInt, dayInt)

	// Read the number of rows
	rowCount, err := br.ReadInt32()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read row count: %v", err)
	}

	return &rowCount, &dateString, nil
}

// ReadTagFile reads the tag file associated with a float file and returns the DatTagRecord instances
func (dr *DatReader) ReadTagRecordsFile(floatfileName string, rowCount int) ([]*DatTagRecord, error) {
	var records []*DatTagRecord

	// Replace " (Float)" with " (Tagname)" to get the tag file name
	tagfileName := strings.Replace(floatfileName, " (Float)", " (Tagname)", 1)

	// Open the tag file
	file, err := os.Open(tagfileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open tag file: %v", err)
	}
	defer file.Close()

	// Create a binary reader
	br := binaryReader(file)

	// Seek to the starting position for reading tag records
	if _, err := file.Seek(0xA1, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to tag records: %v", err)
	}

	// Read the tag records
	for i := 0; i < rowCount; i++ {
		rec, err := NewDatTagRecord(br)
		if err != nil {
			return nil, fmt.Errorf("failed to read tag record: %v", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

type DatReader struct {
	FloatFileNames []string
}

func NewDatReader(path string) (*DatReader, error) {
	path = strings.ReplaceAll(path, "\"", "")
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var floatFileNames []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), " (Float).DAT") {
			floatFileNames = append(floatFileNames, filepath.Join(path, file.Name()))
		}
	}

	if len(floatFileNames) == 0 {
		return nil, fmt.Errorf("no input files")
	}

	return &DatReader{FloatFileNames: floatFileNames}, nil
}

func (dr *DatReader) GetFloatFiles() []string {
	return dr.FloatFileNames
}

// Helper function to create a binary reader with little-endian format
func binaryReader(r io.Reader) *binaryReaderio {
	return &binaryReaderio{Reader: r}
}

type binaryReaderio struct {
	io.Reader
}

func (br *binaryReaderio) ReadByte() (byte, error) {
	var b [1]byte
	_, err := br.Read(b[:])
	return b[0], err
}

func (br *binaryReaderio) ReadInt32() (int32, error) {
	var i int32
	err := binary.Read(br, binary.LittleEndian, &i)
	return i, err
}

func PrintDatFloatRecord(record *DatFloatRecord) {
	slog.Debug(fmt.Sprintf("TimeStamp: %s | TagID: %04d | Value: %16.8f | Status: %c | Marker: %c | Valid: %t",
		record.TimeStamp.Format("2006-01-02 15:04:05.000"),
		record.TagID,
		record.Val,
		record.Status,
		record.Marker,
		record.IsValid))
}
