package main

import "time"

type Appointment struct {
	ID         string    `json:"id"        dynamodbav:"id"`
	UserID     string    `json:"userId"    dynamodbav:"userId"`
	UserEmail  string    `json:"userEmail" dynamodbav:"userEmail"`
	UserName   string    `json:"userName"  dynamodbav:"userName"`
	Date       string    `json:"date"      dynamodbav:"date"`     // YYYY-MM-DD
	TimeSlot   string    `json:"timeSlot"  dynamodbav:"timeSlot"` // HH:MM
	Service    string    `json:"service"   dynamodbav:"service"`
	Status     string    `json:"status"    dynamodbav:"status"` // booked | cancelled
	BarberID   string    `json:"barberId"  dynamodbav:"barberId"`
	BarberName string    `json:"barberName" dynamodbav:"barberName"`
	Notes      string    `json:"notes"     dynamodbav:"notes"`
	CreatedAt  time.Time `json:"createdAt" dynamodbav:"createdAt"`
}

type SlotResponse struct {
	Date      string `json:"date"`
	TimeSlot  string `json:"timeSlot"`
	Available bool   `json:"available"`
}

type Service struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Duration int    `json:"duration"` // minutes
	Price    string `json:"price"`
}

var services = map[string]Service{
	"haircut":       {ID: "haircut", Name: "Haircut", Duration: 30, Price: "$25"},
	"beard":         {ID: "beard", Name: "Beard Trim", Duration: 20, Price: "$15"},
	"haircut_beard": {ID: "haircut_beard", Name: "Haircut + Beard", Duration: 45, Price: "$35"},
}
