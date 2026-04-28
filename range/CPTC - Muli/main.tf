variable "project_name" {
  type        = string
  default     = "CPTC-Mock"
}

variable "team_count" {
  type        = number
  default     = 6
}

##############################################################
# PROJECT
##############################################################
resource "lxd_project" "project" {
  name    = var.project_name
  description = "CPTC MOCK"
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

  depends_on = [ data.lxd_project.proj ]

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

resource "lxd_network" "team_business" {
  count = var.team_count
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-team${count.index + 1}-business"
  type     = "ovn"
  config   = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
  depends_on = [ data.lxd_project.proj ]

}

resource "lxd_network" "team_sensitive" {
  count = var.team_count
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-team${count.index + 1}-sensitive"
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
  depends_on = [ data.lxd_project.proj ]
}

locals {
  windows_images = ["windows-10-base", "windows-2019-base"]
  project_fw_net = concat(["CLASS_WAN", lxd_network.team_wan.name, lxd_network.salt_lan.name])

  shared_wan = lxd_network.team_wan.name
  team_dmz_names = [for n in lxd_network.team_dmz : n.name]
  team_business_names = [for n in lxd_network.team_business : n.name]
  team_sensitive_names = [for n in lxd_network.team_sensitive : n.name]

  guac_net = ["GUAC_WAN", lxd_network.salt_lan.name]
  guac_wan = "GUAC_WAN"
}

variable "guac_subnet_octet" {
  type        = number
  default     = 1
  description = "Third octet for guac subnet (e.g., 1 = 10.0.1.0/24)"
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
  name   = "${var.project_name}-project-guac"
  description = "project-guac-salt"
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
        local.team_dmz_names[count.index],
        local.team_business_names[count.index],
        local.team_sensitive_names[count.index]
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
            - 10.10.0.1/24
          routes:
            - to: default
              via: 10.10.0.1
          nameservers:
            addresses: [10.10.0.1]
        eth2:
          x-openwrt-name: dmz
          dhcp4: false
          addresses:
            - 172.16.0.1/24
          routes:
            - to: 172.16.0.1
              via: 172.16.0.1
          nameservers:
            addresses: [172.16.0.1]
        eth3:
          x-openwrt-name: sensitive
          dhcp4: false
          addresses:
            - 192.168.5.1/24
          routes:
            - to: 192.168.5.1
              via: 192.168.5.1
          nameservers:
            addresses: [192.168.5.1]
      EOF
  }

  timeouts = {
    create = "2m"
    delete = "2m"
  }

  target = "@default"

  depends_on = [ data.lxd_project.proj, lxd_network.team_wan, lxd_network.team_business, lxd_network.team_dmz, lxd_network.team_sensitive ]
}


################################################################
# DMZ VMS
################################################################

variable "dmz_vm_lin_name" {
  type        = list(string)
  default     = ["Jumpbox1", "Jumpbox2", "Jumpbox3", "Jumpbox4", "Jumpbox5", "Jumpbox6"]
}

variable "dmz_vms_lin" {
  type        = list(string)
  default     = ["parrot-v2", "parrot-v2", "parrot-v2", "parrot-v2", "parrot-v2", "parrot-v2"]
}

resource "lxd_instance" "dmz_lin_vms" {
  count   = var.team_count * length(var.dmz_vms_lin)
  project = data.lxd_project.proj.name
  name    = "team${format("%02d", floor(count.index / length(var.dmz_vms_lin)) + 1)}-${var.dmz_vm_lin_name[count.index % length(var.dmz_vms_lin)]}"
  description = "team${format("%02d", floor(count.index / length(var.dmz_vms_lin)) + 1)}-${var.dmz_vm_lin_name[count.index % length(var.dmz_vms_lin)]}"
  type    = "virtual-machine"
  image   = var.dmz_vms_lin[count.index % length(var.dmz_vms_lin)]
  profiles = ["guac-linux"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.guac_wan,
        local.team_dmz_names[floor(count.index / length(var.dmz_vms_lin))]
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
    "security.secureboot" = "false"
    "security.csm" = "true"
    "cloud-init.user-data" = <<-EOF
      #cloud-config
      swap:
        filename: /swapfile
        size: 4G
        maxsize: 4G

      ssh_pwauth: true
      package_update: true
      runcmd:
        - echo "team${format("%02d", floor(count.index / length(var.dmz_vms_lin)) + 1)}-${var.dmz_vm_lin_name[count.index % length(var.dmz_vms_lin)]} localhost" | tee -a /etc/hosts
      EOF
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 10.0.${var.guac_subnet_octet}.${3 + count.index}/16
          nameservers:
            addresses: [10.0.0.1]
        enp6s0:
          dhcp4: false
          addresses: 
            - 10.10.0.${50 + floor(count.index % length(var.dmz_vms_lin))}/24
          routes:
            - to: 0.0.0.0/0
              via: 10.10.0.1
            - to: 172.31.31.2
              via: 10.10.0.1
          nameservers:
            addresses: [10.10.0.1]
      EOF
    
  }

  target = "@default"

  #depends_on = [ lxd_instance.team_fw ]
}

variable "dmz_vm_win_name" {
  type        = list(string)
  default     = ["Win01", "Win02", "Win03", "Win04", "Win05", "Win06"]
}

variable "dmz_vms_win" {
  type        = list(string)
  default     = ["windows-10-base", "windows-10-base", "windows-10-base", "windows-10-base", "windows-10-base", "windows-10-base"]
}

resource "lxd_instance" "dmz_win_vms" {
  count   = var.team_count * length(var.dmz_vms_win)
  project = data.lxd_project.proj.name
  name    = "team${format("%02d", floor(count.index / length(var.dmz_vms_win)) + 1)}-${var.dmz_vm_win_name[count.index % length(var.dmz_vms_win)]}"
  description = "team${format("%02d", floor(count.index / length(var.dmz_vms_win)) + 1)}-${var.dmz_vm_win_name[count.index % length(var.dmz_vms_win)]}"
  type    = "virtual-machine"
  image   = var.dmz_vms_win[count.index % length(var.dmz_vms_win)]
  profiles = ["default-windows"]

  running = false
  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.guac_wan,
        local.team_dmz_names[floor(count.index / length(var.dmz_vms_lin))]
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
          dhcp4: false
          addresses: 
            - 10.0.${var.guac_subnet_octet}.${3 + count.index + (var.team_count * length(var.dmz_vms_lin))}/16
          routes:
            - to: default
              via: 10.0.0.1
          nameservers:
            addresses: [10.0.0.1]
        eth1:
          dhcp4: false
          addresses: 
            - 10.10.0.${50 + (count.index % length(var.dmz_vms_win) + (var.team_count * length(var.dmz_vms_lin)))}/24
          routes:
            - to: default
              via: 10.10.0.1
            - to: 172.31.31.2
              via: 10.10.0.1
          nameservers:
            addresses: [10.10.0.1]
      EOF
    
  }

  target = "micro-01"

  #depends_on = [ lxd_instance.dmz_lin_vms ]
}

#################################################################
# business VMS
#################################################################
resource "lxd_instance" "dmz_1" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}dmz01"
  description = "team${format("%02d", count.index + 1)}dmz01"
  type    = "virtual-machine"
  image   = "guac-xfce4-v02"
  profiles = ["guac-linux"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_dmz_names[count.index],
        local.team_business_names[count.index]
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
      swap:
        filename: /swapfile
        size: 4G
        maxsize: 4G

      ssh_pwauth: true
      package_update: true
      runcmd:
        - echo "team${format("%02d", count.index + 1)}dmz01 localhost" | tee -a /etc/hosts
        - ufw --force reset
      EOF

    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 10.10.0.8/24
          routes:
            - to: default
              via: 10.10.0.1
            - to: 172.31.31.2
              via: 10.10.0.1
          nameservers:
            addresses: [10.10.0.1]
        enp6s0:
          dhcp4: false
          addresses: 
            - 172.16.0.5/24
          routes:
            - to: default
              via: 172.16.0.1
          nameservers:
            addresses: [172.16.0.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

resource "lxd_instance" "dmz_control" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}control01"
  description = "team${format("%02d", count.index + 1)}control01"
  type    = "virtual-machine"
  image   = "guac-xfce4-v02"
  profiles = ["guac-linux"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_business_names[count.index]
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
      swap:
        filename: /swapfile
        size: 4G
        maxsize: 4G

      ssh_pwauth: true
      package_update: true
      runcmd:
        - echo "team${format("%02d", count.index + 1)}control01 localhost" | tee -a /etc/hosts
        - ufw --force reset
      EOF

    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 172.16.0.5/24
          routes:
            - to: default
              via: 172.16.0.1
            - to: 172.31.31.2
              via: 172.16.0.1
          nameservers:
            addresses: [172.16.0.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

resource "lxd_instance" "bus_cert" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}cert01"
  description = "team${format("%02d", count.index + 1)}cert01"
  type    = "virtual-machine"
  image   = "windows-2019-base"
  profiles = ["default-windows"]

  running = false
  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_business_names[count.index]
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
          dhcp4: false
          addresses: 
            - 172.16.0.52/24
          routes:
            - to: default
              via: 172.16.0.1
            - to: 172.31.31.2
              via: 172.16.0.1
          nameservers:
            addresses: [172.16.0.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

#################################################################
# boston01 - Ubuntu 22.04
#################################################################
resource "lxd_instance" "boston01" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}boston01"
  description = "team${format("%02d", count.index + 1)}boston01"
  type    = "virtual-machine"
  image   = "guac-xfce4-v02"
  profiles = ["guac-linux"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_business_names[count.index],
        local.team_sensitive_names[count.index]
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
      swap:
        filename: /swapfile
        size: 4G
        maxsize: 4G

      ssh_pwauth: true
      package_update: true
      runcmd:
        - echo "team${format("%02d", count.index + 1)}boston01 localhost" | tee -a /etc/hosts
        - ufw --force reset
      EOF

    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 172.16.0.71/24
          routes:
            - to: default
              via: 172.16.0.1
            - to: 172.31.31.2
              via: 172.16.0.1
          nameservers:
            addresses: [172.16.0.1]
        enp6s0:
          dhcp4: false
          addresses: 
            - 192.168.5.21/24
          routes:
            - to: default
              via: 192.168.5.1
          nameservers:
            addresses: [192.168.5.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

#################################################################
# dc01 - Windows Server 2019
#################################################################
resource "lxd_instance" "dc01" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}dc01"
  description = "team${format("%02d", count.index + 1)}dc01"
  type    = "virtual-machine"
  image   = "windows-2019-base"
  profiles = ["default-windows"]

  running = false

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_business_names[count.index],
        local.team_sensitive_names[count.index]
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
          dhcp4: false
          addresses: 
            - 172.16.0.67/24
          routes:
            - to: default
              via: 172.16.0.1
            - to: 172.31.31.2
              via: 172.16.0.1
          nameservers:
            addresses: [172.16.0.1]
        eth1:
          dhcp4: false
          addresses: 
            - 192.168.5.80/24
          routes:
            - to: default
              via: 192.168.5.1
          nameservers:
            addresses: [192.168.5.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

#################################################################
# dns01 - Windows Server 2019
#################################################################
resource "lxd_instance" "dns01" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}dns01"
  description = "team${format("%02d", count.index + 1)}dns01"
  type    = "virtual-machine"
  image   = "windows-2019-base"
  profiles = ["default-windows"]

  running = false

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_business_names[count.index]
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
          dhcp4: false
          addresses: 
            - 172.16.0.66/24
          routes:
            - to: default
              via: 172.16.0.1
            - to: 172.31.31.2
              via: 172.16.0.1
          nameservers:
            addresses: [172.16.0.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

#################################################################
# dc02 - Windows Server 2019
#################################################################
resource "lxd_instance" "dc02" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}dc02"
  description = "team${format("%02d", count.index + 1)}dc02"
  type    = "virtual-machine"
  image   = "windows-2019-base"
  profiles = ["default-windows"]

  running = false

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_sensitive_names[count.index]
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
          dhcp4: false
          addresses: 
            - 192.168.5.153/24
          routes:
            - to: default
              via: 192.168.5.1
            - to: 172.31.31.2
              via: 192.168.5.1
          nameservers:
            addresses: [192.168.5.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

#################################################################
# silocontroller01 - Ubuntu 22.04
#################################################################
resource "lxd_instance" "silocontroller01" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}silocontroller01"
  description = "team${format("%02d", count.index + 1)}silocontroller01"
  type    = "virtual-machine"
  image   = "guac-xfce4-v02"
  profiles = ["guac-linux"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.team_sensitive_names[count.index]
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
      swap:
        filename: /swapfile
        size: 4G
        maxsize: 4G

      ssh_pwauth: true
      package_update: true
      runcmd:
        - echo "team${format("%02d", count.index + 1)}silocontroller01 localhost" | tee -a /etc/hosts
        - ufw --force reset
      EOF

    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 192.168.5.90/24
          routes:
            - to: default
              via: 192.168.5.1
            - to: 172.31.31.2
              via: 192.168.5.1
          nameservers:
            addresses: [192.168.5.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}

#################################################################
# blackteam - Ubuntu 22.04 (DMZ, Business, Sensitive)
#################################################################
resource "lxd_instance" "blackteam" {
  count   = var.team_count
  project = data.lxd_project.proj.name
  name = "team${format("%02d", count.index + 1)}blackteam"
  description = "team${format("%02d", count.index + 1)}blackteam"
  type    = "virtual-machine"
  image   = "guac-xfce4-v02"
  profiles = ["guac-linux"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.guac_wan,
        local.team_dmz_names[count.index],
        local.team_business_names[count.index],
        local.team_sensitive_names[count.index]
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
      swap:
          filename: /swapfile
          size: 4G
          maxsize: 4G

      ssh_pwauth: true
      package_update: true
      runcmd:
        - echo "team${format("%02d", count.index + 1)}blackteam localhost" | tee -a /etc/hosts
        - ufw --force reset
      EOF

    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses: 
            - 10.0.${var.guac_subnet_octet}.${3 + count.index + (var.team_count * length(var.dmz_vms_lin)) + (var.team_count * length(var.dmz_vms_win))}/16
          routes:
            - to: default
              via: 10.0.${var.guac_subnet_octet}.1
          nameservers:
            addresses: [10.0.${var.guac_subnet_octet}.1]
        enp6s0:
          dhcp4: false
          addresses: 
            - 10.10.0.252/24
          routes:
            - to: 0.0.0.0/0
              via: 10.10.0.1
            - to: 172.31.31.2
              via: 10.10.0.1
          nameservers:
            addresses: [10.10.0.1]
        enp7s0:
          dhcp4: false
          addresses: 
            - 172.16.0.252/24
          routes:
            - to: 0.0.0.0/0
              via: 172.16.0.1
          nameservers:
            addresses: [172.16.0.1]
        enp8s0:
          dhcp4: false
          addresses: 
            - 192.168.5.252/24
          routes:
            - to: 0.0.0.0/0
              via: 192.168.5.1
          nameservers:
            addresses: [192.168.5.1]
      EOF
  }

  target = "@default"

  depends_on = [ lxd_instance.team_fw ]
}
