// TODO: I am upstreaming these changes to the keyauth middleware in fiber v3
// It's no longer possible to make feature improvements to the v2 branch, which we currently use.
// Until v3 becomes stable and we migrate, this file is a trivially-backported copy of <URL HERE>

// Special thanks to Echo: https://github.com/labstack/echo/blob/master/middleware/key_auth.go
package middleware

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// config.go
type KeyauthKeyLookupFunc func(c *fiber.Ctx) (string, error)

// KAConfig defines the config for middleware.
type KAConfig struct {
	// Next defines a function to skip middleware.
	// Optional. Default: nil
	Next func(*fiber.Ctx) bool

	// SuccessHandler defines a function which is executed for a valid key.
	// Optional. Default: nil
	SuccessHandler fiber.Handler

	// ErrorHandler defines a function which is executed for an invalid key.
	// It may be used to define a custom error.
	// Optional. Default: 401 Invalid or expired key
	ErrorHandler fiber.ErrorHandler

	// KeyLookup is a string in the form of "<source>:<name>" that is used
	// to extract key from the request.
	// Optional. Default value "header:Authorization".
	// Possible values:
	// - "header:<name>"
	// - "query:<name>"
	// - "form:<name>"
	// - "param:<name>"
	// - "cookie:<name>"
	KeyLookup string

	CustomKeyLookup KeyauthKeyLookupFunc

	// AuthScheme to be used in the Authorization header.
	// Optional. Default value "Bearer".
	AuthScheme string

	// Validator is a function to validate key.
	Validator func(*fiber.Ctx, string) (bool, error)
}

// ConfigDefault is the default config
var ConfigDefault = KAConfig{
	SuccessHandler: func(c *fiber.Ctx) error {
		return c.Next()
	},
	ErrorHandler: func(c *fiber.Ctx, err error) error {
		if errors.Is(err, ErrMissingOrMalformedAPIKey) {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired API Key")
	},
	KeyLookup:       "header:" + fiber.HeaderAuthorization,
	CustomKeyLookup: nil,
	AuthScheme:      "Bearer",
}

// Helper function to set default values
func configDefault(config ...KAConfig) KAConfig {
	// Return default config if nothing provided
	if len(config) < 1 {
		return ConfigDefault
	}

	// Override default config
	cfg := config[0]

	// Set default values
	if cfg.SuccessHandler == nil {
		cfg.SuccessHandler = ConfigDefault.SuccessHandler
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = ConfigDefault.ErrorHandler
	}
	if cfg.KeyLookup == "" {
		cfg.KeyLookup = ConfigDefault.KeyLookup
		// set AuthScheme as "Bearer" only if KeyLookup is set to default.
		if cfg.AuthScheme == "" {
			cfg.AuthScheme = ConfigDefault.AuthScheme
		}
	}
	if cfg.Validator == nil {
		panic("fiber: keyauth middleware requires a validator function")
	}

	return cfg
}

// keyauth.go

// The contextKey type is unexported to prevent collisions with context keys defined in
// other packages.
type contextKey int

// The keys for the values in context
const (
	tokenKey contextKey = 0
)

// When there is no request of the key thrown ErrMissingOrMalformedAPIKey
var ErrMissingOrMalformedAPIKey = errors.New("missing or malformed API Key")

const (
	query  = "query"
	form   = "form"
	param  = "param"
	cookie = "cookie"
)

// New creates a new middleware handler
func NewKeyAuth(config ...KAConfig) fiber.Handler {
	// Init config
	cfg := configDefault(config...)

	// Initialize
	if cfg.CustomKeyLookup == nil {
		var err error
		cfg.CustomKeyLookup, err = SingleKeyLookup(cfg.KeyLookup, cfg.AuthScheme)
		if err != nil {
			panic(fmt.Errorf("unable to create lookup function: %w", err))
		}
	}

	// Return middleware handler
	return func(c *fiber.Ctx) error {
		// Filter request to skip middleware
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		// Extract and verify key
		key, err := cfg.CustomKeyLookup(c)
		if err != nil {
			return cfg.ErrorHandler(c, err)
		}

		valid, err := cfg.Validator(c, key)

		if err == nil && valid {
			c.Locals(tokenKey, key)
			return cfg.SuccessHandler(c)
		}
		return cfg.ErrorHandler(c, err)
	}
}

// TokenFromContext returns the bearer token from the request context.
// returns an empty string if the token does not exist
func TokenFromContext(c *fiber.Ctx) string {
	token, ok := c.Locals(tokenKey).(string)
	if !ok {
		return ""
	}
	return token
}

// MultipleKeySourceLookup creates a CustomKeyLookup function that checks multiple sources until one is found
// Each element should be specified according to the format used in KeyLookup
func MultipleKeySourceLookup(keyLookups []string, authScheme string) (KeyauthKeyLookupFunc, error) {
	subExtractors := map[string]KeyauthKeyLookupFunc{}
	var err error
	for _, keyLookup := range keyLookups {
		subExtractors[keyLookup], err = SingleKeyLookup(keyLookup, authScheme)
		if err != nil {
			return nil, err
		}
	}
	return func(c *fiber.Ctx) (string, error) {
		for keyLookup, subExtractor := range subExtractors {
			res, err := subExtractor(c)
			if err == nil && res != "" {
				return res, nil
			}
			if !errors.Is(err, ErrMissingOrMalformedAPIKey) {
				return "", fmt.Errorf("[%s] %w", keyLookup, err)
			}
		}
		return "", ErrMissingOrMalformedAPIKey
	}, nil
}

func SingleKeyLookup(keyLookup, authScheme string) (KeyauthKeyLookupFunc, error) {
	parts := strings.Split(keyLookup, ":")
	if len(parts) <= 1 {
		return nil, fmt.Errorf("invalid keyLookup: %s", keyLookup)
	}
	extractor := keyFromHeader(parts[1], authScheme) // in the event of an invalid prefix, it is interpreted as header:
	switch parts[0] {
	case query:
		extractor = keyFromQuery(parts[1])
	case form:
		extractor = keyFromForm(parts[1])
	case param:
		extractor = keyFromParam(parts[1])
	case cookie:
		extractor = keyFromCookie(parts[1])
	}
	return extractor, nil
}

// keyFromHeader returns a function that extracts api key from the request header.
func keyFromHeader(header, authScheme string) KeyauthKeyLookupFunc {
	return func(c *fiber.Ctx) (string, error) {
		auth := c.Get(header)
		l := len(authScheme)
		if len(auth) > 0 && l == 0 {
			return auth, nil
		}
		if len(auth) > l+1 && auth[:l] == authScheme {
			return auth[l+1:], nil
		}
		return "", ErrMissingOrMalformedAPIKey
	}
}

// keyFromQuery returns a function that extracts api key from the query string.
func keyFromQuery(param string) KeyauthKeyLookupFunc {
	return func(c *fiber.Ctx) (string, error) {
		key := c.Query(param)
		if key == "" {
			return "", ErrMissingOrMalformedAPIKey
		}
		return key, nil
	}
}

// keyFromForm returns a function that extracts api key from the form.
func keyFromForm(param string) KeyauthKeyLookupFunc {
	return func(c *fiber.Ctx) (string, error) {
		key := c.FormValue(param)
		if key == "" {
			return "", ErrMissingOrMalformedAPIKey
		}
		return key, nil
	}
}

// keyFromParam returns a function that extracts api key from the url param string.
func keyFromParam(param string) KeyauthKeyLookupFunc {
	return func(c *fiber.Ctx) (string, error) {
		key, err := url.PathUnescape(c.Params(param))
		if err != nil {
			return "", ErrMissingOrMalformedAPIKey
		}
		return key, nil
	}
}

// keyFromCookie returns a function that extracts api key from the named cookie.
func keyFromCookie(name string) KeyauthKeyLookupFunc {
	return func(c *fiber.Ctx) (string, error) {
		key := c.Cookies(name)
		if key == "" {
			return "", ErrMissingOrMalformedAPIKey
		}
		return key, nil
	}
}
