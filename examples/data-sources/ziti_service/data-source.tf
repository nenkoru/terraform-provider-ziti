data "ziti_service" "test_data_ziti_service" {
  most_recent = true
  filter      = "name = \"test_service\""
}
