package rangestudio

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// CustomTopology defines a user-built project layout. The wizard fills
// this from the UI; HCL generation walks every field and emits a
// matching block of OpenTofu configuration.
type CustomTopology struct {
	NetworkMode    string       `json:"network_mode"`     // "single" | "multi" (default "multi")
	Networks       []NetworkDef `json:"networks"`         // user-defined OVN segments per team
	ProjectFW      FirewallDef  `json:"project_firewall"` // project-level OpenWRT (always present)
	TeamFirewall   FirewallDef  `json:"team_firewall"`    // team-level OpenWRT (always present)
	VMs            []VMDef      `json:"vms"`              // per-team VM templates
	PlacementMode  string       `json:"placement_mode"`   // "default" | "round_robin" | "pinned"
	PinnedTarget   string       `json:"pinned_target"`    // when PlacementMode=pinned: full LXD member name (e.g. "micro-04")
	IncludeSalt    bool         `json:"include_salt"`     // include the project_salt VM
	IncludeGuac    bool         `json:"include_guac"`     // include the project_guac VM
	IncludeProjFW  bool         `json:"include_project_fw"` // include the project firewall
}

// NetworkDef represents a single OVN network created per team.
type NetworkDef struct {
	Name string `json:"name"` // sanitized identifier: "lan", "dmz", ...
}

// FirewallDef configures an OpenWRT firewall instance. Interfaces with
// EthIndex 0 are always WAN/DHCP and must not appear here.
type FirewallDef struct {
	Image      string        `json:"image"`      // LXD image alias, e.g. "openwrt-team-new"
	Profile    string        `json:"profile"`    // LXD profile, e.g. "pfsense"
	WANNetwork string        `json:"wan_network"` // upstream network attached to eth0 (e.g. "team_wan", "CLASS_WAN")
	Interfaces []FWInterface `json:"interfaces"`  // ordered list, eth1, eth2, ...
}

// FWInterface maps an OpenWRT ethN (N >= 1) to a network and static IP.
type FWInterface struct {
	EthName string `json:"eth_name"` // "eth1", "eth2"
	Network string `json:"network"`  // matches NetworkDef.Name OR a literal local.* expression
	IP      string `json:"ip"`       // CIDR: "192.168.1.1/24"
}

// VMNetIP maps a network interface to its static IP.
type VMNetIP struct {
	Network string `json:"network"` // network name (matches entry in Networks)
	IP      string `json:"ip"`      // CIDR: "10.10.0.50/24"
}

// VMCloudInit holds cloud-init user-data settings.
type VMCloudInit struct {
	SwapSize     string   `json:"swap_size"`     // e.g. "4G" (empty = no swap)
	SSHPwAuth    bool     `json:"ssh_pwauth"`    // enable SSH password authentication
	PackageUpdate bool    `json:"package_update"` // run package updates on boot
	RunCmds      []string `json:"runcmds"`       // list of commands to run on boot
}

// VMDef represents one VM template that is replicated per team.
type VMDef struct {
	Name     string      `json:"name"`     // logical role name, e.g. "Jumpbox"
	Image    string      `json:"image"`    // LXD image alias
	Profile  string      `json:"profile"`  // LXD profile
	Type     string      `json:"type"`     // "virtual-machine" | "container" (default vm)
	Networks []string    `json:"networks"` // network names this VM connects to (eth0, eth1, ...)
	NetIPs   []VMNetIP   `json:"net_ips"`  // static IPs per network interface
	CPU      string      `json:"cpu"`      // limits.cpu (e.g. "2")
	Memory   string      `json:"memory"`   // limits.memory (e.g. "4GiB")
	DiskSize string      `json:"disk_size"` // root disk override (e.g. "50GiB")
	NeedsGPU bool        `json:"needs_gpu"` // routes to micro-07-gpu when true
	Target   string      `json:"target"`   // override placement: "" | "default" | "round_robin" | "micro-XX"
	Running  bool        `json:"running"`  // start VM on create (false for Windows)
	// Security settings
	SecureBoot bool      `json:"secure_boot"` // enable secure boot (default false for most images)
	CSM        bool      `json:"csm"`         // enable CSM/legacy boot
	// Cloud-init user data (Linux only)
	CloudInit  VMCloudInit `json:"cloud_init"`
}

// nonGPUNodes is the rotation pool for round-robin placement: every
// micro-* node except micro-07-gpu (which is reserved for GPU work).
var nonGPUNodes = []string{
	"micro-01", "micro-02", "micro-03", "micro-04",
	"micro-05", "micro-06", "micro-08", "micro-09",
}

// templateData carries everything the HCL template needs in one struct
// so the template body itself stays purely declarative.
type templateData struct {
	ProjectName  string
	TeamCount    int
	Topology     CustomTopology
	NonGPUNodes  []string
}

const mainTfTemplate = `variable "project_name" {
  type    = string
  default = "{{.ProjectName}}"
}

variable "team_count" {
  type    = number
  default = {{.TeamCount}}
}

variable "guac_subnet_octet" {
  type        = number
  default     = 1
  description = "Third octet for guac subnet (e.g. 1 = 10.0.1.0/24)"
}

##############################################################
# PROJECT
##############################################################
resource "lxd_project" "project" {
  name        = var.project_name
  description = "${var.project_name} (range-studio)"
  config = {
    "features.storage.volumes" = true
    "features.images"          = false
    "features.profiles"        = false
    "features.storage.buckets" = true
    "features.networks"        = false
  }
}

data "lxd_project" "proj" {
  name       = var.project_name
  depends_on = [lxd_project.project]
}

##############################################################
# NETWORKS
##############################################################
resource "lxd_network" "team_wan" {
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team-wan"
  type    = "ovn"
  config = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [data.lxd_project.proj]
}

resource "lxd_network" "salt_lan" {
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-salt-lan"
  type    = "ovn"
  config = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [data.lxd_project.proj]
}
{{range .Topology.Networks}}
resource "lxd_network" "team_{{sanitize .Name}}" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team${count.index + 1}-{{.Name}}"
  type    = "ovn"
  config = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [data.lxd_project.proj]
}
{{end}}

locals {
  non_gpu_nodes = [{{range $i, $n := .NonGPUNodes}}{{if $i}}, {{end}}"{{$n}}"{{end}}]
{{range .Topology.Networks}}
  team_{{sanitize .Name}}_names = [for n in lxd_network.team_{{sanitize .Name}} : n.name]
{{- end}}
}
{{if .Topology.IncludeProjFW}}
##############################################################
# PROJECT FIREWALL (OpenWRT)
##############################################################
resource "lxd_instance" "project_fw" {
  project     = data.lxd_project.proj.name
  name        = "${var.project_name}-project-fw"
  description = "project firewall"
  type        = "container"
  image       = "{{or .Topology.ProjectFW.Image "openwrt-project"}}"
  profiles    = ["{{or .Topology.ProjectFW.Profile "pfsense"}}"]

  device {
    name = "eth0"
    type = "nic"
    properties = {
      network = "{{or .Topology.ProjectFW.WANNetwork "CLASS_WAN"}}"
    }
  }
  device {
    name = "eth1"
    type = "nic"
    properties = {
      network = lxd_network.team_wan.name
    }
  }
  device {
    name = "eth2"
    type = "nic"
    properties = {
      network = lxd_network.salt_lan.name
    }
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        eth0:
          dhcp4: true
      EOF
  }

  timeouts = {
    create = "2m"
    delete = "2m"
  }
  depends_on = [data.lxd_project.proj, lxd_network.team_wan, lxd_network.salt_lan]
}
{{end}}
{{if .Topology.IncludeSalt}}
##############################################################
# SALT MASTER
##############################################################
resource "lxd_instance" "project_salt" {
  project     = data.lxd_project.proj.name
  name        = "${var.project_name}-salt"
  description = "salt master"
  type        = "virtual-machine"
  image       = "salt-master-new"
  profiles    = ["Salt-master"]

  device {
    name = "eth0"
    type = "nic"
    properties = { network = lxd_network.salt_lan.name }
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses:
            - 172.31.31.2/24
          routes:
            - to: default
              via: 172.31.31.1
          nameservers:
            addresses: [172.31.31.1]
      EOF
  }

  depends_on = [lxd_network.salt_lan]
  timeouts = {
    create = "15m"
    start  = "15m"
  }
}
{{end}}

##############################################################
# TEAM FIREWALL (OpenWRT)
##############################################################
resource "lxd_instance" "team_fw" {
  count       = var.team_count
  project     = data.lxd_project.proj.name
  name        = "${var.project_name}-team${count.index + 1}-fw"
  description = "team${count.index + 1} firewall"
  type        = "container"
  image       = "{{or .Topology.TeamFirewall.Image "openwrt-team-new"}}"
  profiles    = ["{{or .Topology.TeamFirewall.Profile "pfsense"}}"]

  device {
    name = "eth0"
    type = "nic"
    properties = {
      network = lxd_network.team_wan.name
    }
  }
{{range $i, $intf := .Topology.TeamFirewall.Interfaces}}
  device {
    name = "{{$intf.EthName}}"
    type = "nic"
    properties = {
      network = lxd_network.team_{{sanitize $intf.Network}}[count.index].name
    }
  }
{{end}}
  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        eth0:
          dhcp4: true
{{- range .Topology.TeamFirewall.Interfaces}}
        {{.EthName}}:
          x-openwrt-name: {{.Network}}
          dhcp4: false
          addresses:
            - {{.IP}}
          nameservers:
            addresses: [{{trimCIDR .IP}}]
{{- end}}
      EOF
  }

  timeouts = {
    create = "2m"
    delete = "2m"
  }
  depends_on = [data.lxd_project.proj, lxd_network.team_wan]
}

################################################################
# TEAM VMS
################################################################
{{range $vm := .Topology.VMs}}
resource "lxd_instance" "team_{{sanitize $vm.Name}}" {
  count       = var.team_count
  project     = data.lxd_project.proj.name
  name        = "${var.project_name}-team${count.index + 1}-{{$vm.Name}}"
  description = "team${count.index + 1} {{$vm.Name}}"
  type        = "{{or $vm.Type "virtual-machine"}}"
  image       = "{{$vm.Image}}"
  profiles    = ["{{$vm.Profile}}"]
  running     = {{if $vm.Running}}true{{else}}false{{end}}
{{range $i, $net := $vm.Networks}}
  device {
    name = "eth{{$i}}"
    type = "nic"
    properties = {
{{- if eq $net "guac_wan"}}
      network = lxd_network.guac_wan.name
{{- else}}
      network = lxd_network.team_{{sanitize $net}}[count.index].name
{{- end}}
    }
  }
{{end}}{{if $vm.DiskSize}}
  device {
    name = "root"
    type = "disk"
    properties = {
      path = "/"
      pool = "local"
      size = "{{$vm.DiskSize}}"
    }
  }
{{end}}
  config = {
{{- if $vm.CPU}}
    "limits.cpu"    = "{{$vm.CPU}}"
{{- end}}
{{- if $vm.Memory}}
    "limits.memory" = "{{$vm.Memory}}"
{{- end}}
{{- if not $vm.SecureBoot}}
    "security.secureboot" = "false"
{{- end}}
{{- if $vm.CSM}}
    "security.csm" = "true"
{{- end}}
{{- if or $vm.CloudInit.SwapSize $vm.CloudInit.SSHPwAuth $vm.CloudInit.PackageUpdate $vm.CloudInit.RunCmds}}
    "cloud-init.user-data" = <<-EOF
      #cloud-config
{{- if $vm.CloudInit.SwapSize}}
      swap:
        filename: /swapfile
        size: {{$vm.CloudInit.SwapSize}}
        maxsize: {{$vm.CloudInit.SwapSize}}
{{- end}}
{{- if $vm.CloudInit.SSHPwAuth}}
      ssh_pwauth: true
{{- end}}
{{- if $vm.CloudInit.PackageUpdate}}
      package_update: true
{{- end}}
{{- if $vm.CloudInit.RunCmds}}
      runcmd:
{{- range $vm.CloudInit.RunCmds}}
        - {{.}}
{{- end}}
{{- end}}
      EOF
{{- end}}
{{- if or $vm.NetIPs (hasGuacWan $vm.Networks)}}
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
{{- range $i, $net := $vm.Networks}}
{{- if eq $net "guac_wan"}}
{{- $nip := findNetIp $vm.NetIPs "guac_wan"}}
{{- if $nip.IP}}
        enp{{add $i 5}}s0:
          dhcp4: false
          addresses:
            - {{$nip.IP}}
{{- else}}
        enp{{add $i 5}}s0:
          dhcp4: false
          addresses:
            - 10.0.$var.guac_subnet_octet}.${3 + count.index}/16
          routes:
            - to: default
              via: 10.0.0.1
          nameservers:
            addresses: [10.0.0.1]
{{- else}}
{{- $nip := findNetIp $vm.NetIPs $net}}
{{- if $nip.IP}}
        enp{{add $i 5}}s0:
          dhcp4: false
          addresses:
            - {{$nip.IP}}
          routes:
            - to: default
              via: {{gateway $nip.IP}}
          nameservers:
            addresses: [{{gateway $nip.IP}}]
{{- end}}
{{- end}}
{{- end}}
      EOF
{{- end}}
  }
{{placement $vm $.Topology.PlacementMode $.Topology.PinnedTarget}}
  depends_on = [lxd_instance.team_fw]
}
{{end}}
`

// GenerateMainTF executes the text/template to output raw HCL.
func GenerateMainTF(projectName string, teamCount int, topo CustomTopology) (string, error) {
	if teamCount < 1 {
		teamCount = 1
	}
	if topo.NetworkMode == "single" {
		teamCount = 1
	}

	funcs := template.FuncMap{
		"trimCIDR": func(ip string) string {
			parts := strings.Split(ip, "/")
			return parts[0]
		},
		"sanitize": func(s string) string {
			s = strings.ReplaceAll(s, "-", "_")
			s = strings.ReplaceAll(s, " ", "_")
			s = strings.ReplaceAll(s, ".", "_")
			return strings.ToLower(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"gateway": func(cidr string) string {
			// Extract IP, replace last octet with .1 for gateway
			ip := strings.Split(cidr, "/")[0]
			parts := strings.Split(ip, ".")
			if len(parts) == 4 {
				parts[3] = "1"
				return strings.Join(parts, ".")
			}
			return ip
		},
		"hasGuacWan": func(networks []string) bool {
			for _, n := range networks {
				if n == "guac_wan" {
					return true
				}
			}
			return false
		},
		"findNetIp": func(netIps []VMNetIP, network string) VMNetIP {
			for _, nip := range netIps {
				if nip.Network == network {
					return nip
				}
			}
			return VMNetIP{Network: network, IP: ""}
		},
		"placement": func(vm VMDef, mode, pinned string) string {
			// Per-VM override wins over project default.
			if vm.NeedsGPU {
				return `  target = "@micro-07-gpu"`
			}
			t := vm.Target
			if t == "" {
				t = mode
			}
			switch t {
			case "default", "":
				return "  # target left unset (cluster default)"
			default:
				if strings.HasPrefix(t, "micro-") || strings.HasPrefix(t, "@") {
					tgt := strings.TrimPrefix(t, "@")
					return fmt.Sprintf(`  target = "@%s"`, tgt)
				}
				if t == "pinned" && pinned != "" {
					return fmt.Sprintf(`  target = "@%s"`, strings.TrimPrefix(pinned, "@"))
				}
			}
			return "  # target left unset (cluster default)"
		},
	}

	tmpl, err := template.New("maintf").Funcs(funcs).Parse(mainTfTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	data := templateData{
		ProjectName: projectName,
		TeamCount:   teamCount,
		Topology:    topo,
		NonGPUNodes: nonGPUNodes,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
