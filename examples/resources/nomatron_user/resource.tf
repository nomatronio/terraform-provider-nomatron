resource "nomatron_user" "example" {
  username = "tf-test-user"
  name     = "Terraform Test User"
  password = "ChangeMe123!"

  metadata = {
    team  = "platform"
    owner = "terraform"
  }
}
