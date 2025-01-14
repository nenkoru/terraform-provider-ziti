resource "ziti_host_config_v1" "forward_protocol_host" {
  name              = "forward_protocol.host.v1"
  address           = "localhost"
  port              = 5432
  forward_protocol  = true
  allowed_protocols = ["tcp", "udp"]
}

resource "ziti_service" "test_service" {
  name    = "test_service"
  configs = [ziti_host_config_v1.forward_protocol_host.id]
}

resource "ziti_identity" "test_ziti_identity" {
  name = "test_identity"
  tags = {
    test_value = "test"
  }
  app_data = {
    test_app_data = "test_app_data"
  }
  role_attributes = ["test"]
  service_hosting_costs = {
    "${ziti_service.test_service.id}" = 10
  }
}

resource "ziti_service_policy" "test_ziti_service_policy" {
  name     = "test_ziti_service_policy"
  semantic = "AnyOf"
  type     = "Dial"
  tags = {
    test_value = "test"
  }
  identity_roles      = ["@${ziti_identity.test_ziti_identity.id}"]
  service_roles       = ["@${ziti_service.test_service.id}"]
  posture_check_roles = ["#default"]
}
