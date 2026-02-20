package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"splitwise-backend/config"
	"splitwise-backend/database"
	"splitwise-backend/models"

	"github.com/google/uuid"
)

type NotificationService struct{}

var notifService *NotificationService

func GetNotificationService() *NotificationService {
	if notifService == nil {
		notifService = &NotificationService{}
	}
	return notifService
}

// ============================================================
// PUSH NOTIFICATIONS via FCM HTTP v1 API
// ============================================================

type FCMMessage struct {
	To           string            `json:"to"`
	Notification FCMNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

type FCMNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Sound string `json:"sound"`
}

func (ns *NotificationService) sendPush(fcmToken string, title string, body string, data map[string]string) {
	if fcmToken == "" {
		return
	}

	msg := FCMMessage{
		To: fcmToken,
		Notification: FCMNotification{
			Title: title,
			Body:  body,
			Sound: "default",
		},
		Data: data,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("❌ FCM marshal error: %v", err)
		return
	}

	// Using FCM legacy HTTP API (simpler setup)
	// For production, use FCM HTTP v1 API with service account
	req, err := http.NewRequest("POST", "https://fcm.googleapis.com/fcm/send", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ FCM request error: %v", err)
		return
	}

	// Note: Replace with your FCM server key
	// For production, use Firebase Admin SDK
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key=YOUR_FCM_SERVER_KEY")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ FCM send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Printf("✅ Push notification sent to token: %s...", fcmToken[:20])
	} else {
		log.Printf("⚠️  FCM returned status: %d", resp.StatusCode)
	}
}

// ============================================================
// EMAIL NOTIFICATIONS via SendGrid
// ============================================================

type SendGridEmail struct {
	Personalizations []SGPersonalization `json:"personalizations"`
	From             SGEmail             `json:"from"`
	Subject          string              `json:"subject"`
	Content          []SGContent         `json:"content"`
}

type SGPersonalization struct {
	To []SGEmail `json:"to"`
}

type SGEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type SGContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (ns *NotificationService) sendEmail(toEmail string, toName string, subject string, htmlBody string) {
	if config.AppConfig.SendGridAPIKey == "" {
		log.Printf("⚠️  SendGrid API key not set, skipping email to %s", toEmail)
		return
	}

	email := SendGridEmail{
		Personalizations: []SGPersonalization{
			{
				To: []SGEmail{{Email: toEmail, Name: toName}},
			},
		},
		From:    SGEmail{Email: config.AppConfig.SendGridFrom, Name: config.AppConfig.AppName},
		Subject: subject,
		Content: []SGContent{
			{Type: "text/html", Value: htmlBody},
		},
	}

	jsonData, err := json.Marshal(email)
	if err != nil {
		log.Printf("❌ Email marshal error: %v", err)
		return
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Email request error: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.AppConfig.SendGridAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Email send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("✅ Email sent to %s", toEmail)
	} else {
		log.Printf("⚠️  SendGrid returned status: %d", resp.StatusCode)
	}
}

// ============================================================
// NOTIFICATION EVENTS
// ============================================================

// NotifyExpenseAdded sends push + email to all split participants
func (ns *NotificationService) NotifyExpenseAdded(expense models.Expense, splits []models.ExpenseSplit, payer models.User, group models.Group) {
	for _, split := range splits {
		if split.UserID == expense.PaidBy {
			continue // Don't notify the payer
		}

		var user models.User
		if err := database.DB.First(&user, split.UserID).Error; err != nil {
			continue
		}

		title := fmt.Sprintf("%s added an expense", payer.Name)
		body := fmt.Sprintf("You owe %s %.2f for \"%s\" in %s", expense.Currency, split.OwedAmount, expense.Description, group.Name)

		// Push notification
		ns.sendPush(user.FCMToken, title, body, map[string]string{
			"type":       "expense_added",
			"expense_id": expense.ID.String(),
			"group_id":   expense.GroupID.String(),
		})

		// Email notification
		htmlBody := buildExpenseEmailHTML(payer.Name, user.Name, expense.Description, expense.Amount, split.OwedAmount, expense.Currency, group.Name)
		ns.sendEmail(user.Email, user.Name, fmt.Sprintf("%s added \"%s\" in %s", payer.Name, expense.Description, group.Name), htmlBody)
	}
}

// NotifySettlement sends push + email to the payee
func (ns *NotificationService) NotifySettlement(settlement models.Settlement, payer models.User, payee models.User, group models.Group) {
	title := fmt.Sprintf("%s paid you", payer.Name)
	body := fmt.Sprintf("%s paid you INR %.2f in %s", payer.Name, settlement.Amount, group.Name)

	// Push
	ns.sendPush(payee.FCMToken, title, body, map[string]string{
		"type":     "settlement",
		"group_id": settlement.GroupID.String(),
	})

	// Email
	htmlBody := buildSettlementEmailHTML(payer.Name, payee.Name, settlement.Amount, group.Name)
	ns.sendEmail(payee.Email, payee.Name, fmt.Sprintf("%s settled up with you in %s", payer.Name, group.Name), htmlBody)
}

// NotifyMemberAdded sends push + email to the newly added member
func (ns *NotificationService) NotifyMemberAdded(group models.Group, adder models.User, newMember models.User) {
	title := fmt.Sprintf("You were added to \"%s\"", group.Name)
	body := fmt.Sprintf("%s added you to the group \"%s\"", adder.Name, group.Name)

	// Push
	ns.sendPush(newMember.FCMToken, title, body, map[string]string{
		"type":     "member_added",
		"group_id": group.ID.String(),
	})

	// Email
	htmlBody := buildMemberAddedEmailHTML(adder.Name, newMember.Name, group.Name)
	ns.sendEmail(newMember.Email, newMember.Name, title, htmlBody)
}

// NotifyInvitation sends email to non-registered users
func (ns *NotificationService) NotifyInvitation(email string, inviterName string, groupName string) {
	subject := fmt.Sprintf("%s invited you to join \"%s\" on %s", inviterName, groupName, config.AppConfig.AppName)
	htmlBody := buildInvitationEmailHTML(inviterName, groupName)
	ns.sendEmail(email, "", subject, htmlBody)
}

// ============================================================
// EMAIL TEMPLATES
// ============================================================

func buildExpenseEmailHTML(payerName, userName, description string, totalAmount, owedAmount float64, currency, groupName string) string {
	tmpl := `
<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
	<div style="background: white; border-radius: 12px; padding: 32px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
		<h2 style="color: #1DB954; margin-top: 0;">💰 New Expense Added</h2>
		<p>Hi <strong>{{.UserName}}</strong>,</p>
		<p><strong>{{.PayerName}}</strong> added a new expense in <strong>{{.GroupName}}</strong>:</p>
		<div style="background: #f8f9fa; border-radius: 8px; padding: 16px; margin: 16px 0;">
			<p style="margin: 4px 0; font-size: 18px;"><strong>{{.Description}}</strong></p>
			<p style="margin: 4px 0; color: #666;">Total: {{.Currency}} {{printf "%.2f" .TotalAmount}}</p>
			<p style="margin: 4px 0; color: #e53e3e; font-size: 18px;"><strong>Your share: {{.Currency}} {{printf "%.2f" .OwedAmount}}</strong></p>
		</div>
		<p style="color: #999; font-size: 12px; margin-top: 24px;">— SplitApp</p>
	</div>
</body>
</html>`

	t, _ := template.New("expense").Parse(tmpl)
	var buf bytes.Buffer
	t.Execute(&buf, map[string]interface{}{
		"PayerName":   payerName,
		"UserName":    userName,
		"Description": description,
		"TotalAmount": totalAmount,
		"OwedAmount":  owedAmount,
		"Currency":    currency,
		"GroupName":   groupName,
	})
	return buf.String()
}

func buildSettlementEmailHTML(payerName, payeeName string, amount float64, groupName string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
	<div style="background: white; border-radius: 12px; padding: 32px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
		<h2 style="color: #1DB954; margin-top: 0;">✅ Payment Recorded</h2>
		<p>Hi <strong>%s</strong>,</p>
		<p><strong>%s</strong> recorded a payment of <strong>INR %.2f</strong> to you in <strong>%s</strong>.</p>
		<p>Check the app to see your updated balances.</p>
		<p style="color: #999; font-size: 12px; margin-top: 24px;">— SplitApp</p>
	</div>
</body>
</html>`, payeeName, payerName, amount, groupName)
}

func buildMemberAddedEmailHTML(adderName, memberName, groupName string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
	<div style="background: white; border-radius: 12px; padding: 32px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
		<h2 style="color: #1DB954; margin-top: 0;">👋 You've been added to a group!</h2>
		<p>Hi <strong>%s</strong>,</p>
		<p><strong>%s</strong> added you to the group <strong>"%s"</strong>.</p>
		<p>Open the app to start splitting expenses with your group!</p>
		<p style="color: #999; font-size: 12px; margin-top: 24px;">— SplitApp</p>
	</div>
</body>
</html>`, memberName, adderName, groupName)
}

func buildInvitationEmailHTML(inviterName, groupName string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
	<div style="background: white; border-radius: 12px; padding: 32px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
		<h2 style="color: #1DB954; margin-top: 0;">🎉 You're invited!</h2>
		<p><strong>%s</strong> invited you to join <strong>"%s"</strong> on SplitApp.</p>
		<p>SplitApp makes it easy to split expenses with friends, roommates, and groups.</p>
		<div style="margin: 24px 0;">
			<a href="%s" style="background: #1DB954; color: white; padding: 12px 32px; border-radius: 8px; text-decoration: none; font-weight: bold;">Join Now</a>
		</div>
		<p style="color: #999; font-size: 12px; margin-top: 24px;">— SplitApp</p>
	</div>
</body>
</html>`, inviterName, groupName, config.AppConfig.AppURL)
}

// bytes import needed for template buffer
var _ = uuid.Nil // keep import
