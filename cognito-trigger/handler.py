import boto3
from botocore.exceptions import ClientError

cognito = boto3.client("cognito-idp")
ses = boto3.client("sesv2")

def handler(event, context):
    """
    Cognito Post-Confirmation trigger.
    - Adds every newly confirmed user to the 'customers' group.
      The shop owner must manually add themselves to 'admins' via AWS Console or CLI after signup.
    - Registers the user's just-verified email as an SES identity, so booking
      confirmation emails can be sent to it. While the SES account is in
      sandbox mode, AWS still requires the recipient to click a one-time link
      in a separate "Amazon SES Address Verification Request" email before
      SES will deliver to that address — this just triggers that step
      automatically instead of the shop owner doing it by hand per customer.
      Once SES production access is granted, this step becomes a no-op.
    """
    cognito.admin_add_user_to_group(
        UserPoolId=event["userPoolId"],
        Username=event["userName"],
        GroupName="customers",
    )

    email = event.get("request", {}).get("userAttributes", {}).get("email")
    if email:
        try:
            ses.create_email_identity(EmailIdentity=email)
        except ClientError as e:
            # Already registered, pending, or SES is in production mode — safe to ignore.
            print(f"ses create_email_identity for {email}: {e}")

    return event
