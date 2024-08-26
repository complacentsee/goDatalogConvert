package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/complacentsee/goDatalogConvert/libDAT"
	"github.com/complacentsee/goDatalogConvert/libFTH"
	"github.com/complacentsee/goDatalogConvert/libPI"
	"github.com/complacentsee/goDatalogConvert/libUtil"
)

const (
	BatchSize = 1000
)

func main() {
	// Define the command-line flag for the directory path
	dirPath := flag.String("path", ".", "Path to the directory containing DAT files")
	host := flag.String("host", "localhost", "hostname of pi server")
	processName := flag.String("processName", "dat2fth", "hostname of pi server")
	tagMapCSV := flag.String("tagMapCSV", "", "Path to the CSV file containing the tag map.")
	debugLevel := flag.Bool("debug", false, "Enable Debug Logging")
	flag.Parse()

	var programLevel = new(slog.LevelVar) // Info by default

	tagMaps := make(map[string]string)
	useTagMap := false

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel}))
	slog.SetDefault(logger)

	if *debugLevel {
		programLevel.Set(slog.LevelDebug)
		slog.Debug("Debug level logging enabled")
	}
	// Check if the tag map file is provided
	if *tagMapCSV != "" {
		err := libUtil.LoadTagMapCSV(*tagMapCSV, tagMaps)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to load tag map CSV: %v", err))
			return
		}
		if len(tagMaps) < 1 {
			slog.Error("Application parameters called for tag mapping and failed")
			return
		} else {
			useTagMap = true
			for k, v := range tagMaps {
				slog.Debug(fmt.Sprintf("Datalog Tag: %s, Historian Tag: %s", k, v))
			}
		}
	} else {
		slog.Info("No tag map provided. Continuing without loading tag map.")
	}

	slog.Info(fmt.Sprintf("Connecting to piserver at: %s, with process name %s",
		*host, *processName))

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

	// Initialize the DatReader
	dr, err := libDAT.NewDatReader(*dirPath)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Process each file in the directory
	for _, floatfileName := range dr.GetFloatFiles() {
		slog.Info(fmt.Sprintf("Converting %s", filepath.Base(floatfileName)))
		pointCache := libPI.NewPointLookup()

		// Read and process the tag file associated with the Float file
		tags, err := dr.ReadTagFile(floatfileName)
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading tag file for %s: %v", floatfileName, err))
			return
		}

		for _, tag := range tags {
			tagName := tag.Name
			if useTagMap {
				var exists bool
				tagName, exists = tagMaps[tag.Name]
				if !exists {
					continue
				}
			}

			libDAT.PrintTagRecord(tag)
			_, exists := pointCache.GetPointByDataLogName(tag.Name)
			if exists {
				continue
			}

			pointC := libFTH.AddToPIPointCache(tag.Name, tag.ID, 0, tagName)
			pointCache.AddPoint(pointC)
		}
		pointCache.PrintAll()

		start := time.Now() // Start timing

		// Process each record in the Float file
		records, err := dr.ReadFloatFile(floatfileName)
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading float file for %s: %v", floatfileName, err))
			return
		}
		duration := time.Since(start) // Calculate duration
		start = time.Now()

		slog.Info(fmt.Sprintf("Loaded %d records from %s in %v", len(records), floatfileName, duration))
		err = libFTH.ConvertDatFloatRecordsToPutSnapshots(records, pointCache)
		if err != nil {
			slog.Error(fmt.Sprintf("Error inserting values into historian: %v", err))
		}

		duration = time.Since(start) // Calculate duration
		slog.Info(fmt.Sprintf("Wrote %d records from %s in %v", len(records), floatfileName, duration))
	}
}
