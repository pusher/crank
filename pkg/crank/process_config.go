package crank

import (
	"encoding/json"
	"os"
	"time"
)

type ProcessConfig struct {
	Command      string        `json:"command"`
	Args         []string      `json:"args"`
	StartTimeout time.Duration `json:"start_timeout"`
	StopTimeout  time.Duration `json:"stop_timeout"`
}

func loadProcessConfig(path string) (config *ProcessConfig, err error) {
	var reader *os.File
	if reader, err = os.Open(path); err != nil {
		return
	}
	defer reader.Close()

	config = new(ProcessConfig)
	jsonDecoder := json.NewDecoder(reader)
	err = jsonDecoder.Decode(config)
	return
}

func (self *ProcessConfig) save(path string) (err error) {
	var writer *os.File
	if writer, err = os.Create(path); err != nil {
		return
	}
	defer writer.Close()

	jsonEncoder := json.NewEncoder(writer)
	return jsonEncoder.Encode(self)
}
