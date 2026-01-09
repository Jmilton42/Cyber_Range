package windows

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	// ComputerNamePhysicalDnsHostname for SetComputerNameExW
	ComputerNamePhysicalDnsHostname = 5
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procSetComputerNameEx = kernel32.NewProc("SetComputerNameExW")
)

// SetHostname sets the Windows computer hostname
// Change takes effect after reboot
func SetHostname(hostname string) error {
	hostnamePtr, err := syscall.UTF16PtrFromString(hostname)
	if err != nil {
		return fmt.Errorf("failed to convert hostname: %w", err)
	}

	ret, _, err := procSetComputerNameEx.Call(
		uintptr(ComputerNamePhysicalDnsHostname),
		uintptr(unsafe.Pointer(hostnamePtr)),
	)

	if ret == 0 {
		return fmt.Errorf("SetComputerNameExW failed: %w", err)
	}

	return nil
}
