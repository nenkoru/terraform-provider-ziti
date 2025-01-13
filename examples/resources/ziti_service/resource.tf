resource "ziti_host_config_v1" "forward_protocol_host" {
    name = "forward_protocol.host.v1"
    address = "localhost"
    port    = 5432
    forward_protocol = true
    allowed_protocols = ["tcp", "udp"]
}

resource "ziti_service" "test_service" {
    name = "test_service"
    configs = [ziti_host_config_v1.forward_protocol_host.id]
}
