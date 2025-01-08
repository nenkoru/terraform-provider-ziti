data "ziti_intercept_config_v1" "example_reference_by_filter" {
    most_recent = true
    filter = "name contains \"v1\""
}

data "ziti_intercept_config_v1" "example_reference_by_name" {
    name = "simple_intercept.intercept.v1"
}

data "ziti_intercept_config_v1" "example_reference_by_id" {
    id = "4k50QRBgdJqtNE3YhyuteV"
}
