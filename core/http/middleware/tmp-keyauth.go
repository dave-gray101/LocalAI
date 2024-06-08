// TODO: I'm trying to upstream this middleware to fiber.
// Ideally, this file will be removed.
// In the meantime, this is a copy of fiber's keyauth, with my patch applied.

// Special thanks to Echo: https://github.com/labstack/echo/blob/master/middleware/key_auth.go
package middleware

import (
	"errors"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
)

// When there is no request of the key thrown ErrMissingOrMalformedAPIKey
var ErrMissingOrMalformedAPIKey = errors.New("missing or malformed API Key")

const (
	query  = "query"
	form   = "form"
	param  = "param"
	cookie = "cookie"
)

type extractorFunc func(c *fiber.Ctx) (string, error)

// New creates a new middleware handler
func NewKeyAuth(config ...keyauth.Config) fiber.Handler {
	// Init config
	cfg := configDefault(config...)

	// Initialize

	parts := strings.Split(cfg.KeyLookup, "|")

	var extractor extractorFunc
	if len(parts) <= 1 {
		extractor = parseSingleExtractor(cfg.KeyLookup, cfg.AuthScheme)
	} else {
		subExtractors := []extractorFunc{}
		for _, keyLookup := range parts {
			subExtractors = append(subExtractors, parseSingleExtractor(keyLookup, cfg.AuthScheme))
		}
		extractor = func(c *fiber.Ctx) (string, error) {
			for _, subExtractor := range subExtractors {
				res, err := subExtractor(c)
				if err == nil && res != "" {
					return res, nil
				}
				if !errors.Is(err, ErrMissingOrMalformedAPIKey) {
					return "", err
				}
			}
			return "", ErrMissingOrMalformedAPIKey
		}
	}

	// Return middleware handler
	return func(c *fiber.Ctx) error {
		// Filter request to skip middleware
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		// Extract and verify key
		key, err := extractor(c)
		if err != nil {
			return cfg.ErrorHandler(c, err)
		}

		valid, err := cfg.Validator(c, key)

		if err == nil && valid {
			c.Locals(cfg.ContextKey, key)
			return cfg.SuccessHandler(c)
		}
		return cfg.ErrorHandler(c, err)
	}
}

func parseSingleExtractor(keyLookup string, authScheme string) extractorFunc {
	parts := strings.Split(keyLookup, ":")
	extractor := keyFromHeader(parts[1], authScheme)
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
	return extractor
}

// keyFromHeader returns a function that extracts api key from the request header.
func keyFromHeader(header, authScheme string) extractorFunc {
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
func keyFromQuery(param string) extractorFunc {
	return func(c *fiber.Ctx) (string, error) {
		key := c.Query(param)
		if key == "" {
			return "", ErrMissingOrMalformedAPIKey
		}
		return key, nil
	}
}

// keyFromForm returns a function that extracts api key from the form.
func keyFromForm(param string) extractorFunc {
	return func(c *fiber.Ctx) (string, error) {
		key := c.FormValue(param)
		if key == "" {
			return "", ErrMissingOrMalformedAPIKey
		}
		return key, nil
	}
}

// keyFromParam returns a function that extracts api key from the url param string.
func keyFromParam(param string) extractorFunc {
	return func(c *fiber.Ctx) (string, error) {
		key, err := url.PathUnescape(c.Params(param))
		if err != nil {
			return "", ErrMissingOrMalformedAPIKey
		}
		return key, nil
	}
}

// keyFromCookie returns a function that extracts api key from the named cookie.
func keyFromCookie(name string) extractorFunc {
	return func(c *fiber.Ctx) (string, error) {
		key := c.Cookies(name)
		if key == "" {
			return "", ErrMissingOrMalformedAPIKey
		}
		return key, nil
	}
}

// from config.go ::

// Helper function to set default values
func configDefault(config ...keyauth.Config) keyauth.Config {
	// Return default config if nothing provided
	if len(config) < 1 {
		return keyauth.ConfigDefault
	}

	// Override default config
	cfg := config[0]

	// Set default values
	if cfg.SuccessHandler == nil {
		cfg.SuccessHandler = keyauth.ConfigDefault.SuccessHandler
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = keyauth.ConfigDefault.ErrorHandler
	}
	if cfg.KeyLookup == "" {
		cfg.KeyLookup = keyauth.ConfigDefault.KeyLookup
		// set AuthScheme as "Bearer" only if KeyLookup is set to default.
		if cfg.AuthScheme == "" {
			cfg.AuthScheme = keyauth.ConfigDefault.AuthScheme
		}
	}
	if cfg.Validator == nil {
		panic("fiber: keyauth middleware requires a validator function")
	}
	if cfg.ContextKey == nil {
		cfg.ContextKey = keyauth.ConfigDefault.ContextKey
	}

	return cfg
}
