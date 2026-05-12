variable "project_name" {
  type        = string
  default     = "CSC-3570-001"
}

variable "team_count" {
  type        = number
  default     = 2
}

##############################################################
# PROJECT
##############################################################
resource "lxd_project" "project" {
  name    = var.project_name
  description = "Project for CSC-3570 IT Security course"
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

resource "lxd_network" "team_dmz" {
  count = var.team_count
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-team${count.index + 1}-dmz"
  type     = "ovn"
  config   = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [ data.lxd_project.proj, lxd_network.team_lan ]

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
  project_fw_net = concat(["CLASS_WAN", lxd_network.team_wan.name, lxd_network.salt_lan.name])

  shared_wan = lxd_network.team_wan.name
  team_lan_names = [for n in lxd_network.team_lan : n.name]
  team_dmz_names = [for n in lxd_network.team_dmz : n.name]
  guac_net = ["GUAC_WAN", lxd_network.salt_lan.name]
  guac_wan = "GUAC_WAN"
}

###############################################################
# SALT
###############################################################
resource "lxd_instance" "project_fw" {
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-project-fw"
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


  depends_on = [ lxd_network.salt_lan ]
  timeouts = {
    create = "15m"
    start  = "15m"
  }
  
}

resource "lxd_instance" "project_guac_salt" {
  project = data.lxd_project.proj.name
  name   = "${var.project_name}-guac-salt-project"
  description = "project-guac-salt"
  type = "virtual-machine"
  image = "guac-xfce4-v01"
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
            - to: 172.31.31.1
              via: 172.31.31.1
          nameservers:
            addresses: [172.31.31.1]
      EOF
  }


  depends_on = [ lxd_network.salt_lan, lxd_instance.project_fw ]
}



##############################################################
# Team FW
##############################################################
resource "lxd_instance" "team_fw" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name    = "team${count.index + 1}-fw"
  description = "team${count.index + 1}-fw"
  type    = "container"
  image   = "openwrt-team-new"
  profiles = ["pfsense"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.shared_wan,
        local.team_lan_names[count.index],
        local.team_dmz_names[count.index]
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
          x-openwrt-name: lan
          dhcp4: false
          addresses:
            - 192.168.${1 + count.index}.1/24
          routes:
            - to: default
              via: 192.168.${1 + count.index}.1
          nameservers:
            addresses: [192.168.${1 + count.index}.1]
        eth2:
          x-openwrt-name: dmz
          dhcp4: false
          addresses:
            - 192.168.${1 + count.index}.129/25
          routes:
            - to: 192.168.${1 + count.index}.129
              via: 192.168.${1 + count.index}.129
          nameservers:
            addresses: [192.168.${1 + count.index}.129]
      EOF
  }

  timeouts = {
    create = "2m"
    delete = "2m"
  }

  target = "@default"

  depends_on = [ data.lxd_project.proj, lxd_network.team_wan, lxd_network.team_lan, lxd_network.team_dmz ]
}

################################################################
# LAN VMS
################################################################

variable "lan_vm_name" {
  type        = list(string)
  default     = ["win19", "win10"]
}

variable "lan_vms" {
  type        = list(string)
  default     = ["windows-2019-base", "windows-10-base"]
}

resource "lxd_instance" "lan_vms" {
  count   = var.team_count * length(var.lan_vms)
  project = data.lxd_project.proj.name
  name    = "team${floor(count.index / length(var.lan_vms)) + 1}-${var.lan_vm_name[count.index % length(var.lan_vms)]}"
  description = "team${floor(count.index / length(var.lan_vms)) + 1}-${var.lan_vm_name[count.index % length(var.lan_vms)]}"
  type    = "virtual-machine"
  image   = var.lan_vms[count.index % length(var.lan_vms)]
  profiles = ["default-windows"]

  running = false
  dynamic "device" {
    for_each = toset([lxd_network.team_lan[floor(count.index / length(var.lan_vms))].name])
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
        eth0:
          dhcp4: false
          addresses: 
            - 192.168.${1 + floor(count.index / length(var.lan_vms))}.${2 + (count.index % length(var.lan_vms))}/24
          routes:
            - to: default
              via: 192.168.${1 + floor(count.index / length(var.lan_vms))}.1
            - to: 172.31.31.2
              via: 192.168.${1 + floor(count.index / length(var.lan_vms))}.1
          nameservers:
            addresses: [192.168.${1 + floor(count.index / length(var.lan_vms))}.1]
      EOF
    
  }

  target = "@default"

  depends_on = [ lxd_instance.dmz_ubuntu ]
}

variable "guac_subnet_octet" {
  type        = number
  default     = 1
  description = "Third octet for guac subnet (e.g., 1 = 10.0.1.0/24)"
}

#################################################################
# DMZ VMS
#################################################################
resource "lxd_instance" "dmz_ubuntu" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${count.index + 1}-dmz-ubuntu"
  description = "team${count.index + 1}-dmz-ubuntu"
  type    = "virtual-machine"
  image   = "guac-xfce4-v02"
  profiles = ["guac-linux"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_dmz_names[count.index]
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
"cloud-init.user-data" = <<-EOF
      #cloud-config
      runcmd:
        - echo "team${count.index + 1}-dmz-ubuntu localhost" | tee -a /etc/hosts
        - ufw --force reset
      EOF

    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 192.168.${1 + count.index}.130/25
          routes:
            - to: default
              via: 192.168.${1 + count.index}.129
            - to: 172.31.31.2
              via: 192.168.${1 + count.index}.129
          nameservers:
            addresses: [192.168.${1 + count.index}.129]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

variable "dmz_vm_name" {
  type        = list(string)
  default     = ["Jumpbox1", "Jumpbox2"]
}

variable "dmz_vms" {
  type        = list(string)
  default     = ["guac-xfce4-v02", "guac-xfce4-v02"]
}

resource "lxd_instance" "dmz_jumpbox" {
  count = var.team_count * length(var.dmz_vms)
  project = data.lxd_project.proj.name
  name = "team${floor(count.index / length(var.dmz_vms)) + 1}-${var.dmz_vm_name[count.index % length(var.dmz_vms)]}"
  description = "team${floor(count.index / length(var.dmz_vms)) + 1}-${var.dmz_vm_name[count.index % length(var.dmz_vms)]}"
  type    = "virtual-machine"
  image   = var.dmz_vms[count.index % length(var.dmz_vms)]
  profiles = ["medium-linux-jumpbox"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.guac_wan,
        local.team_dmz_names[floor(count.index / length(var.dmz_vms))]
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
    "cloud-init.user-data" = <<-EOF
      #cloud-config
      runcmd:
        - echo "team${floor(count.index / length(var.dmz_vm_name)) + 1}-${var.dmz_vm_name[count.index % length(var.dmz_vm_name)]} localhost" | tee -a /etc/hosts
      EOF
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 10.0.${var.guac_subnet_octet}.${3 + count.index}/16
          routes:
            - to: default
              via: 10.0.0.1
          nameservers:
            addresses: [10.0.0.1]
        enp6s0:
          dhcp4: false
          addresses: 
            - 192.168.${1 + floor(count.index / length(var.dmz_vms))}.${201 + (count.index % length(var.dmz_vms))}/25
          routes:
            - to: default
              via: 192.168.${1 + floor(count.index / length(var.dmz_vms))}.129
          nameservers:
            addresses: [192.168.${1 + floor(count.index / length(var.dmz_vms))}.129]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.dmz_ubuntu, lxd_network.team_dmz ]
}
