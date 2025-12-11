# Import a secret using the format: app_id/secret_name
terraform import spiceai_secret.database_password 123/DATABASE_PASSWORD

# Import multiple secrets for the same app
terraform import spiceai_secret.api_token 123/EXTERNAL_API_TOKEN
terraform import spiceai_secret.aws_access_key 123/AWS_ACCESS_KEY_ID

# Note: After import, you must set the 'value' attribute in your configuration
# since secret values are not returned by the API (they are masked).