package audit

import (
	"encoding/csv"
	"os"
	"sync"
)

const maxBufferSize = 50

type Recorder struct {
	mutex      sync.Mutex
	active     bool
	file       *os.File
	rec        *csv.Writer
	bufferSize int
}

var globalRecorder Recorder = Recorder{
	active:     false,
	file:       nil,
	rec:        nil,
	bufferSize: 0,
}

func InitRecorder(filename string) {
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
