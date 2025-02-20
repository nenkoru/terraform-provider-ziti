resource "ziti_posture_check_mac_addresses" "test_ziti_posture_mac_addresses" {
  name            = "test_posture_mac_addresses"
  role_attributes = ["test_operating_system"]
  tags = {
    test_tag = "test"
    tttt     = "ttt"
  }
  mac_addresses = ["00:1A:2B:3C:4D:5E"]
}
