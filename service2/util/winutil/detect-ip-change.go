package winutil

import (
	"fmt"

	"github.com/pkg/errors"

	//"log"
	"syscall"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

var (
	modws2_32   = windows.NewLazySystemDLL("ws2_32.dll")
	modiphlpapi = windows.NewLazySystemDLL("iphlpapi.dll")

	procWSACreateEvent    = modws2_32.NewProc("WSACreateEvent")
	procNotifyAddrChange  = modiphlpapi.NewProc("NotifyAddrChange")
	procNotifyRouteChange = modiphlpapi.NewProc("NotifyRouteChange")
)

func WSACreateEvent() (windows.Handle, error) {
	handlePtr, _, errNum := syscall.Syscall(procWSACreateEvent.Addr(), 0, 0, 0, 0)
	if handlePtr == 0 {
		return 0, errNum
	} else {
		return windows.Handle(handlePtr), nil
	}
}

type Handle uintptr

//https://docs.microsoft.com/en-us/windows/desktop/api/iphlpapi/nf-iphlpapi-notifyaddrchange
//DWORD NotifyAddrChange(
//  PHANDLE      Handle,
//  LPOVERLAPPED overlapped
//);
func NotifyAddrChange(handle *Handle, overlapped *windows.Overlapped) error {
	r1, _, e1 := procNotifyAddrChange.Call(uintptr(unsafe.Pointer(handle)), uintptr(unsafe.Pointer(overlapped)))
	if handle == nil && overlapped == nil {
		if r1 == windows.NO_ERROR {
			return nil
		}
	} else {
		if r1 == uintptr(windows.ERROR_IO_PENDING) {
			return nil
		}
	}

	if e1 != windows.ERROR_SUCCESS {
		return e1
	} else {
		return fmt.Errorf("r1:%v", r1)
	}
}

type Notifier struct {
	overlapped windows.Overlapped
}

func NewNotifier() (n Notifier) {
	n.overlapped = windows.Overlapped{}
	hEvent, err := WSACreateEvent()
	if err != nil {
		log.Fatal().Err(err)
	}
	n.overlapped.HEvent = windows.Handle(hEvent)
	return
}

func (n *Notifier) WaitForIPChange() error {
	hand := Handle(0)
	err := NotifyAddrChange(&hand, &n.overlapped)
	if err != nil {
		return errors.Wrap(err, "NotifyAddrChange")
	}

	t := 0xFFFFFFFF
	//t := 1000 * 10 // wait 10 seconds
	event, err := windows.WaitForSingleObject(n.overlapped.HEvent, uint32(t))
	if err != nil {
		return errors.Wrap(err, "WaitForSingleObject")
	}
	if event != windows.WAIT_OBJECT_0 {
		return errors.Wrap(err, "WaitForSingleObject timeout")
	}
	return nil
}
