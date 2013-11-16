package last

/*
#include <mach/mach.h>
#include <mach/mach_host.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func ReadSysMemStats(s *MemStats) error {
	if s == nil {
		return nil
	}
	var vm_pagesize C.vm_size_t
	var vm_stat C.vm_statistics_data_t
	var count C.mach_msg_type_number_t = C.HOST_VM_INFO_COUNT

	host_port := C.host_t(C.mach_host_self())

	C.host_page_size(host_port, &vm_pagesize)

	status := C.host_statistics(
		host_port,
		C.HOST_VM_INFO,
		C.host_info_t(unsafe.Pointer(&vm_stat)),
		&count)

	if status != C.KERN_SUCCESS {
		return fmt.Errorf("could not get vm statistics: %d", status)
	}

	// Stats in bytes
	free := uint64(vm_stat.free_count)
	active := uint64(vm_stat.active_count)
	inactive := uint64(vm_stat.inactive_count)
	wired := uint64(vm_stat.wire_count)
	pagesize := uint64(vm_pagesize)

	s.Used = (active + inactive + wired) * pagesize
	s.Free = free * pagesize
	s.Total = s.Used + s.Free

	return nil
}
