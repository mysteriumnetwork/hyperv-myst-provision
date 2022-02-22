package hyperv_wmi

import (
	"encoding/xml"
	"log"
	"strings"

	"github.com/gabriel-samfira/go-wmi/virt/vm"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

const (
	scsiType = "Microsoft:Hyper-V:Synthetic SCSI Controller"
	diskType = "Microsoft:Hyper-V:Synthetic Disk Drive"
	vhdType  = "Microsoft:Hyper-V:Virtual Hard Disk"
)

func getPathFromResultingResourceSettings(r ole.VARIANT) string {
	URI := r.ToArray().ToValueArray()
	return URI[0].(string)
}

func getPathFromResultingSystem(r ole.VARIANT) string {
	return r.Value().(string)
}

type KVXml struct {
	Property []struct {
		Name  string `xml:"NAME,attr"`
		Value string `xml:"VALUE"`
	} `xml:"PROPERTY"`
}

type KVMap map[string]interface{}

func NewKVMap(i interface{}) KVMap {
	m, ok := i.(map[string]interface{})
	if !ok {
		return nil
	}
	return KVMap(m)
}

func decodeXMLArray(txt []interface{}) KVMap {
	dict := make(KVMap, 0)

	for _, rec := range txt {
		r := strings.NewReader(rec.(string))
		parser := xml.NewDecoder(r)
		kv := KVXml{}
		parser.Decode(&kv)

		data := ""
		for _, v := range kv.Property {
			if v.Name == "Data" {
				data = v.Value
			}
			if v.Name == "Name" {
				dict[v.Value] = data
			}
		}
	}

	return dict
}

////////// manager class utils

func (m *Manager) getHostPath() (string, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{
			wmi.QueryFields{
				Key:   "InstallDate",
				Value: "NULL",
				Type:  wmi.Is},
		},
	}
	computerSystemResult, err := m.con.GetOne(vm.ComputerSystemClass, []string{}, qParams)
	if err != nil {
		return "", errors.Wrap(err, "GetOne")
	}

	pth, err := computerSystemResult.Path()
	if err != nil {
		return "", errors.Wrap(err, "path_")
	}

	return pth, nil
}

func (m *Manager) getDefaultClassValue(class, resourceSubType string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "InstanceID", Value: "%Default%", Type: wmi.Like}},
	}
	if resourceSubType != "" {
		qParams = append(qParams,
			&wmi.AndQuery{wmi.QueryFields{Key: "ResourceSubType", Value: resourceSubType, Type: wmi.Equals}},
		)
	}

	el, err := m.con.GetOne(class, []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "GetOne")
	}
	return el, nil
}

func (m *Manager) waitForJob(jobState *wmi.Result, jobPath ole.VARIANT) error {
	log.Println("waitForJob>", jobState.Value().(int32))
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "WaitForJob")
		}
	}
	return nil
}

//func (m *Manager) waitForJob2(jobState *hyperv-wmi.Result, jobPath ole.VARIANT) error {
//	fmt.Println("waitForJob>", jobState.Value().(int32))
//	if jobState.Value().(int32) == hyperv-wmi.JobStatusStarted {
//		err := WaitForJob(jobPath.Value().(string))
//		if err != nil {
//			return errors.Wrap(err, "WaitForJob")
//		}
//	}
//	return nil
//}
