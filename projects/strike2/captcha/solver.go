package captcha

import (
	"fmt"
	"time"

	api2captcha "github.com/2captcha/2captcha-go"
)

// Solver wraps 2Captcha API for Amazon WAF captcha solving
type Solver struct {
	client *api2captcha.Client
}

// NewSolver creates a new 2Captcha solver
func NewSolver(apiKey string) *Solver {
	client := api2captcha.NewClient(apiKey)
	client.DefaultTimeout = 120 // seconds
	client.PollingInterval = 5  // seconds
	return &Solver{
		client: client,
	}
}

// AmazonWAFRequest represents parameters for Amazon WAF captcha
type AmazonWAFRequest struct {
	SiteKey         string `json:"sitekey"`
	URL             string `json:"url"`
	IV              string `json:"iv"`
	Context         string `json:"context"`
	ChallengeScript string `json:"challenge_script"`
	CaptchaScript   string `json:"captcha_script"`
}

// AmazonWAFResponse represents solved captcha result
type AmazonWAFResponse struct {
	Token    string        `json:"token"`
	SolvedIn time.Duration `json:"solved_in"`
	Cost     string        `json:"cost"`
}

// SolveAmazonWAF solves Amazon WAF captcha using 2Captcha
func (s *Solver) SolveAmazonWAF(req AmazonWAFRequest) (*AmazonWAFResponse, error) {
	startTime := time.Now()

	cap := api2captcha.AmazonWAF{
		SiteKey:         req.SiteKey,
		Iv:              req.IV,
		Context:         req.Context,
		Url:             req.URL,
		ChallengeScript: req.ChallengeScript,
		CaptchaScript:   req.CaptchaScript,
	}

	code, _, err := s.client.Solve(cap.ToRequest())
	if err != nil {
		return nil, fmt.Errorf("2captcha solve failed: %w", err)
	}

	solvedIn := time.Since(startTime)

	return &AmazonWAFResponse{
		Token:    code,
		SolvedIn: solvedIn,
		Cost:     "~$2.99",
	}, nil
}

// GetBalance returns current 2Captcha balance
func (s *Solver) GetBalance() (float64, error) {
	return s.client.GetBalance()
}
