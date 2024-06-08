package middleware

import (
	"errors"
	"strings"

	"github.com/go-skynet/LocalAI/core/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
)

func readAuthHeader(c *fiber.Ctx) string {
	authHeader := c.Get("Authorization")

	// elevenlabs
	xApiKey := c.Get("xi-api-key")
	if xApiKey != "" {
		authHeader = "Bearer " + xApiKey
	}

	// anthropic
	xApiKey = c.Get("x-api-key")
	if xApiKey != "" {
		authHeader = "Bearer " + xApiKey
	}

	return authHeader
}

// Creates the auth middleware responsible for checking if API key is valid. If no API key is set, no auth is required.
func GetAuth(applicationConfig *config.ApplicationConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if len(applicationConfig.ApiKeys) == 0 {
			return c.Next()
		}

		if len(applicationConfig.ApiKeys) == 0 {
			return c.Next()
		}

		authHeader := readAuthHeader(c)
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Authorization header missing"})
		}

		// If it's a bearer token
		authHeaderParts := strings.Split(authHeader, " ")
		if len(authHeaderParts) != 2 || authHeaderParts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid Authorization header format"})
		}

		apiKey := authHeaderParts[1]
		for _, key := range applicationConfig.ApiKeys {
			if apiKey == key {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid API key"})
	}
}

const CONTEXT_LOCALS_KEY_API_KEY = "API_KEY"

func GetKeyAuthConfig(applicationConfig *config.ApplicationConfig) keyauth.Config {
	return keyauth.Config{
		KeyLookup:    "header:Authorization|header:x-api-key|header:xi-api-key",
		Validator:    getApiKeyValidationFunction(applicationConfig),
		ErrorHandler: getApiKeyErrorHandler(applicationConfig),
		AuthScheme:   "Bearer",
		ContextKey:   CONTEXT_LOCALS_KEY_API_KEY,
	}
}

func getApiKeyErrorHandler(applicationConfig *config.ApplicationConfig) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		if errors.Is(err, ErrMissingOrMalformedAPIKey) {
			if len(applicationConfig.ApiKeys) == 0 {
				return ctx.Next() // if no keys are set up, any error we get here is not an error.
			}
			if applicationConfig.OpaqueErrors {
				return ctx.SendStatus(403)
			}
		}
		if applicationConfig.OpaqueErrors {
			return ctx.SendStatus(500)
		}
		return err
	}
}

func getApiKeyValidationFunction(applicationConfig *config.ApplicationConfig) func(*fiber.Ctx, string) (bool, error) {

	return func(ctx *fiber.Ctx, apiKey string) (bool, error) {
		if len(applicationConfig.ApiKeys) == 0 {
			return true, nil // If no keys are setup, accept everything
		}
	}
}
