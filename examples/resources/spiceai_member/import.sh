# Import a member using their user ID
terraform import spiceai_member.developer 123

# Import an admin member
terraform import spiceai_member.admin 456

# Import a specific team member (when using for_each)
terraform import 'spiceai_member.team["alice"]' 789

# Note: Organization owners cannot be managed via Terraform.
# Attempting to modify or delete an owner will result in an error.