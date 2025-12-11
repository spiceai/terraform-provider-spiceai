# Add a regular member to the organization
resource "spiceai_member" "developer" {
  username = "johndoe"
  roles    = ["member"]
}

# Add an admin member to the organization
resource "spiceai_member" "admin" {
  username = "janedoe"
  roles    = ["admin", "member"]
}

# Add multiple team members
resource "spiceai_member" "team" {
  for_each = toset(["alice", "bob", "charlie"])

  username = each.key
  roles    = ["member"]
}

# Add a member with admin privileges for specific tasks
resource "spiceai_member" "ops_admin" {
  username = "ops-user"
  roles    = ["admin"]
}