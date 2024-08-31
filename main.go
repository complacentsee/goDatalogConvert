package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/complacentsee/goDatalogConvert/LibDAT"
	"github.com/complacentsee/goDatalogConvert/LibFTH"
	"github.com/complacentsee/goDatalogConvert/LibPI"
	"github.com/complacentsee/goDatalogConvert/libUtil"
)

type datRecord struct {
	Records     []*LibDAT.DatFloatRecord
	PointLookup *LibPI.PointLookup
}

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

	var wg sync.WaitGroup

	if *debugLevel {
		programLevel.Set(slog.LevelDebug)
		slog.Debug("Debug level logging enabled")
	}

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

	slog.Info(fmt.Sprintf("Connecting to piserver at: %s, with process name %s", *host, *processName))

	LibFTH.SetProcessName(*processName)
	err := LibFTH.Connect(*host)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	defer LibFTH.Disconnect()

	if _, err := os.Stat(*dirPath); os.IsNotExist(err) {
		slog.Error("Error: Directory not found")
		return
	}

	dr, err := LibDAT.NewDatReader(*dirPath)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Semaphore to limit concurrent DAT file reads to 3
	sem := make(chan struct{}, 10)

	// Buffered channel to hold one extra DAT file's worth of records
	recordChan := make(chan datRecord)
	doneChan := make(chan struct{})

	// Start inserter goroutine
	go insertRecords(recordChan, doneChan)

	// Start processing files one by one in a separate goroutine
	go func() {
		for _, floatfileName := range dr.GetFloatFiles() {
			wg.Add(1)         // Increment WaitGroup counter for each file to process
			sem <- struct{}{} // Acquire semaphore slot
			go processFile(floatfileName, dr, tagMaps, useTagMap, recordChan, &wg, sem)
		}

		// Wait for all processing to finish and close the channel
		wg.Wait()
		close(recordChan)
	}()

	// Wait for all records to be inserted
	<-doneChan

	slog.Info("Processing complete.")
}

func processFile(fileName string, dr *LibDAT.DatReader, tagMaps map[string]string, useTagMap bool, recordChan chan<- datRecord, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()          // Decrement the counter when the function returns
	defer func() { <-sem }() // Release semaphore slot when done

	start := time.Now()
	pointCache := LibPI.NewPointLookup()

	tags, err := dr.ReadTagFile(fileName)
	if err != nil {
		slog.Error(fmt.Sprintf("Error reading tag file for %s: %v", fileName, err))
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

		LibDAT.PrintTagRecord(tag)
		_, exists := pointCache.GetPointByDataLogName(tag.Name)
		if exists {
			continue
		}

		pointC := LibFTH.AddToPIPointCache(tag.Name, tag.ID, 0, tagName)
		pointCache.AddPoint(pointC)
	}
	pointCache.PrintAll()

	records, err := dr.ReadFloatFile(fileName)
	if err != nil {
		slog.Error(fmt.Sprintf("Error reading float file for %s: %v", fileName, err))
		return
	}

	// Immediately send the records to the channel for historian processing
	recordChan <- datRecord{Records: records, PointLookup: pointCache}

	duration := time.Since(start)
	slog.Info(fmt.Sprintf("Loaded %d records from %s in %f seconds", len(records), fileName, duration.Seconds()))

	// Release semaphore now that the file reading and preparation is complete
	// (This was moved from the defer to here to ensure it happens as soon as possible)
}

func insertRecords(recordChan <-chan datRecord, doneChan chan<- struct{}) {
	var wg sync.WaitGroup // Use a separate WaitGroup for the historian inserts

	for datrecords := range recordChan {
		wg.Add(1)
		go func(dr datRecord) {
			defer wg.Done()

			records := dr.Records
			pointCache := dr.PointLookup
			err := LibFTH.ConvertDatFloatRecordsToPutSnapshots(records, pointCache)
			if err != nil {
				slog.Error(fmt.Sprintf("Error inserting values into historian: %v", err))
			}

		}(datrecords)
	}

	// Wait for all historian inserts to finish
	wg.Wait()

	// Signal completion
	doneChan <- struct{}{}
}
