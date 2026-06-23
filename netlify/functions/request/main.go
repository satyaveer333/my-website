package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type ProjectRequest struct {
	ClientName  string
	Email       string
	Requirement string
	SubmittedAt time.Time
}

// Helper function to send HTML emails
func sendEmail(toEmail, subject, htmlBody, smtpUser, smtpPass string) error {
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	headers := "From: Satyaveer Singh <" + smtpUser + ">\r\n" +
		"To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n"

	message := []byte(headers + htmlBody)

	return smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpUser, []string{toEmail}, message)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod != "POST" {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusMethodNotAllowed, Body: "Method Not Allowed"}, nil
	}

	values, err := url.ParseQuery(request.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest, Body: "Failed to parse form data"}, nil
	}

	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if smtpUser == "" || smtpPass == "" {
		log.Println("Error: Missing email environment variables")
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Server configuration error"}, nil
	}

	req := ProjectRequest{
		ClientName:  values.Get("name"),
		Email:       values.Get("email"),
		Requirement: values.Get("requirements"),
		SubmittedAt: time.Now(),
	}

	tableHTML := fmt.Sprintf(`
		<table style="width: 100%%; max-width: 600px; border-collapse: collapse; margin-top: 20px; font-family: sans-serif;">
			<tr style="background-color: #f2f2f2;">
				<th style="padding: 12px; border: 1px solid #ddd; text-align: left;">Field</th>
				<th style="padding: 12px; border: 1px solid #ddd; text-align: left;">Details</th>
			</tr>
			<tr>
				<td style="padding: 12px; border: 1px solid #ddd; font-weight: bold;">Name</td>
				<td style="padding: 12px; border: 1px solid #ddd;">%s</td>
			</tr>
			<tr>
				<td style="padding: 12px; border: 1px solid #ddd; font-weight: bold;">Email</td>
				<td style="padding: 12px; border: 1px solid #ddd;">%s</td>
			</tr>
			<tr>
				<td style="padding: 12px; border: 1px solid #ddd; font-weight: bold;">Requirements</td>
				<td style="padding: 12px; border: 1px solid #ddd;">%s</td>
			</tr>
		</table>
	`, req.ClientName, req.Email, req.Requirement)

	// 1. Send Email to Freelancer (Synchronous - no "go" keyword)
	freelancerBody := `<h3>New Project Request</h3><p>You have received a new lead from your portfolio:</p>` + tableHTML
	err = sendEmail(smtpUser, "New Freelance Lead: "+req.ClientName, freelancerBody, smtpUser, smtpPass)
	if err != nil {
		log.Printf("Failed to send email to freelancer: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Failed to route internal mail"}, nil
	}

	// 2. Send Email to Client (Synchronous)
	clientBody := fmt.Sprintf(`
		<div style="font-family: sans-serif; color: #333;">
			<h2>Thank you for reaching out, %s!</h2>
			<p><em>"Good design is good business." — Thomas Watson Jr.</em></p>
			<p>I genuinely appreciate you taking the time to share your requirements. I am reviewing your project details right now and will contact you very soon to discuss the next steps.</p>
			<p>Here is a copy of what you submitted:</p>
			%s
			<br>
			<p>Best regards,</p>
			<p><strong>Satyaveer Singh</strong><br>Freelance Web Developer</p>
		</div>
	`, req.ClientName, tableHTML)
	
	err = sendEmail(req.Email, "Received: Your Web Development Request", clientBody, smtpUser, smtpPass)
	if err != nil {
		log.Printf("Failed to send email to client: %v\n", err)
		// We don't necessarily want to show an ugly error if the client email bounces, 
		// but we log it to be safe.
	}

	// 3. ONLY return success AFTER emails are safely sent
	successHTML := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 40px auto; text-align: center;">
			<h2 style="color: #2ecc71;">Transmission Successful</h2>
			<p style="color: #fff;">Thanks for reaching out, %s. I have emailed you a confirmation and will be in touch soon.</p>
			<a href="/" style="color: #58a6ff; text-decoration: none; border: 1px solid #58a6ff; padding: 10px 20px; border-radius: 5px; display: inline-block; margin-top: 20px;">&larr; Return to Terminal</a>
		</div>
		<style>body { background-color: #0d1117; }</style>
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