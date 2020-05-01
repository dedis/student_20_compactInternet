package audit

import (
	"encoding/csv"
	"os"
	"strings"
	"sync"
)

// TODO: Add support for DOS-like paths
const pathSeparator = "/"

const maxBufferSize = 50

// Recorder stores test logs in a thread-safe way
type Recorder struct {
	mutex      sync.Mutex
	active     bool
	folder     string
	file       *os.File
	rec        *csv.Writer
	bufferSize int
}

var globalRecorder Recorder = Recorder{
	active:     false,
	folder:     "",
	file:       nil,
	rec:        nil,
	bufferSize: 0,
}

// GetOutputDir returns the path to the directory used to store logs
// the boolean is true if the path is valid
func GetOutputDir() (bool, string) {
	globalRecorder.mutex.Lock()
	defer globalRecorder.mutex.Unlock()

	if globalRecorder.active {
		return true, globalRecorder.folder
	}

	return false, ""
}

// InitRecorder initializes the globalRecorder
func InitRecorder(filename string) {
	path := strings.Split(filename, pathSeparator)

	globalRecorder.folder = strings.Join(path[:len(path)-1], pathSeparator) + pathSeparator

	var err error
	globalRecorder.file, err = os.Create(filename)
	if err != nil {
		panic("Could not create the output file for the auditor")
	}

	globalRecorder.rec = csv.NewWriter(globalRecorder.file)
	globalRecorder.active = true
	globalRecorder.bufferSize = 0
}

// record is thread-safe
func record(payload ...string) {
	globalRecorder.mutex.Lock()
	defer globalRecorder.mutex.Unlock()

	if globalRecorder.active {
		globalRecorder.rec.Write(payload)
		globalRecorder.bufferSize++
	}
	if globalRecorder.bufferSize >= maxBufferSize {
		globalRecorder.rec.Flush()
		globalRecorder.bufferSize = 0
	}
}

// stopRecording is thread-safe
func stopRecording() {
	globalRecorder.mutex.Lock()
	defer globalRecorder.mutex.Unlock()

	if globalRecorder.active {
		globalRecorder.rec.Flush()
		defer globalRecorder.file.Close()
		globalRecorder.active = false
	}
}
