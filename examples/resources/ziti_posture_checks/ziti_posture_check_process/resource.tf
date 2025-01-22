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
