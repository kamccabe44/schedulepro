package main

import "time"

type Event struct {
	ID          string    `json:"id"          dynamodbav:"id"`
	Title       string    `json:"title"       dynamodbav:"title"`
	StartTime   string    `json:"startTime"   dynamodbav:"startTime"`
	EndTime     string    `json:"endTime"     dynamodbav:"endTime"`
	StartDate   string    `json:"startDate"   dynamodbav:"startDate"`
	Description string    `json:"description" dynamodbav:"description"`
	Location    string    `json:"location"    dynamodbav:"location"`
	Color       string    `json:"color"       dynamodbav:"color"`
	CreatedAt   time.Time `json:"createdAt"   dynamodbav:"createdAt"`
}
