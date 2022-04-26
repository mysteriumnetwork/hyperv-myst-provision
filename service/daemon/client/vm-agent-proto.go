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

	"github.com/pkg/errors"

	hyperv_wmi "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/mysteriumnetwork/hyperv-node/vm-agent/server"
	"github.com/rs/zerolog/log"
)

func VmAgentSetLauncherVersion(ip string) error {

	ep := "http://" + ip + ":8080/set?launcher=vmh-0.0.1/windows"
	resp, err := http.Get(ep)
	if err != nil {
		return errors.Wrap(err, "VmAgentSetLauncherVersion")
	}
	if resp.StatusCode != 200 {
		log.Error().Msgf("Status %v: %v", resp.Status, ep)
		return errors.New("VmAgentSetLauncherVersion: wrong status")
	}

	return nil
}

func VmAgentGetState(ip string) error {
	log.Info().Msg("VmAgentGetState>")

	ep := "http://" + ip + ":8080/state"
	resp, err := http.Get(ep)
	if err != nil {
		return errors.Wrap(err, "VmAgentGetState")
	}
	if resp.StatusCode != 200 {
		log.Error().Msgf("Status %v: %v", resp.Status, ep)
		return errors.New("VmAgentGetState: wrong status")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "VmAgentGetState")
	}

	m := make(hyperv_wmi.KVMap, 0)
	_ = json.Unmarshal(body, &m)
	log.Info().Msgf("VmAgentGetState> %v", m)

	return nil
}

func VmAgentUpdateNode(ip string) error {
	ep := "http://" + ip + ":8080/update"
	resp, err := http.Get(ep)
	if err != nil {
		return errors.Wrap(err, "VmAgentUpdateNode")
	}
	if resp.StatusCode != 200 {
		log.Error().Msgf("Status %v: %v", resp.Status, ep)
		return errors.New("VmAgentUpdateNode: wrong status")
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
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		log.Err(err).Msg("Send http request")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("VmAgentUploadKeystore: wrong status")
	}

	return nil
}
