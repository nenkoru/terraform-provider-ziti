data "ziti_service_edge_router_policy" "test_reference_ziti_service_edge_router_policy" {
  most_recent = true
  filter      = "name contains \"test\""
}
