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
        "3kJOVK9NNIq0lfQvJJrsi4" = 10
    }
}
