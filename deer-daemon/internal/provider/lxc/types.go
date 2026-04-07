package lxc

// CTListEntry represents a container in the list returned by GET /nodes/{node}/lxc.
type CTListEntry struct {
	VMID     int     `json:"vmid"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Template int     `json:"template,omitempty"` // 1 if template
	MaxMem   int64   `json:"maxmem"`
	MaxDisk  int64   `json:"maxdisk"`
	CPU      float64 `json:"cpu"`
	Mem      int64   `json:"mem"`
	Uptime   int64   `json:"uptime"`
}

// CTStatus represents the status of an LXC container.
type CTStatus struct {
	VMID   int     `json:"vmid"`
	Name   string  `json:"name"`
	Status string  `json:"status"` // "running", "stopped"
	CPU    float64 `json:"cpu"`
	MaxMem int64   `json:"maxmem"`
	Mem    int64   `json:"mem"`
}

// CTConfig represents an LXC container's configuration.
type CTConfig struct {
	Hostname string `json:"hostname,omitempty"`
	Memory   int    `json:"memory"`
	Cores    int    `json:"cores"`
	Net0     string `json:"net0,omitempty"`
	RootFS   string `json:"rootfs,omitempty"`
}

// CTInterface represents a network interface from the container.
type CTInterface struct {
	Name   string `json:"name"`
	HWAddr string `json:"hwaddr"`
	Inet   string `json:"inet,omitempty"` // e.g. "10.0.0.5/24"
	Inet6  string `json:"inet6,omitempty"`
}

// NodeStatus represents a Proxmox node's resource status.
type NodeStatus struct {
	CPU      float64      `json:"cpu"`
	MaxCPU   int          `json:"maxcpu"`
	Memory   MemoryStatus `json:"memory"`
	RootFS   DiskStatus   `json:"rootfs"`
	Uptime   int64        `json:"uptime"`
	KVersion string       `json:"kversion"`
}

// MemoryStatus is memory info from node status.
type MemoryStatus struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
	Free  int64 `json:"free"`
}

// DiskStatus is disk info from node status.
type DiskStatus struct {
	Total     int64 `json:"total"`
	Used      int64 `json:"used"`
	Available int64 `json:"avail"`
}

// TaskStatus represents the status of an asynchronous Proxmox task.
type TaskStatus struct {
	Status     string `json:"status"`               // "running", "stopped"
	ExitStatus string `json:"exitstatus,omitempty"` // "OK" on success
	Type       string `json:"type"`
	ID         string `json:"id"`
	Node       string `json:"node"`
	PID        int    `json:"pid"`
	StartTime  int64  `json:"starttime"`
	EndTime    int64  `json:"endtime,omitempty"`
}
