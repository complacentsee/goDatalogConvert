package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/complacentsee/goDatalogConvert/libFTH"
	"github.com/complacentsee/goDatalogConvert/libdat"
)

const (
	BatchSize = 1000
)

func main() {
	// Define the command-line flag for the directory path
	dirPath := flag.String("path", ".", "Path to the directory containing DAT files")
	host := flag.String("host", "localhost", "hostname of pi server")
	processName := flag.String("processName", "dat2fth", "hostname of pi server")
	debugLevel := flag.Bool("debug", false, "Enable Debug Logging")
	flag.Parse()

	var programLevel = new(slog.LevelVar) // Info by default

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel}))
	slog.SetDefault(logger)

	if *debugLevel {
		programLevel.Set(slog.LevelDebug)
		slog.Debug("Debug level logging enabled")
	}

	slog.Info(fmt.Sprintf("Connecting to piserver at: %s, with process name %s", *host, *processName))
	libFTH.SetProcessName(*processName)
	err := libFTH.Connect(*host)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	defer libFTH.Disconnect()

	// Check if the directory exists
	if _, err := os.Stat(*dirPath); os.IsNotExist(err) {
		slog.Error("Error: Directory not found")
		return
	}

	// pointCache := libPI.NewPointLookup()

	// PointList := []string{"fastcosine", "fastcosine90", "noexists"}

	// // Loop over the slice using a for loop
	// for i, point := range PointList {
	// 	_, exists := pointCache.GetPointByDataLogName(point)
	// 	if exists {
	// 		continue
	// 	}
	// 	pointC := libFTH.GetPIPointCache(point, i, 0, &point)
	// 	pointCache.AddPoint(pointC)
	// }

	// pointCache.PrintAll()

	// Initialize the DatReader
	dr, err := libdat.NewDatReader(*dirPath)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Process each file in the directory
	for _, floatfileName := range dr.GetFloatFiles() {

		slog.Info(fmt.Sprintf("Converting %s", filepath.Base(floatfileName)))
		pointIDs := make(map[int]int)

		// Read and process the Tag file associated with the Float file
		tags, err := dr.ReadTagFile(floatfileName)
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading tag file for %s: %v", floatfileName, err))
			return
		}

		for _, tag := range tags {
			//libdat.PrintTagRecord(tag)
			pointIDs[tag.ID] = tag.ID // Simplified mapping since PI stuff is removed
		}

		start := time.Now() // Start timing

		// Process each record in the Float file
		records, err := dr.ReadFloatFile(floatfileName)
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading float file for %s: %v", floatfileName, err))
			return
		}

		duration := time.Since(start) // Calculate duration

		slog.Info(fmt.Sprintf("Processed %d records from %s in %v", len(records), floatfileName, duration))

		for _, record := range records {
			slog.Debug(fmt.Sprintf("TimeStamp: %s | TagID: %04d | Value: %16.8f | Status: %c | Marker: %c | Valid: %t",
				record.TimeStamp.Format("2006-01-02 15:04:05.000"),
				record.TagID,
				record.Val,
				record.Status,
				record.Marker,
				record.IsValid))

		}

	}
}

// Placeholder function to process the batch (e.g., store the data)
// func processBatch(batchCount int, ptids []int32, vs []float64, ts []libdat.PITimestamp) {
// 	// This function would normally handle storing or further processing the batch
// 	fmt.Printf("Processing batch of %d records\n", batchCount)
// }
