package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type ProjectRequest struct {
	ClientName  string
	Email       string
	Phone       string
	Requirement string
	SubmittedAt time.Time
}

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[0-9\s\-()]{7,20}$`)
)

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

// Helper function to determine if request expects JSON response
func isJSONRequest(request events.APIGatewayProxyRequest) bool {
	for k, v := range request.Headers {
		if strings.EqualFold(k, "accept") && strings.Contains(strings.ToLower(v), "application/json") {
			return true
		}
		if strings.EqualFold(k, "x-requested-with") && strings.EqualFold(v, "xmlhttprequest") {
			return true
		}
	}
	return false
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod != "POST" {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusMethodNotAllowed, Body: "Method Not Allowed"}, nil
	}

	values, err := url.ParseQuery(request.Body)
	if err != nil {
		if isJSONRequest(request) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"success":false,"error":"Failed to parse form data"}`,
			}, nil
		}
		return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest, Body: "Failed to parse form data"}, nil
	}

	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if smtpUser == "" || smtpPass == "" {
		log.Println("Error: Missing email environment variables")
		if isJSONRequest(request) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"success":false,"error":"Server configuration error"}`,
			}, nil
		}
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Server configuration error"}, nil
	}

	clientName := strings.TrimSpace(values.Get("name"))
	email := strings.TrimSpace(values.Get("email"))
	phone := strings.TrimSpace(values.Get("phone"))
	requirements := strings.TrimSpace(values.Get("requirements"))

	// Server-side validation
	var validationErrors []string
	if len(clientName) < 2 {
		validationErrors = append(validationErrors, "Client/Business Name must be at least 2 characters long.")
	}
	if !emailRegex.MatchString(email) {
		validationErrors = append(validationErrors, "A valid contact email address is required.")
	}
	
	// Clean phone number for validation checks
	cleanedPhone := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(phone)
	indianPhoneRegex := regexp.MustCompile(`^(?:\+91|91|0)?[6-9]\d{9}$`)
	genericPhoneRegex := regexp.MustCompile(`^\+?\d{7,15}$`)

	if phone == "" {
		validationErrors = append(validationErrors, "Phone number is required.")
	} else if !indianPhoneRegex.MatchString(cleanedPhone) && !genericPhoneRegex.MatchString(cleanedPhone) {
		validationErrors = append(validationErrors, "A valid phone number is required (e.g. +91 98765 43210).")
	}
	
	if len(requirements) < 10 {
		validationErrors = append(validationErrors, "System requirements must be at least 10 characters long.")
	}

	if len(validationErrors) > 0 {
		errMsg := strings.Join(validationErrors, " ")
		if isJSONRequest(request) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       fmt.Sprintf(`{"success":false,"error":%q}`, errMsg),
			}, nil
		}

		errorHTML := fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 40px auto; text-align: center; color: #ffffff;">
				<h2 style="color: #e74c3c;">Validation Failed</h2>
				<p>%s</p>
				<a href="javascript:history.back()" style="color: #58a6ff; text-decoration: none; border: 1px solid #58a6ff; padding: 10px 20px; border-radius: 5px; display: inline-block; margin-top: 20px;">&larr; Return & Correct</a>
			</div>
			<style>body { background-color: #0d1117; }</style>
		`, errMsg)

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    map[string]string{"Content-Type": "text/html"},
			Body:       errorHTML,
		}, nil
	}

	req := ProjectRequest{
		ClientName:  clientName,
		Email:       email,
		Phone:       phone,
		Requirement: requirements,
		SubmittedAt: time.Now(),
	}

	tableHTML := fmt.Sprintf(`
		<table style="width: 100%%; max-width: 600px; border-collapse: collapse; margin-top: 20px; font-family: sans-serif; color: #333333;">
			<tr style="background-color: #f8f9fa;">
				<th style="padding: 12px; border: 1px solid #dee2e6; text-align: left; width: 30%%;">Field</th>
				<th style="padding: 12px; border: 1px solid #dee2e6; text-align: left;">Details</th>
			</tr>
			<tr>
				<td style="padding: 12px; border: 1px solid #dee2e6; font-weight: bold;">Name</td>
				<td style="padding: 12px; border: 1px solid #dee2e6;">%s</td>
			</tr>
			<tr>
				<td style="padding: 12px; border: 1px solid #dee2e6; font-weight: bold;">Email</td>
				<td style="padding: 12px; border: 1px solid #dee2e6;"><a href="mailto:%s" style="color: #0969da; text-decoration: none;">%s</a></td>
			</tr>
			<tr>
				<td style="padding: 12px; border: 1px solid #dee2e6; font-weight: bold;">Phone</td>
				<td style="padding: 12px; border: 1px solid #dee2e6;"><a href="tel:%s" style="color: #0969da; text-decoration: none;">%s</a></td>
			</tr>
			<tr>
				<td style="padding: 12px; border: 1px solid #dee2e6; font-weight: bold;">Requirements</td>
				<td style="padding: 12px; border: 1px solid #dee2e6; white-space: pre-wrap;">%s</td>
			</tr>
		</table>
	`, req.ClientName, req.Email, req.Email, req.Phone, req.Phone, req.Requirement)

	// 1. Send Email to Freelancer (Synchronous)
	freelancerBody := `<h3>New Project Request</h3><p>You have received a new lead from your portfolio:</p>` + tableHTML
	err = sendEmail(smtpUser, "New Freelance Lead: "+req.ClientName, freelancerBody, smtpUser, smtpPass)
	if err != nil {
		log.Printf("Failed to send email to freelancer: %v\n", err)
		if isJSONRequest(request) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"success":false,"error":"Failed to route internal mail"}`,
			}, nil
		}
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Body: "Failed to route internal mail"}, nil
	}

	// 2. Send Email to Client (Synchronous)
	clientBody := fmt.Sprintf(`
		<div style="font-family: sans-serif; color: #333333; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #dee2e6; border-radius: 8px;">
			<h2 style="color: #0969da; margin-top: 0;">Thank you for reaching out, %s!</h2>
			<p><em>"Good design is good business." — Thomas Watson Jr.</em></p>
			<p>I genuinely appreciate you taking the time to share your requirements. I am reviewing your project details right now and will contact you very soon to discuss the next steps.</p>
			<p style="font-weight: bold; margin-top: 24px; border-bottom: 2px solid #0969da; padding-bottom: 8px;">Submission Summary</p>
			%s
			<br>
			<p>Best regards,</p>
			<p><strong>Satyaveer Singh</strong><br>Freelance Web Developer</p>
		</div>
	`, req.ClientName, tableHTML)
	
	err = sendEmail(req.Email, "Received: Your Web Development Request", clientBody, smtpUser, smtpPass)
	if err != nil {
		log.Printf("Failed to send email to client: %v\n", err)
		// We don't necessarily fail the submission response if only client auto-responder fails
	}

	// 3. Success response
	if isJSONRequest(request) {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       fmt.Sprintf(`{"success":true,"message":"Transmission Successful","clientName":%q}`, req.ClientName),
		}, nil
	}

	successHTML := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 40px auto; text-align: center; color: #ffffff;">
			<h2 style="color: #2ecc71;">Transmission Successful</h2>
			<p>Thanks for reaching out, %s. I have emailed you a confirmation and will be in touch soon.</p>
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