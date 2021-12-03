package network

import (
	"encoding/xml"
	"fmt"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
	"strings"
	"time"
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

func decodeXMLArray(txt []interface{}) map[string]string {
	dict := make(map[string]string, 0)

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

////
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
	fmt.Println("waitForJob>", jobState.Value().(int32))
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := wmi.WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "WaitForJob")
		}
	}
	return nil
}

func (m *Manager) waitForJob2(jobState *wmi.Result, jobPath ole.VARIANT) error {
	fmt.Println("waitForJob>", jobState.Value().(int32))
	if jobState.Value().(int32) == wmi.JobStatusStarted {
		err := WaitForJob(jobPath.Value().(string))
		if err != nil {
			return errors.Wrap(err, "WaitForJob")
		}
	}
	return nil
}

// WaitForJob will wait for a WMI job to complete
func WaitForJob(jobPath string) error {
	for {
		jobData, err := NewJobState(jobPath)
		if err != nil {
			return err
		}
		if jobData.JobState == wmi.JobStatusRunning {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if jobData.JobState != wmi.JobStateCompleted {
			return fmt.Errorf("Job failed: %s (%d)", jobData.ErrorDescription, jobData.ErrorCode)
		}
		break
	}
	return nil
}

// NewJobState returns a new Jobstate, given a path
func NewJobState(path string) (wmi.JobState, error) {
	conn, err := wmi.NewLocation(path)
	if err != nil {
		return wmi.JobState{}, err
	}

	// This may blow up. In theory, both CIM_ConcreteJob and Msvm_Concrete job will
	// work with this. Also, anything that inherits CIM_ConctreteJob will also work.
	// TODO: Make this more robust
	//if strings.HasSuffix(conn.Class, "_ConcreteJob") == false {
	//	return wmi.JobState{}, fmt.Errorf("Path is not a valid ConcreteJob. Got: %s", conn.Class)
	//}

	jobData, err := conn.GetResult()
	if err != nil {
		return wmi.JobState{}, err
	}
	//fmt.Println(jobData.GetText(2))

	j := wmi.JobState{}
	err = wmi.PopulateStruct(jobData, &j)
	if err != nil {
		fmt.Println(1)
		return wmi.JobState{}, err
	}
	return j, nil
}
