terraform {
  required_version = "~> 0.12"
}

provider "local" {

}

provider "google" {

}

variable "drone_gke_test_key_path" {
  type = string
}

locals {
  invalid_permissions = ["resourcemanager.projects.list"]
}

data "google_iam_role" "gke_developer" {
  name = "roles/container.developer"
}

resource "google_project_iam_custom_role" "drone_gke_tester" {
  role_id     = "drone_gke_tester"
  title       = "drone-gke Tester"
  description = "Custom role used for testing drone-gke (clone of roles/container.developer)"
  permissions = [for permission in data.google_iam_role.gke_developer.included_permissions : permission if !contains(local.invalid_permissions, permission)]
}

resource "google_service_account" "drone_gke_test" {
  account_id   = "drone-gke-test"
  display_name = "drone-gke Test"
}

resource "google_service_account_key" "drone_gke_test_key" {
  service_account_id = "${google_service_account.drone_gke_test.name}"
}

resource "google_project_iam_binding" "drone_gke_test_binding" {
  role    = "projects/${google_project_iam_custom_role.drone_gke_tester.project}/roles/${google_project_iam_custom_role.drone_gke_tester.role_id}"

  members = [
    "serviceAccount:${google_service_account.drone_gke_test.email}",
  ]
}

resource "local_file" "drone_gke_test_credentials" {
    sensitive_content = "${base64decode(google_service_account_key.drone_gke_test_key.private_key)}"
    filename          = "${var.drone_gke_test_key_path}"
}
