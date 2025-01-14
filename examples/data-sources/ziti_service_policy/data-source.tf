data "ziti_service_policy" "test_reference_ziti_service_policy" {
  most_recent = true
  filter      = "name contains \"test\""
}
