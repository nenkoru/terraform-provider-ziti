data "ziti_posture_check_mfa" "test_ziti_posture_mfa" {
  filter      = "name contains \"test\""
  most_recent = true
}
