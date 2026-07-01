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

// DaySchedule holds a barber's hours for one weekday.
type DaySchedule struct {
	Open        bool `json:"open"        dynamodbav:"open"`
	OpenHour    int  `json:"openHour"    dynamodbav:"openHour"`
	OpenMinute  int  `json:"openMinute"  dynamodbav:"openMinute"`
	CloseHour   int  `json:"closeHour"   dynamodbav:"closeHour"`
	CloseMinute int  `json:"closeMinute" dynamodbav:"closeMinute"`
}

// BarberService is a service offering defined by a barber.
type BarberService struct {
	ID       string `json:"id"       dynamodbav:"id"`
	Name     string `json:"name"     dynamodbav:"name"`
	Duration int    `json:"duration" dynamodbav:"duration"` // minutes
	Price    string `json:"price"    dynamodbav:"price"`
}

// BarberSettings holds a barber's full schedule, service list, and payment handles.
type BarberSettings struct {
	BarberID      string                 `json:"barberId"      dynamodbav:"barberId"`
	Schedule      map[string]DaySchedule `json:"schedule"      dynamodbav:"schedule"`  // key = "Monday" etc.
	Services      []BarberService        `json:"services"      dynamodbav:"services"`
	VenmoHandle   string                 `json:"venmoHandle"   dynamodbav:"venmoHandle"`
	CashAppHandle string                 `json:"cashAppHandle" dynamodbav:"cashAppHandle"`
}

var services = map[string]Service{
	"haircut":       {ID: "haircut", Name: "Haircut", Duration: 30, Price: "$25"},
	"beard":         {ID: "beard", Name: "Beard Trim", Duration: 20, Price: "$15"},
	"haircut_beard": {ID: "haircut_beard", Name: "Haircut + Beard", Duration: 45, Price: "$35"},
}
