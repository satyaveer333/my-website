package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProjectRequest struct {
	ClientName  string    `bson:"client_name"`
	Email       string    `bson:"email"`
	Requirement string    `bson:"requirement"`
	SubmittedAt time.Time `bson:"submitted_at"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod != "POST" {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusMethodNotAllowed, Body: "Method Not Allowed"}, nil
	}

	values, err := url.ParseQuery(request.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest, Body: "Failed to parse form data"}, nil
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Println("Error: MONGODB_URI environment variable is not set")
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Database configuration error"}, nil
	}

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Database connection failed"}, nil
	}
	defer client.Disconnect(ctx)

	collection := client.Database("freelanceDB").Collection("client_requests")

	req := ProjectRequest{
		ClientName:  values.Get("name"),
		Email:       values.Get("email"),
		Requirement: values.Get("requirements"),
		SubmittedAt: time.Now(),
	}

	_, err = collection.InsertOne(ctx, req)
	if err != nil {
		log.Printf("DB Insert Error: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Failed to save data"}, nil
	}

	successHTML := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 40px auto; text-align: center;">
			<h2 style="color: #2ecc71;">Success!</h2>
			<p>Thanks for reaching out, %s. I have received your requirements and will email you soon.</p>
			<a href="/" style="color: #3498db; text-decoration: none;">&larr; Back to Portfolio</a>
		</div>
	`, req.ClientName)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "text/html"},
		Body:       successHTML,
	}, nil
}

func main() {
	lambda.Start(handler)
}
