variable "project_name" {
  type    = string
  default = "ResearchProj"
}

variable "team_count" {
  type    = number
  default = 4
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

resource "lxd_network" "team_lan" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team${count.index + 1}-lan"
  type    = "ovn"
  config = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [data.lxd_project.proj]
}

resource "lxd_network" "team_dmz" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team${count.index + 1}-dmz"
  type    = "ovn"
  config = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [data.lxd_project.proj]
}


locals {
  non_gpu_nodes = ["micro-01", "micro-02", "micro-03", "micro-04", "micro-05", "micro-06", "micro-08", "micro-09"]

  team_lan_names = [for n in lxd_network.team_lan : n.name]
  team_dmz_names = [for n in lxd_network.team_dmz : n.name]
}

##############################################################
# PROJECT FIREWALL (OpenWRT)
##############################################################
resource "lxd_instance" "project_fw" {
  project     = data.lxd_project.proj.name
  name        = "${var.project_name}-project-fw"
  description = "project firewall"
  type        = "container"
  image       = "openwrt-project"
  profiles    = ["pfsense"]

  device {
    name = "eth0"
    type = "nic"
    properties = {
      network = "CLASS_WAN"
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

  device {
    name = "eth3"
    type = "nic"
    properties = {
      network = "dmz"
    }
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        eth0:
          dhcp4: true
        eth3:
          dhcp4: false
          addresses:
            - 10.10.0.1/24
          nameservers:
            addresses: [10.10.0.1]
      EOF
  }

  timeouts = {
    create = "2m"
    delete = "2m"
  }
  depends_on = [data.lxd_project.proj, lxd_network.team_wan, lxd_network.salt_lan]
}


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


##############################################################
# TEAM FIREWALL (OpenWRT)
##############################################################
resource "lxd_instance" "team_fw" {
  count       = var.team_count
  project     = data.lxd_project.proj.name
  name        = "${var.project_name}-team${count.index + 1}-fw"
  description = "team${count.index + 1} firewall"
  type        = "container"
  image       = "openwrt-team-new"
  profiles    = ["pfsense"]

  device {
    name = "eth0"
    type = "nic"
    properties = {
      network = lxd_network.team_wan.name
    }
  }

  device {
    name = "eth1"
    type = "nic"
    properties = {
      network = lxd_network.team_lan[count.index].name
    }
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        eth0:
          dhcp4: true
        eth1:
          x-openwrt-name: lan
          dhcp4: false
          addresses:
            - 192.168.1.1/24
          nameservers:
            addresses: [192.168.1.1]
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

resource "lxd_instance" "team_jumpbox" {
  count       = var.team_count
  project     = data.lxd_project.proj.name
  name        = "${var.project_name}-team${count.index + 1}-Jumpbox"
  description = "team${count.index + 1} Jumpbox"
  type        = "virtual-machine"
  image       = "ubuntu-2204-local"
  profiles    = ["guac-linux"]

  device {
    name = "eth0"
    type = "nic"
    properties = {
      network = lxd_network.team_lan[count.index].name
    }
  }

  device {
    name = "root"
    type = "disk"
    properties = {
      path = "/"
      pool = "local"
      size = "50GiB"
    }
  }

  config = {
    "limits.cpu"    = "2"
    "limits.memory" = "4GiB"
  }
  target = "@${local.non_gpu_nodes[count.index % length(local.non_gpu_nodes)]}"
  depends_on = [lxd_instance.team_fw]
}

