resource "ziti_host_config_v1" "simple_host" {
  name     = "simple_host.host.v1"
  address  = "localhost"
  port     = 5432
  protocol = "tcp"
}

resource "ziti_host_config_v1" "forward_protocol_host" {
  name              = "forward_protocol.host.v1"
  address           = "localhost"
  port              = 5432
  forward_protocol  = true
  allowed_protocols = ["tcp", "udp"]
}

resource "ziti_host_config_v1" "forward_port_host" {
  name         = "forward_port.host.v1"
  address      = "localhost"
  protocol     = "tcp"
  forward_port = true
  allowed_port_ranges = [
    {
      low  = 80
      high = 443
    }
  ]
}

resource "ziti_host_config_v1" "forward_port_protocol_host" {
  name              = "forward_port_protocol.host.v1"
  address           = "localhost"
  forward_protocol  = true
  allowed_protocols = ["tcp", "udp"]
  forward_port      = true
  allowed_port_ranges = [
    {
      low  = 80
      high = 443
    }
  ]
}

resource "ziti_host_config_v1" "forward_port_protocol_address_host" {
  name              = "forward_port_protocol_address.host.v1"
  forward_protocol  = true
  forward_address   = true
  forward_port      = true
  allowed_addresses = ["localhost"]
  allowed_protocols = ["tcp", "udp"]
  allowed_port_ranges = [
    {
      low  = 80
      high = 443
    }
  ]
}

resource "ziti_host_config_v1" "forward_port_protocol_address_allowed_addresses_host" {
  name                     = "forward_port_protocol_address_allowed_addresses.host.v1"
  forward_protocol         = true
  forward_address          = true
  forward_port             = true
  allowed_addresses        = ["localhost"]
  allowed_source_addresses = ["192.168.0.1"]
  allowed_protocols        = ["tcp", "udp"]
  allowed_port_ranges = [
    {
      low  = 80
      high = 443
    }
  ]
}


resource "ziti_host_config_v1" "forward_port_protocol_address_allowed_addresses_listen_host" {
  name                     = "forward_port_protocol_address_allowed_addresses_listen.host.v1"
  forward_protocol         = true
  forward_address          = true
  forward_port             = true
  allowed_addresses        = ["localhost"]
  allowed_source_addresses = ["192.168.0.1"]
  allowed_protocols        = ["tcp", "udp"]
  listen_options = {
    connect_timeout = "10s"
    precedence      = "default"
  }
  allowed_port_ranges = [
    {
      low  = 80
      high = 443
    }
  ]
}

resource "ziti_host_config_v1" "forward_port_protocol_address_allowed_addresses_listen_port_checks_host" {
  name                     = "forward_port_protocol_address_allowed_addresses_listen_port_checks.host.v1"
  forward_protocol         = true
  forward_address          = true
  forward_port             = true
  allowed_addresses        = ["localhost"]
  allowed_source_addresses = ["192.168.0.1"]
  allowed_protocols        = ["tcp", "udp"]
  port_checks = [
    {
      address  = "localhost"
      interval = "5s"
      timeout  = "10s"
      actions = [
        {
          trigger  = "fail"
          duration = "10s"
          action   = "mark unhealthy"
        },
        {
          trigger  = "fail"
          duration = "10s"
          action   = "mark unhealthy"
        }
      ]

    }
  ]
  listen_options = {
    connect_timeout = "10s"
    precedence      = "default"
  }
  allowed_port_ranges = [
    {
      low  = 80
      high = 443
    }
  ]
}

resource "ziti_host_config_v1" "forward_port_protocol_address_allowed_addresses_listen_http_checks_host" {
  name                     = "forward_port_protocol_address_allowed_addresses_listen_http_checks.host.v1"
  forward_protocol         = true
  forward_address          = true
  forward_port             = true
  allowed_addresses        = ["localhost"]
  allowed_source_addresses = ["192.168.0.1"]
  allowed_protocols        = ["tcp", "udp"]
  http_checks = [
    {
      url            = "https://localhost/health"
      method         = "GET"
      expect_status  = 200
      expect_in_body = "healthy"
      interval       = "5s"
      timeout        = "10s"
      actions = [
        {
          trigger  = "fail"
          duration = "10s"
          action   = "mark unhealthy"
        }
      ]

    }
  ]
  port_checks = [
    {
      address  = "localhost"
      interval = "5s"
      timeout  = "10s"
      actions = [
        {
          trigger  = "fail"
          duration = "10s"
          action   = "mark unhealthy"
        }
      ]

    }
  ]
  listen_options = {
    connect_timeout = "10s"
    precedence      = "default"
  }
  allowed_port_ranges = [
    {
      low  = 80
      high = 443
    }
  ]
}
