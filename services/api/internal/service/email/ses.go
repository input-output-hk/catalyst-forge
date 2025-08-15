package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	sesv2 "github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type SESService struct {
	client  *sesv2.Client
	sender  string
	baseURL string
}

type SESOptions struct {
	Region  string
	Sender  string
	BaseURL string
}

func NewSES(ctx context.Context, opts SESOptions, cfgLoaders ...func(*config.LoadOptions) error) (*SESService, error) {
	lo := []func(*config.LoadOptions) error{}
	if opts.Region != "" {
		lo = append(lo, config.WithRegion(opts.Region))
	}
	lo = append(lo, cfgLoaders...)
	awscfg, err := config.LoadDefaultConfig(ctx, lo...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &SESService{client: sesv2.NewFromConfig(awscfg), sender: opts.Sender, baseURL: opts.BaseURL}, nil
}

func (s *SESService) SendInvite(ctx context.Context, to string, inviteLink string) error {
	subject := "You're invited to Catalyst Foundry"
	html := fmt.Sprintf("<p>You have been invited. Click <a href=\"%s\">here</a> to verify your email and continue setup.</p>", inviteLink)
	text := fmt.Sprintf("You have been invited. Open this link to verify: %s", inviteLink)

	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.sender),
		Destination: &sestypes.Destination{
			ToAddresses: []string{to},
		},
		Content: &sestypes.EmailContent{
			Simple: &sestypes.Message{
				Subject: &sestypes.Content{Data: aws.String(subject)},
				Body: &sestypes.Body{
					Html: &sestypes.Content{Data: aws.String(html)},
					Text: &sestypes.Content{Data: aws.String(text)},
				},
			},
		},
	})
	return err
}
