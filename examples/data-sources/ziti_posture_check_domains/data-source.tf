data "ziti_posture_check_domains" "test_ziti_posture_domains" {
  filter      = "name contains \"test\""
  most_recent = true
}
