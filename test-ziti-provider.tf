# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    ziti = {
      source = "nenkoru/ziti"
    }
  }
}

provider "ziti" {
  username            = "testadmin"
  password        = "testadmin"
  mgmt_endpoint            = "https://localhost:1280/edge/management/v1"
}

resource "ziti_host_config_v1" "simple_host" {
    name = "simple_host.host.v1"
    address = "localhost"
    port    = 5432
    protocol = "tcp"
}

resource "ziti_host_config_v1" "forward_protocol_host" {
    name = "forward_protocol.host.v1"
    address = "localhost"
    port    = 5432
    forward_protocol = true
    allowed_protocols = ["tcp", "udp"]
}

resource "ziti_host_config_v1" "forward_port_host" {
    name = "forward_port.host.v1"
    address = "localhost"
    protocol = "tcp"
    forward_port = true
    allowed_port_ranges = [
        {
            low = 80
            high = 443
        }
    ]
}

resource "ziti_host_config_v1" "forward_port_protocol_host" {
    name = "forward_port_protocol.host.v1"
    address = "localhost"
    forward_protocol = true
    allowed_protocols = ["tcp", "udp"]
    forward_port = true
    allowed_port_ranges = [
        {
            low = 80
            high = 443
        }
    ]
}

resource "ziti_host_config_v1" "forward_port_protocol_address_host" {
    name = "forward_port_protocol_address.host.v1"
    forward_protocol = true
    forward_address = true
    forward_port = true
    allowed_addresses = ["localhost"]
    allowed_protocols = ["tcp", "udp"]
    allowed_port_ranges = [
        {
            low = 80
            high = 443
        }
    ]
}

resource "ziti_host_config_v1" "forward_port_protocol_address_allowed_addresses_host" {
    name = "forward_port_protocol_address_allowed_addresses.host.v1"
    forward_protocol = true
    forward_address = true
    forward_port = true
    allowed_addresses = ["localhost"]
    allowed_source_addresses = ["192.168.0.1"]
    allowed_protocols = ["tcp", "udp"]
    allowed_port_ranges = [
        {
            low = 80
            high = 443
        }
    ]
}


resource "ziti_host_config_v1" "forward_port_protocol_address_allowed_addresses_listen_host" {
    name = "forward_port_protocol_address_allowed_addresses_listen.host.v1"
    forward_protocol = true
    forward_address = true
    forward_port = true
    allowed_addresses = ["localhost"]
    allowed_source_addresses = ["192.168.0.1"]
    allowed_protocols = ["tcp", "udp"]
    listen_options = {
        connect_timeout = "10s"
        precedence = "default"
    }
    allowed_port_ranges = [
        {
            low = 80
            high = 443
        }
    ]
}

resource "ziti_host_config_v1" "forward_port_protocol_address_allowed_addresses_listen_port_checks_host" {
    name = "forward_port_protocol_address_allowed_addresses_listen_port_checks.host.v1"
    forward_protocol = true
    forward_address = true
    forward_port = true
    allowed_addresses = ["localhost"]
    allowed_source_addresses = ["192.168.0.1"]
    allowed_protocols = ["tcp", "udp"]
    port_checks = [
        {
            address = "localhost"
            interval = "5s"
            timeout = "10s"
            actions = [
                {
                    trigger = "fail"
                    duration = "10s"
                    action = "mark unhealthy"
                },
{
                    trigger = "fail"
                    duration = "10s"
                    action = "mark unhealthy"
                }
            ]

        }
    ]
    listen_options = {
        connect_timeout = "10s"
        precedence = "default"
    }
    allowed_port_ranges = [
        {
            low = 80
            high = 443
        }
    ]
}

resource "ziti_host_config_v1" "forward_port_protocol_address_allowed_addresses_listen_http_checks_host" {
    name = "forward_port_protocol_address_allowed_addresses_listen_http_checks.host.v1"
    forward_protocol = true
    forward_address = true
    forward_port = true
    allowed_addresses = ["localhost"]
    allowed_source_addresses = ["192.168.0.1"]
    allowed_protocols = ["tcp", "udp"]
    http_checks = [
        {
            url = "https://localhost/health"
            method = "GET"
            expect_status = 200
            expect_in_body = "healthy"
            interval = "5s"
            timeout = "10s"
            actions = [
                {
                    trigger = "fail"
                    duration = "10s"
                    action = "mark unhealthy"
                }
            ]

        }
    ]
    port_checks = [
        {
            address = "localhost"
            interval = "5s"
            timeout = "10s"
            actions = [
                {
                    trigger = "fail"
                    duration = "10s"
                    action = "mark unhealthy"
                }
            ]

        }
    ]
    listen_options = {
        connect_timeout = "10s"
        precedence = "default"
    }
    allowed_port_ranges = [
        {
            low = 80
            high = 443
        }
    ]
}

data "ziti_host_config_v1" "test_reference_configs" {
    most_recent = true
    filter = "name contains \"v1\""

}

data "ziti_host_config_v1" "test_reference_by_name" {
    name = ziti_host_config_v1.simple_host.name

}

data "ziti_host_config_v1_ids" "test_config_ids" {
    filter = "name contains \"v1\""
}

resource "ziti_service" "test_service" {
    name = "test_service"
    configs = [ziti_host_config_v1.forward_protocol_host.id]
}

resource "ziti_intercept_config_v1" "simple_intercept" {
    name = "simple_intercept.intercept.v1"

    addresses = ["localhost"]
    protocols = ["tcp", "udp"]
    port_ranges = [
        {
            low = 80
            high = 443
        }
    ]
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
    name = "test_ziti_service_policy"
    semantic = "AnyOf"
    type = "Dial"
    tags = {
        test_value = "test"
    }
    identity_roles = ["@${ziti_identity.test_ziti_identity.id}"]
    service_roles = ["@${ziti_service.test_service.id}"]
    posture_check_roles = ["#default"]
}

resource "ziti_service_edge_router_policy" "test_ziti_service_edge_router_policy" {
    name = "test_ziti_service_edge_router_policy"
    semantic = "AllOf"
    tags = {
        test_value = "test"
    }
    edge_router_roles = ["#all"]
    service_roles = ["@${ziti_service.test_service.id}"]
}

resource "ziti_edge_router_policy" "test_ziti_service_edge_router_policy" {
    name = "test_ziti_service_edge_router_policy"
    semantic = "AllOf"
    tags = {
        test_value = "test"
    }
    edge_router_roles = ["#all"]
    identity_roles = ["#all"]
}


resource "ziti_posture_check_multi_process" "test_ziti_posture_multi_process" {
    name = "test_posture_multi_process"
    role_attributes = ["test_multi_process"]
    tags = {
        test_tag = "test"
        tttt = "ttt"
    }
    processes = [
        {
            path = "/usr/bin"
            os_type = "Linux"
        }
    ]
    semantic = "AnyOf"
}

data "ziti_posture_check_multi_process" "test_ziti_posture_multi_process" {
    filter = "name contains \"test\""
    most_recent = true
}

data "ziti_posture_check_multi_process" "test_ziti_posture_multi_process_ids" {
    filter = "name contains \"test\""
}

resource "ziti_posture_check_process" "test_ziti_posture_process" {
    name = "test_posture_process"
    role_attributes = ["test_process"]
    tags = {
        test_tag = "test"
        tttt = "ttt"
    }
    process = {
        path = "/usr/bin"
        os_type = "Linux"
    }
}

data "ziti_posture_check_process" "test_ziti_posture_process" {
    filter = "name contains \"test\""
    most_recent = true
}

data "ziti_posture_check_process_ids" "test_ziti_posture_process_ids" {
    filter = "name contains \"test\""
}
