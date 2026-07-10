package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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
