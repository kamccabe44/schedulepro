import boto3

client = boto3.client("cognito-idp")

def handler(event, context):
    """
    Cognito Post-Confirmation trigger.
    Automatically adds every newly confirmed user to the 'customers' group.
    The shop owner must manually add themselves to 'admins' via AWS Console or CLI after signup.
    """
    client.admin_add_user_to_group(
        UserPoolId=event["userPoolId"],
        Username=event["userName"],
        GroupName="customers",
    )
    return event
