data "ziti_posture_check_operating_system" "test_ziti_posture_operating_system" {
  filter      = "name contains \"test\""
  most_recent = true
}
