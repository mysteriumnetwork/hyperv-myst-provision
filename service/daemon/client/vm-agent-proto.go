package client

import (

	//"log"

	"encoding/json"
	"io/ioutil"
	"net/http"

	hyperv_wmi "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
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
