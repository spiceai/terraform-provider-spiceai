# Get all members in the organization
data "spiceai_members" "all" {}

# Output the list of usernames
output "member_usernames" {
  description = "Usernames of all organization members"
  value       = [for member in data.spiceai_members.all.members : member.username]
}

# Find the organization owner
output "organization_owner" {
  description = "The organization owner"
  value       = [for member in data.spiceai_members.all.members : member.username if member.is_owner][0]
}

# List all admin members
output "admin_members" {
  description = "Members with admin role"
  value = [for member in data.spiceai_members.all.members : member.username
  if contains(member.roles, "admin")]
}

# Get detailed member information
output "members_info" {
  description = "Detailed information about all members"
  value = [for member in data.spiceai_members.all.members : {
    user_id    = member.user_id
    username   = member.username
    roles      = member.roles
    is_owner   = member.is_owner
    created_at = member.created_at
  }]
}

# Count of members by role
output "member_count" {
  description = "Total number of organization members"
  value       = length(data.spiceai_members.all.members)
}