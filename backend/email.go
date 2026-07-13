package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// sendBookingConfirmation emails the customer a confirmation of their appointment.
// Best-effort: failures are logged but never block the booking itself.
func sendBookingConfirmation(ctx context.Context, appt Appointment) {
	if fromEmail == "" || appt.UserEmail == "" {
		return
	}

	subject := fmt.Sprintf("Appointment confirmed — %s", formatApptDate(appt.Date))
	body := fmt.Sprintf(
		"Hi %s,\n\n"+
			"Your appointment at %s is confirmed!\n\n"+
			"  Date:    %s\n"+
			"  Time:    %s\n"+
			"  Service: %s\n"+
			"  Barber:  %s\n\n"+
			"%s"+
			"See you soon!\n%s",
		firstNonEmpty(appt.UserName, "there"),
		siteName,
		formatApptDate(appt.Date),
		appt.TimeSlot,
		appt.Service,
		firstNonEmpty(appt.BarberName, "your barber"),
		noteLine(appt.Notes),
		siteName,
	)

	_, err := sesClient.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(fromEmail),
		Destination: &types.Destination{
			ToAddresses: []string{appt.UserEmail},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body)},
				},
			},
		},
	})
	if err != nil {
		log.Printf("failed to send booking confirmation email to %s: %v", appt.UserEmail, err)
	}
}

// sendBarberBookingNotice emails the barber that a new appointment was booked.
// Best-effort: failures are logged but never block the booking itself.
func sendBarberBookingNotice(ctx context.Context, appt Appointment) {
	if fromEmail == "" || appt.BarberID == "" {
		return
	}

	user, err := cognitoClient.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: &userPoolID,
		Username:   &appt.BarberID,
	})
	if err != nil {
		log.Printf("failed to look up barber %s for booking notice: %v", appt.BarberID, err)
		return
	}
	var barberEmail string
	for _, attr := range user.UserAttributes {
		if aws.ToString(attr.Name) == "email" {
			barberEmail = aws.ToString(attr.Value)
			break
		}
	}
	if barberEmail == "" {
		return
	}

	subject := fmt.Sprintf("New booking — %s at %s", formatApptDate(appt.Date), appt.TimeSlot)
	body := fmt.Sprintf(
		"Hi %s,\n\n"+
			"You have a new appointment on %s.\n\n"+
			"  Date:     %s\n"+
			"  Time:     %s\n"+
			"  Service:  %s\n"+
			"  Customer: %s\n\n"+
			"%s"+
			"— %s",
		firstNonEmpty(appt.BarberName, "there"),
		siteName,
		formatApptDate(appt.Date),
		appt.TimeSlot,
		appt.Service,
		firstNonEmpty(appt.UserName, appt.UserEmail),
		noteLine(appt.Notes),
		siteName,
	)

	_, err = sesClient.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(fromEmail),
		Destination: &types.Destination{
			ToAddresses: []string{barberEmail},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body)},
				},
			},
		},
	})
	if err != nil {
		log.Printf("failed to send booking notice email to barber %s: %v", barberEmail, err)
	}
}

func noteLine(notes string) string {
	if notes == "" {
		return ""
	}
	return fmt.Sprintf("  Notes:   %s\n\n", notes)
}

func firstNonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func formatApptDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("Monday, January 2, 2006")
}
