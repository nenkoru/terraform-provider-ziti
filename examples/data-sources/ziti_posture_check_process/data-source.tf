data "ziti_posture_check_process" "test_ziti_posture_process" {
  filter      = "name contains \"test\""
  most_recent = true
}
