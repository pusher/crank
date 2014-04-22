package crank

import (
	"encoding/json"
	"fmt"
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
	if err = jsonDecoder.Decode(config); err != nil {
		return nil, err
	}
	if config.Command == "" {
		return nil, fmt.Errorf("Missing command")
	}
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

func (self *ProcessConfig) String() string {
	return fmt.Sprintf("command=%v args=%v start_timeout=%v stop_timeout=%v", self.Command, self.Args, self.StartTimeout, self.StopTimeout)
}
