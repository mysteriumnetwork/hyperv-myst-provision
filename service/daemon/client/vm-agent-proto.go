package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	hyperv_wmi "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/mysteriumnetwork/hyperv-node/vm-agent/server"
	"github.com/rs/zerolog/log"
)

func VmAgentSetLauncherVersion(ip string) error {

	ep := "http://" + ip + ":8080/set?launcher=vmh-0.0.1/windows"
	resp, err := http.Get(ep)
	if err != nil {
		log.Err(err).Msg("Send http request")
		return nil
	}
	if resp.Status != "200" {
		log.Error().Msgf("Status %v: %v", resp.Status, ep)
	}

	return nil
}

func VmAgentGetState(ip string) error {

	ep := "http://" + ip + ":8080/state"
	resp, err := http.Get(ep)
	if err != nil {
		log.Err(err).Msg("Send http request")
		return nil
	}
	if resp.Status != "200" {
		log.Error().Msgf("Status %v: %v", resp.Status, ep)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	m := make(hyperv_wmi.KVMap, 0)
	_ = json.Unmarshal(body, m)
	log.Info().Msgf("VmAgentGetState > %v", m)

	return nil
}

func VmAgentUpdateNode(ip string) error {
	ep := "http://" + ip + ":8080/update"
	resp, err := http.Get(ep)
	if err != nil {
		log.Err(err).Msg("Send http request")
		return err
	}
	if resp.Status != "200" {
		log.Error().Msgf("Status %v: %v", resp.Status, ep)
	}
	return nil
}

func VmAgentUploadKeystore(ip string, path string) error {

	url := "http://" + ip + ":8080/upload"
	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, f := range files {
		if !f.IsDir() {

			fPath := server.Keystore + f.Name()
			file, err := os.Open(fPath)
			part, err := writer.CreateFormFile("files", filepath.Base(fPath))
			_, err = io.Copy(part, file)
			if err != nil {
				fmt.Println(err)
				return err
			}
			file.Close()
		}
	}

	if err = writer.Close(); err != nil {
		fmt.Println(err)
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		log.Err(err).Msg("Send http request")
		return err
	}
	defer res.Body.Close()

	if res.Status != "200" {
		log.Error().Msgf("Status %v: %v", res.Status, url)
	}

	return nil
}
