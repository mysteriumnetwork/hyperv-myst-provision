package client

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/hyperv-node/model"

	"github.com/rs/zerolog/log"
)

func VmAgentSetLauncherVersion(ip string) error {
	log.Info().Msg("VmAgentSetLauncherVersion >>>")

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

	m := make(model.KVMap, 0)
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

	log.Info().Msgf("VmAgentUploadKeystore >>> %s", path)
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return errors.Wrap(err, "ioutil.ReadDir")
	}
	for _, f := range files {
		if !f.IsDir() {
			fPath := filepath.Join(path, f.Name())
			file, _ := os.Open(fPath)
			part, _ := writer.CreateFormFile("files", filepath.Base(fPath))

			_, err = io.Copy(part, file)
			if err != nil {
				return errors.Wrap(err, "io.Copy")
			}
			file.Close()
		}
	}

	if err = writer.Close(); err != nil {
		return errors.Wrap(err, "writer.Close")
	}
	url := "http://" + ip + ":8080/upload"
	method := "POST"
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		log.Err(err).Msg("New request")
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
