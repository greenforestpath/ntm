// Package redaction provides detection and redaction of sensitive content.
//
// NOTE: This package is a compatibility wrapper. The canonical engine now lives in
// `internal/redaction`.
package redaction

import redactionlib "github.com/Dicklesworthstone/ntm/internal/redaction"

type Mode = redactionlib.Mode

const (
	ModeOff    = redactionlib.ModeOff
	ModeWarn   = redactionlib.ModeWarn
	ModeRedact = redactionlib.ModeRedact
	ModeBlock  = redactionlib.ModeBlock
)

type Category = redactionlib.Category

const (
	CategoryOpenAIKey     = redactionlib.CategoryOpenAIKey
	CategoryAnthropicKey  = redactionlib.CategoryAnthropicKey
	CategoryGitHubToken   = redactionlib.CategoryGitHubToken
	CategoryAWSAccessKey  = redactionlib.CategoryAWSAccessKey
	CategoryAWSSecretKey  = redactionlib.CategoryAWSSecretKey
	CategoryJWT           = redactionlib.CategoryJWT
	CategoryGoogleAPIKey  = redactionlib.CategoryGoogleAPIKey
	CategoryPrivateKey    = redactionlib.CategoryPrivateKey
	CategoryDatabaseURL   = redactionlib.CategoryDatabaseURL
	CategoryPassword      = redactionlib.CategoryPassword
	CategoryGenericAPIKey = redactionlib.CategoryGenericAPIKey
	CategoryGenericSecret = redactionlib.CategoryGenericSecret
	CategoryBearerToken   = redactionlib.CategoryBearerToken
)

type Finding = redactionlib.Finding
type Result = redactionlib.Result
type Config = redactionlib.Config
type ConfigError = redactionlib.ConfigError

func DefaultConfig() Config { return redactionlib.DefaultConfig() }
