provider "vmware" {
  vcenter_server = "vcenter.vsphere55.test"
  user = "root"
  password = "jetbrains"
  insecure_connection = true
}
resource "vmware_virtual_machine" "vm" {
  name =  "vm-1"
  image = "basic"
  linked_clone = true
  domain = "vsphere55.test"
  ip_address = "192.168.1.1"
  subnet_mask = "255.255.255.0"
  gateway = "192.168.1.10"
}
