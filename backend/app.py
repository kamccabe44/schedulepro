import json
import os
import uuid
from datetime import datetime, date

import boto3
from boto3.dynamodb.conditions import Key

dynamodb = boto3.resource("dynamodb")
table = dynamodb.Table(os.environ["TABLE_NAME"])


def handler(event, context):
    method = event.get("requestContext", {}).get("http", {}).get("method", "")
    path = event.get("rawPath", "")
    path_params = event.get("pathParameters") or {}
    query_params = event.get("queryStringParameters") or {}
    body = {}
    if event.get("body"):
        body = json.loads(event["body"])

    try:
        if path == "/events" and method == "GET":
            return list_events(query_params)
        elif path == "/events" and method == "POST":
            return create_event(body)
        elif path.startswith("/events/") and method == "GET":
            return get_event(path_params["id"])
        elif path.startswith("/events/") and method == "PUT":
            return update_event(path_params["id"], body)
        elif path.startswith("/events/") and method == "DELETE":
            return delete_event(path_params["id"])
        else:
            return respond(404, {"error": "Not found"})
    except Exception as e:
        print(f"Error: {e}")
        return respond(500, {"error": str(e)})


def list_events(params):
    start = params.get("start")
    end = params.get("end")

    if start:
        # Query by date range using the GSI
        start_date = start[:10]  # YYYY-MM-DD
        end_date = (end or start)[:10]

        # Scan with filter (for simplicity; replace with GSI range query if needed)
        result = table.scan(
            FilterExpression="startDate BETWEEN :s AND :e",
            ExpressionAttributeValues={":s": start_date, ":e": end_date},
        )
    else:
        result = table.scan()

    items = sorted(result.get("Items", []), key=lambda x: x.get("startTime", ""))
    return respond(200, items)


def create_event(body):
    validate_event(body)
    item = {
        "id": str(uuid.uuid4()),
        "title": body["title"],
        "startTime": body["startTime"],
        "endTime": body["endTime"],
        "startDate": body["startTime"][:10],  # YYYY-MM-DD for GSI
        "description": body.get("description", ""),
        "location": body.get("location", ""),
        "color": body.get("color", "#3b82f6"),
        "createdAt": datetime.utcnow().isoformat(),
    }
    table.put_item(Item=item)
    return respond(201, item)


def get_event(event_id):
    result = table.get_item(Key={"id": event_id})
    item = result.get("Item")
    if not item:
        return respond(404, {"error": "Event not found"})
    return respond(200, item)


def update_event(event_id, body):
    existing = table.get_item(Key={"id": event_id}).get("Item")
    if not existing:
        return respond(404, {"error": "Event not found"})

    allowed = ["title", "startTime", "endTime", "description", "location", "color"]
    updates = {k: v for k, v in body.items() if k in allowed}
    if "startTime" in updates:
        updates["startDate"] = updates["startTime"][:10]
    updates["updatedAt"] = datetime.utcnow().isoformat()

    expr = "SET " + ", ".join(f"#{k} = :{k}" for k in updates)
    names = {f"#{k}": k for k in updates}
    values = {f":{k}": v for k, v in updates.items()}

    result = table.update_item(
        Key={"id": event_id},
        UpdateExpression=expr,
        ExpressionAttributeNames=names,
        ExpressionAttributeValues=values,
        ReturnValues="ALL_NEW",
    )
    return respond(200, result["Attributes"])


def delete_event(event_id):
    table.delete_item(Key={"id": event_id})
    return respond(204, None)


def validate_event(body):
    required = ["title", "startTime", "endTime"]
    missing = [f for f in required if not body.get(f)]
    if missing:
        raise ValueError(f"Missing required fields: {', '.join(missing)}")


def respond(status, body):
    return {
        "statusCode": status,
        "headers": {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*",
        },
        "body": json.dumps(body, default=str) if body is not None else "",
    }
