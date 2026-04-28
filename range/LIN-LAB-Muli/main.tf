variable "project_name" {
  type        = string
  default     = "DCIGS"
  }

variable "team_count" {
  type        = number
  default     = 35
}

##############################################################
# PROJECT
##############################################################
resource "lxd_project" "project" {
  name    = var.project_name
  description = "Linux Security DCIG"
  config = {
    "features.storage.volumes" = true
    "features.images"          = false
    "features.profiles"        = false
    "features.storage.buckets" = true
    "features.networks"        = false
  }
}

data "lxd_project" "proj" {
  name   = var.project_name
  depends_on = [ lxd_project.project ]
}

##############################################################
# NETWORKS
##############################################################

resource "lxd_network" "team_lan" {
  count = var.team_count
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-team${count.index + 1}-lan"
  type     = "ovn"
  config   = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }

  depends_on = [ data.lxd_project.proj, lxd_network.team_wan ]

}

resource "lxd_network" "team_wan" {

  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-team-wan"
  type     = "ovn"
  config   = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [ data.lxd_project.proj ]
}

resource "lxd_network" "salt_lan" {
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-salt-lan"
  type     = "ovn"
  config   = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [ data.lxd_project.proj, lxd_network.team_wan ]
}

locals {
  windows_images = ["windows-10-base", "windows-2019-base"]
  project_fw_net = concat(["DCIG_WAN", lxd_network.team_wan.name, lxd_network.salt_lan.name])

  shared_wan = lxd_network.team_wan.name
  team_lan_names = [for n in lxd_network.team_lan : n.name]

  guac_net = ["GUAC_WAN", lxd_network.salt_lan.name]
  guac_wan = "GUAC_WAN"
}

###############################################################
# Firewall
###############################################################
resource "lxd_instance" "project_fw" {
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-openwrt"
  description = "project-openwrt"
  image  = "openwrt-project"
  type  = "container"
  profiles = ["pfsense"]
  
  dynamic "device" {
    for_each = toset([for i in range(length(local.project_fw_net)) : i])
    content {
      name = "eth${device.value}"
      type = "nic"
      properties = {
        network = local.project_fw_net[device.value]
      }
    }
    
  }
  target = "@default"

  timeouts = {
    create = "2m"
    delete = "2m"
  }

  depends_on = [ data.lxd_project.proj, lxd_network.team_wan, lxd_network.salt_lan ]
}

##############################################################
# SALT
##############################################################

resource "lxd_instance" "project_salt" {
  project = data.lxd_project.proj.name
  name = "${var.project_name}-salt"
  description = "project-salt"
  type = "virtual-machine"
  image = "salt-master-new"
  profiles = ["Salt-master"]

  dynamic "device" {
    for_each = toset([lxd_network.salt_lan.name])
    content {
      name = "eth0"
      type = "nic"
      properties = { network = device.value }
    }
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
            - to: 10.8.11.201
              via: 172.31.31.1
            - to: default
              via: 172.31.31.1
          nameservers:
            addresses: [172.31.31.1]
      EOF
  }
  target = "@Cluster-C"


  depends_on = [ lxd_network.salt_lan ]
  timeouts = {
    create = "15m"
    start  = "15m"
  }
  
}

resource "lxd_instance" "project_guac" {
  project = data.lxd_project.proj.name
  name   = "${var.project_name}-guac-project"
  description = "project-guac"
  type = "virtual-machine"
  image = "guac-xfce4-v02"
  profiles = ["guac-linux"]
  

  dynamic "device" {
    for_each = toset([for i in range(length(local.guac_net)) : i])
    content {
      name = "eth-${device.value}"
      type = "nic"
      properties = {
        network = local.guac_net[device.value]
      }
    }
    
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 10.0.${var.guac_subnet_octet}.2/16
          routes:
            - to: default
              via: 10.0.0.1
          nameservers:
            addresses: [10.0.0.1]
        enp6s0:
          dhcp4: false
          addresses:
            - 172.31.31.3/24
          routes:
            - to: default
              via: 172.31.31.1
          nameservers:
            addresses: [172.31.31.1]
      EOF
  }

  target = "@default"


  depends_on = [ lxd_network.salt_lan, lxd_instance.project_fw ]
}

##############################################################
# Team FW
##############################################################
resource "lxd_instance" "team_fw" {
  count   = var.team_count 
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team${count.index + 1}-fw"
  description = "team${count.index + 1}-fw"
  type    = "container"
  image   = "openwrt-team-new"
  profiles = ["pfsense"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.shared_wan,
        local.team_lan_names[count.index]
      ]) : i => net
    }
    content {
      name = "eth${device.key}"
      type = "nic"
      properties = {
        network = device.value
      }
    }
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        eth0:
          dhcp4: true
        eth1:
          dhcp4: false
          addresses:
            - 192.168.${1 + count.index}.1/24
          routes:
            - to: default
              via: 192.168.${1 + count.index}.1
          nameservers:
            addresses: [192.168.${1 + count.index}.1]
          EOF
  }

  timeouts = {
    create = "2m"
    delete = "2m"
  }

  target = "@default"

  depends_on = [ lxd_instance.project_fw ]
}

################################################################
# LAN VMS
################################################################

variable "lan_image" {
  type        = list(string)
  default     = ["guac-xfce4-v02", "guac-xfce4-v02", "guac-xfce4-v02"]
}

variable "lan_name" {
  type        = list(string)
  default     = ["ubuntu1", "ubuntu2", "ubuntu3"]

}

resource "lxd_instance" "lan_linux" {
    count = var.team_count * length(var.lan_name)
    project  = data.lxd_project.proj.name
    name = "${var.project_name}-team${floor(count.index / length(var.lan_name)) + 1}-${var.lan_name[count.index % length(var.lan_name)]}"
    description = "team${floor(count.index / length(var.lan_name)) + 1}-${var.lan_name[count.index % length(var.lan_name)]}"
    type = var.lan_name[count.index % length(var.lan_name)] == "mint" ? "container" : "virtual-machine"
    image = var.lan_image[count.index % length(var.lan_image)]
    profiles = contains(local.windows_images, var.lan_image[count.index % length(var.lan_image)]) ? ["default-windows"] : ["guac-linux"]


    device {
      name = "eth0"
      type = "nic"
      properties = {
        network = lxd_network.team_lan[floor(count.index / length(var.lan_name))].name
      }
    }

    config = {
      "cloud-init.network-config" = <<-EOF
        version: 2
        ethernets:
          enp5s0:
            dhcp4: false
            addresses:
              - 192.168.${1 + floor(count.index / length(var.lan_name))}.${2 + (count.index % length(var.lan_name))}/24
            nameservers:
              addresses: [192.168.${1 + floor(count.index / length(var.lan_name))}.1]
            routes:
              - to: default
                via: 192.168.${1 + floor(count.index / length(var.lan_name))}.1
        EOF
    }

    target = "@default"

    depends_on = [ lxd_instance.team_fw ]
}

variable "guac_name" {
  type=string
  default="Guac"
}

variable "guac_subnet_octet" {
  type        = number
  default     = 1
  description = "Third octet for guac subnet (e.g., 1 = 10.0.1.0/24)"
}

resource "lxd_instance" "guac" {
  count = var.team_count
  project  = data.lxd_project.proj.name
  name = "${var.project_name}-team${count.index + 1}-${var.guac_name}"
  description = "team${count.index + 1}-${var.guac_name}"
  type = "virtual-machine"
  image = "guac-xfce4-v02"
  profiles = ["guac-linux"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.guac_wan,
        local.team_lan_names[count.index]
      ]) : i => net
    }
    content {
      name = "eth${device.key}"
      type = "nic"
      properties = {
        network = device.value
      }
    }
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 10.0.${var.guac_subnet_octet}.${4 + count.index}/16
          routes:
            - to: default
              via: 10.0.0.1
          nameservers:
            addresses: [10.0.0.1]
        enp6s0:
          dhcp4: false
          addresses:
            - 192.168.${1 + count.index}.${10 + count.index}/24
          routes:
            - to: 172.31.31.2
              via: 192.168.${1 + count.index}.1
      EOF
  }

  target = "@default"

  depends_on = [ lxd_network.team_lan, lxd_instance.project_fw, lxd_instance.team_fw ]

}