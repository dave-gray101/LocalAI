package routes

import (
	"github.com/go-skynet/LocalAI/core"
	"github.com/go-skynet/LocalAI/core/http/endpoints/localai"
	"github.com/go-skynet/LocalAI/core/http/endpoints/openai"
	"github.com/go-skynet/LocalAI/core/http/middleware"
	"github.com/go-skynet/LocalAI/pkg/model"
	"github.com/gofiber/fiber/v2"
)

func RegisterOpenAIRoutes(app *fiber.App,
	application *core.Application,
	requestExtractor *middleware.RequestExtractor) {

	requestExtractorMiddleware := middleware.NewRequestExtractor(application.ModelLoader, application.ApplicationConfig)
	// openAI compatible API endpoint

	// chat
	chatChain := []fiber.Handler{
		requestExtractorMiddleware.SetModelName,
		requestExtractor.SetDefaultModelNameToFirstAvailable,
		requestExtractorMiddleware.SetOpenAIRequest,
		openai.ChatEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig),
	}
	app.Post("/v1/chat/completions", chatChain...)
	app.Post("/chat/completions", chatChain...)

	// edit
	editChain := []fiber.Handler{
		requestExtractorMiddleware.SetModelName,
		requestExtractor.SetDefaultModelNameToFirstAvailable,
		requestExtractorMiddleware.SetOpenAIRequest,
		openai.EditEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig),
	}
	app.Post("/v1/edits", editChain...)
	app.Post("/edits", editChain...)

	// assistant
	app.Get("/v1/assistants", openai.ListAssistantsEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Get("/assistants", openai.ListAssistantsEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Post("/v1/assistants", openai.CreateAssistantEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Post("/assistants", openai.CreateAssistantEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Delete("/v1/assistants/:assistant_id", openai.DeleteAssistantEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Delete("/assistants/:assistant_id", openai.DeleteAssistantEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Get("/v1/assistants/:assistant_id", openai.GetAssistantEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Get("/assistants/:assistant_id", openai.GetAssistantEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Post("/v1/assistants/:assistant_id", openai.ModifyAssistantEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Post("/assistants/:assistant_id", openai.ModifyAssistantEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Get("/v1/assistants/:assistant_id/files", openai.ListAssistantFilesEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Get("/assistants/:assistant_id/files", openai.ListAssistantFilesEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Post("/v1/assistants/:assistant_id/files", openai.CreateAssistantFileEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Post("/assistants/:assistant_id/files", openai.CreateAssistantFileEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Delete("/v1/assistants/:assistant_id/files/:file_id", openai.DeleteAssistantFileEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Delete("/assistants/:assistant_id/files/:file_id", openai.DeleteAssistantFileEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Get("/v1/assistants/:assistant_id/files/:file_id", openai.GetAssistantFileEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Get("/assistants/:assistant_id/files/:file_id", openai.GetAssistantFileEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))

	// files
	app.Post("/v1/files", openai.UploadFilesEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Post("/files", openai.UploadFilesEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Get("/v1/files", openai.ListFilesEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Get("/files", openai.ListFilesEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Get("/v1/files/:file_id", openai.GetFilesEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Get("/files/:file_id", openai.GetFilesEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Delete("/v1/files/:file_id", openai.DeleteFilesEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Delete("/files/:file_id", openai.DeleteFilesEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Get("/v1/files/:file_id/content", openai.GetFilesContentsEndpoint(application.BackendConfigLoader, application.ApplicationConfig))
	app.Get("/files/:file_id/content", openai.GetFilesContentsEndpoint(application.BackendConfigLoader, application.ApplicationConfig))

	// completion
	completionChain := []fiber.Handler{
		requestExtractorMiddleware.SetModelName,
		requestExtractor.SetDefaultModelNameToFirstAvailable,
		requestExtractorMiddleware.SetOpenAIRequest,
		openai.CompletionEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig),
	}
	app.Post("/v1/completions", completionChain...)
	app.Post("/completions", completionChain...)
	app.Post("/v1/engines/:model/completions", completionChain...)

	// embeddings
	embeddingChain := []fiber.Handler{
		requestExtractorMiddleware.SetModelName,
		requestExtractorMiddleware.SetOpenAIRequest,
		openai.EmbeddingsEndpoint(application.EmbeddingsBackendService),
	}
	app.Post("/v1/embeddings", embeddingChain...)
	app.Post("/embeddings", embeddingChain...)
	app.Post("/v1/engines/:model/embeddings", embeddingChain...)

	// audio
	app.Post("/v1/audio/transcriptions", requestExtractorMiddleware.SetModelName, requestExtractorMiddleware.SetOpenAIRequest, openai.TranscriptEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig))
	app.Post("/v1/audio/speech", requestExtractor.SetModelName, localai.TTSEndpoint(application.TextToSpeechBackendService))

	// images
	imageChain := []fiber.Handler{ // Currently only used once, but makes it easier to read?
		requestExtractorMiddleware.SetModelName,
		requestExtractor.BuildConstantDefaultModelNameMiddleware(model.StableDiffusionBackend), // This is the previous value - is it correct?
		requestExtractorMiddleware.SetOpenAIRequest,
		openai.ImageEndpoint(application.BackendConfigLoader, application.ModelLoader, application.ApplicationConfig),
	}
	app.Post("/v1/images/generations", imageChain...)

	if application.ApplicationConfig.ImageDir != "" {
		app.Static("/generated-images", application.ApplicationConfig.ImageDir)
	}

	if application.ApplicationConfig.AudioDir != "" {
		app.Static("/generated-audio", application.ApplicationConfig.AudioDir)
	}

	// models
	app.Get("/v1/models", openai.ListModelsEndpoint(application.ListModelsService))
	app.Get("/models", openai.ListModelsEndpoint(application.ListModelsService))
}
