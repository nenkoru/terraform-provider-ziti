resource "ziti_posture_check_mfa" "test_ziti_posture_mfa" {
  name            = "test_posture_mfa"
  role_attributes = ["test_operating_system"]
  tags = {
    test_tag = "test"
    tttt     = "ttt"
  }
  ignore_legacy_endpoints = false
  prompt_on_unlock        = true
  prompt_on_wake          = true
  timeout_seconds         = -1
}
