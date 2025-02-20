resource "ziti_posture_check_multi_process" "test_ziti_posture_multi_process" {
  name            = "test_posture_multi_process"
  role_attributes = ["test_multi_process"]
  tags = {
    test_tag = "test"
    tttt     = "ttt"
  }
  processes = [
    {
      path    = "/usr/bin"
      os_type = "Linux"
    }
  ]
  semantic = "AnyOf"
}

