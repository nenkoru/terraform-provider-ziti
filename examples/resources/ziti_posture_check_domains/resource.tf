resource "ziti_posture_check_domains" "test_ziti_posture_domains" {
  name            = "test_posture_domains"
  role_attributes = ["test_operating_system"]
  tags = {
    test_tag = "test"
    tttt     = "ttt"
  }
  domains = ["contoso.net"]
}
