package crank

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type ProcessConfig struct {
	Command      []string      `json:"command"`
	Cwd          string        `json:"cwd"`
	StartTimeout time.Duration `json:"start_timeout"`
	StopTimeout  time.Duration `json:"stop_timeout"`
}

var DefaultConfig = &ProcessConfig{
	Command:      []string{},
	Cwd:          "",
	StartTimeout: time.Second * 30,
	StopTimeout:  time.Second * 30,
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
		return DefaultConfig, err
	}
	if len(config.Command) == 0 {
		return config, fmt.Errorf("Missing command")
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

func (self *ProcessConfig) clone() *ProcessConfig {
	c := new(ProcessConfig)
	(*c) = (*self)
	return c
}

func (self *ProcessConfig) String() string {
	return fmt.Sprintf("command=%v start_timeout=%v stop_timeout=%v", self.Command, self.StartTimeout, self.StopTimeout)
}
