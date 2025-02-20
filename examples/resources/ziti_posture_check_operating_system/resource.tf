resource "ziti_posture_check_operating_system" "test_ziti_posture_operating_system" {
  name            = "test_posture_operating_systems"
  role_attributes = ["test_operating_system"]
  tags = {
    test_tag = "test"
    tttt     = "ttt"
  }
  operating_systems = [
    {
      type     = "Windows"
      versions = ["10.0.0"]
    }
  ]
}
