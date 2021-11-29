package network

import (
	"fmt"
	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
	"github.com/pkg/errors"
)

const (
	scsiType = "Microsoft:Hyper-V:Synthetic SCSI Controller"
	diskType = "Microsoft:Hyper-V:Synthetic Disk Drive"
	vhdType  = "Microsoft:Hyper-V:Virtual Hard Disk"
)

func (m *Manager) getDefaultClassValue(resourceSubType string) (*wmi.Result, error) {
	qParams := []wmi.Query{
		&wmi.AndQuery{wmi.QueryFields{Key: "ResourceSubType", Value: resourceSubType, Type: wmi.Equals}},
		&wmi.AndQuery{wmi.QueryFields{Key: "InstanceID", Value: "%Default%", Type: wmi.Like}},
	}

	class := ResourceAllocationSettingData
	if resourceSubType == vhdType {
		class = StorageAllocSettingDataClass
	}

	swColl, err := m.con.Gwmi(class, []string{}, qParams)
	if err != nil {
		return nil, errors.Wrap(err, "Gwmi")
	}
	el, err := swColl.ItemAtIndex(0)
	if err != nil {
		return nil, errors.Wrap(err, "ItemAtIndex")
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
