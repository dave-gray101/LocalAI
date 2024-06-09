package middleware

import (
	"errors"

	"github.com/go-skynet/LocalAI/core/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
)

// This file contains the configuration generators and handler functions that are used along with the fiber/keyauth middleware
// Currently this requires an upstream patch - tmp-keyauth.go contains temporary code that will be removed if my patch is accepted.

const CONTEXT_LOCALS_KEY_API_KEY = "API_KEY"

func GetKeyAuthConfig(applicationConfig *config.ApplicationConfig) KAConfig {
	return KAConfig{
		KeyLookup:            "header:Authorization",
		AdditionalKeyLookups: []string{"header:x-api-key", "header:xi-api-key"},
		Validator:            getApiKeyValidationFunction(applicationConfig),
		ErrorHandler:         getApiKeyErrorHandler(applicationConfig),
		AuthScheme:           "Bearer",
		ContextKey:           CONTEXT_LOCALS_KEY_API_KEY,
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
		for _, validKey := range applicationConfig.ApiKeys {
			if apiKey == validKey {
				return true, nil
			}
		}
		return false, keyauth.ErrMissingOrMalformedAPIKey
	}
}
