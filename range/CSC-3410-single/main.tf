variable "project_name" {
  type        = string
  default     = "CSC-3410"
}

variable "team_count" {
  type        = number
  default     = 65
}

#############################################################
# PROJECT
#############################################################
resource "lxd_project" "proj" {
  name        = var.project_name
  description = "${var.project_name}"
  config = {
    "features.storage.volumes" = true
    "features.images"          = false
    "features.profiles"        = false
    "features.storage.buckets" = true
    "features.networks"        = false
  }
}

data "lxd_project" "proj" {
  name   = lxd_project.proj.name
  depends_on = [lxd_project.proj]
}

#############################################################
# NETWORKS
#############################################################
resource "lxd_network" "salt_lan" {
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-salt-lan"
  type    = "ovn"
  config = {
    "bridge.mtu"            = "1500"
    "ipv4.address"        = "none"
    "network"            = "internal_link5"
  }

  depends_on = [ data.lxd_project.proj ]
}

resource "lxd_network" "team_lan" {
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team-lan"
  type    = "ovn"
  config = {
    "bridge.mtu"            = "1500"
    "ipv4.address"        = "none"
    "network"            = "internal_link5"
  }

  depends_on = [data.lxd_project.proj]
  
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

locals {
  project_fw_net = concat(["CLASS_WAN", lxd_network.team_wan.name , lxd_network.salt_lan.name])
  guac_net = ["GUAC_WAN", lxd_network.salt_lan.name]
  guac_wan = "GUAC_WAN"
 }

######################################################################
# SALT
######################################################################
resource "lxd_instance" "project_fw" {
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-openwrt"
  description = "project-openwrt"
  type    = "container"
  image   = "openwrt-project"
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

  depends_on = [data.lxd_project.proj, lxd_network.salt_lan, lxd_network.team_lan]
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


  depends_on = [ lxd_network.salt_lan, lxd_instance.project_fw ]
}



########################################################################
# Instances
########################################################################
resource "lxd_instance" "team_fw" {
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-team-fw"
  description = "project-openwrt"
  image  = "openwrt-team-new"
  type  = "container"
  profiles = ["pfsense"]
  
  dynamic "device" {
    for_each = {
      for i, net in tolist([
        lxd_network.team_wan.name,
        lxd_network.team_lan.name
      ]) : i => net
    }
    content {
      name = "eth-${device.key}"
      type = "nic"
      properties = {
        network = device.value
      }
    }
  }

  target = "@Cluster-C"

  timeouts = {
    create = "2m"
    delete = "2m"
  }

  depends_on = [ data.lxd_project.proj, lxd_network.team_wan, lxd_network.salt_lan ]
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
        lxd_network.team_lan.name
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
            - 10.0.${var.guac_subnet_octet}.${3 + count.index}/16
          routes:
            - to: default
              via: 10.0.0.1
          nameservers:
            addresses: [10.0.0.1]
        enp6s0:
          dhcp4: false
          addresses:
            - 192.168.1.${2+ count.index}/24
          routes:
            - to: 172.31.31.2
              via: 192.168.1.1
      EOF

    "cloud-init.user-data" = <<-EOF
      #cloud-config
      runcmd:
      - echo "${var.project_name}-team${count.index + 1}-${var.guac_name} localhost" | tee -a /etc/hosts
      - apt install nasm -y
      - apt install gdb -y
      - apt install ddd -y
      - apt install vim -y
      - apt install gcc -y
      - apt install gcc-multilib -y
    EOF
  }


  target = "@Cluster-C"

  depends_on = [ lxd_instance.project_guac_salt, lxd_network.team_lan, lxd_instance.project_fw ]

}
