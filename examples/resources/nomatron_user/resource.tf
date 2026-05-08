resource "nomatron_user" "example" {
  username = "tf-test-user"
  name     = "Terraform Test User"
  password_wo = "ChangeMe123!"

  metadata = {
    team  = "platform"
    owner = "terraform"
  }
}
