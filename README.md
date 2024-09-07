# goDatalogConvert
[![Build Status](https://github.com/complacentsee/goDatalogConvert/actions/workflows/buildvalidate.yml/badge.svg)](https://github.com/complacentsee/goDatalogConvert/actions/workflows/buildvalidate.yml)

`goDatalogConvert` is a Go-based tool for importing large amounts of raw DAT files into a FactoryTalk Historian server. This tool leverages a low-level C API (`piapi.dll`) to push data efficiently into the historian, capable of processing up to 250,000 points per second.

## Features

- Imports raw `.DAT` files directly into a FactoryTalk Historian server.
- Supports mapping of Datalog tags to Historian tags using a CSV file.
- Allows configurable logging levels for better debugging and monitoring.
- Concurrent processing of multiple DAT files for efficient data import.

## Requirements

To run `goDatalogConvert`, ensure the following dependencies are installed:

- **piapi.dll**: This is installed by default on all servers with the pi-sdk installed/ all PINS servers. 
  
  When running the import from a remote node:
  - Ensure that the remote IP address has write permissions in the Historian server settings (SMT > Security > Mappings & Trusts).
  - You may need to configure access based on the process name (`dat2fth`) in SMT > Security.

## Latest Release
[Latest Release](https://github.com/complacentsee/goDatalogConvert/releases/latest)


## Building

1. Clone the repository:
    ```bash
    git clone https://github.com/complacentsee/goDatalogConvert.git
    cd goDatalogConvert
    ```

2. Build the executable (if cross compiling):
    ```bash
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
    CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
    go build -v -o goDatalogConvert.exe
    ```

3. Run the executable with the appropriate flags:
    ```bash
    ./goDatalogConvert.exe -path /path/to/dat/files -host historian_server -processName dat2fth -tagMapCSV /path/to/tagmap.csv
    ```

## Usage

The `goDatalogConvert` tool reads all `.DAT` files in the specified directory and pushes the values onto a FactoryTalk Historian server.

### Command-line Arguments

- `-path` (default: `.`): Path to the directory containing DAT files.
- `-host` (default: `localhost`): The hostname of the FactoryTalk Historian server.
- `-processName` (default: `dat2fth`): The process name used for the historian connection.
- `-tagMapCSV`: Path to a CSV file containing the tag map for translating Datalog tags to Historian tags.
- `-debug`: Enable debug-level logging for detailed output.

### Example

```bash
./goDatalogConvert.exe -path /data/datfiles -host historian-server -processName dat2fth -tagMapCSV tagmap.csv -debug
```

## Important Notes

- Ensure that all Historian points are created manually before starting the import. This ensures that the data is correctly mapped and stored.
- For best results, consider stopping incoming real-time data collection on the historian server and configure appropriate compression settings (`CompDev`) for each point.

## Credits

This tool was inspired by the original work on the `DatalogConvert` project by [shrddr](https://github.com/shrddr/DatalogConvert). Special thanks to the original author for laying the groundwork.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
