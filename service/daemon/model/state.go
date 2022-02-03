/**
 * Copyright (c) 2021 BlockDev AG
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package model

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Enabled      bool   `json:"enabled"`
	AdapterID    string `json:"adapter-id"`
	ImageVersion string `json:"image_version"`
}

func (c *Config) getDefaultValues() {
	c.Enabled = true
}

func (c *Config) getFilePath() string {
	return `.\myst_vm_helper.conf`
}

func (c *Config) Read() {
	f := c.getFilePath()

	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		c.getDefaultValues()
		c.Save()
		return
	}

	file, err := os.Open(f)
	if err != nil {
		return
	}

	c.getDefaultValues()
	json.NewDecoder(file).Decode(&c)
}

func (c *Config) Save() {
	f := c.getFilePath()

	file, err := os.Create(f)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", " ")
	enc.Encode(&c)
}
