package main

import (
	"context"
	"sort"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// BarberMonthlyStat is the appointment count for one barber in one month.
type BarberMonthlyStat struct {
	BarberID   string `json:"barberId"`
	BarberName string `json:"barberName"`
	Booked     int    `json:"booked"`
	Cancelled  int    `json:"cancelled"`
	Total      int    `json:"total"`
}

// adminMonthlyStats returns appointment counts per barber for a given month.
// Accessible to admins only. Expects month=YYYY-MM.
func adminMonthlyStats(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if !isAdmin(req) {
		return respond(403, map[string]string{"error": "forbidden"})
	}

	month := req.QueryStringParameters["month"]
	if len(month) != 7 || month[4] != '-' {
		return respond(400, map[string]string{"error": "month query parameter required (YYYY-MM)"})
	}

	var appts []Appointment
	var lastKey map[string]types.AttributeValue
	for {
		out, err := db.Scan(ctx, &dynamodb.ScanInput{
			TableName:        &tableName,
			FilterExpression: aws.String("begins_with(#date, :month)"),
			ExpressionAttributeNames: map[string]string{
				"#date": "date",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":month": &types.AttributeValueMemberS{Value: month},
			},
			ExclusiveStartKey: lastKey,
		})
		if err != nil {
			return respond(500, map[string]string{"error": err.Error()})
		}
		var page []Appointment
		if err := attributevalue.UnmarshalListOfMaps(out.Items, &page); err != nil {
			return respond(500, map[string]string{"error": err.Error()})
		}
		appts = append(appts, page...)
		if out.LastEvaluatedKey == nil {
			break
		}
		lastKey = out.LastEvaluatedKey
	}

	stats := map[string]*BarberMonthlyStat{}
	for _, a := range appts {
		if a.BarberID == "" {
			continue
		}
		s, ok := stats[a.BarberID]
		if !ok {
			s = &BarberMonthlyStat{BarberID: a.BarberID, BarberName: a.BarberName}
			stats[a.BarberID] = s
		}
		if s.BarberName == "" {
			s.BarberName = a.BarberName
		}
		if a.Status == "cancelled" {
			s.Cancelled++
		} else {
			s.Booked++
		}
		s.Total++
	}

	result := make([]BarberMonthlyStat, 0, len(stats))
	for _, s := range stats {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].BarberName < result[j].BarberName })

	return respond(200, result)
}
