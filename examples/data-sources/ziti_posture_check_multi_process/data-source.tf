data "ziti_posture_check_multi_process" "test_ziti_posture_multi_process" {
  filter      = "name contains \"test\""
  most_recent = true
}
