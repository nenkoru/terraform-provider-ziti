data "ziti_host_config_v1" "example_reference_by_filter" {
    most_recent = true
    filter = "name contains \"v1\""
}

data "ziti_host_config_v1" "example_reference_by_name" {
    name = "simple_host.host.v1"
}

data "ziti_host_config_v1" "example_reference_by_name" {
    id = "4k50QRBgdJqtNE3YhyuteV"
}
