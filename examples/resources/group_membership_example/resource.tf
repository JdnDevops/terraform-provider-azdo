data "azdo_group_membership" "example" {
  group      = "[Templates]\\Contributors"
  members    = ["AzdNetman Build Service (DefaultCollection)", "AzdKubernetes Build Service (DefaultCollection)", "Kubernetes Build Service (DefaultCollection)"]
  project_id = "unique-id"
}
