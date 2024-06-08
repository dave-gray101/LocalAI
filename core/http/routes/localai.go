package routes

import (
	"github.com/go-skynet/LocalAI/core"
	"github.com/go-skynet/LocalAI/core/http/endpoints/localai"
	"github.com/go-skynet/LocalAI/core/http/middleware"
	"github.com/go-skynet/LocalAI/internal"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

func RegisterLocalAIRoutes(app *fiber.App,
	application *core.Application,
	requestExtractor *middleware.RequestExtractor) {

	app.Get("/swagger/*", swagger.HandlerDefault) // default

	// LocalAI API endpoints

	modelGalleryEndpointService := localai.CreateModelGalleryEndpointService(application.ApplicationConfig.Galleries, application.ApplicationConfig.ModelPath, application.GalleryService)
	app.Post("/models/apply", modelGalleryEndpointService.ApplyModelGalleryEndpoint())
	app.Post("/models/delete/:name", modelGalleryEndpointService.DeleteModelGalleryEndpoint())

	app.Get("/models/available", modelGalleryEndpointService.ListModelFromGalleryEndpoint())
	app.Get("/models/galleries", modelGalleryEndpointService.ListModelGalleriesEndpoint())
	app.Post("/models/galleries", modelGalleryEndpointService.AddModelGalleryEndpoint())
	app.Delete("/models/galleries", modelGalleryEndpointService.RemoveModelGalleryEndpoint())
	app.Get("/models/jobs/:uuid", modelGalleryEndpointService.GetOpStatusEndpoint())
	app.Get("/models/jobs", modelGalleryEndpointService.GetAllStatusEndpoint())

	app.Post("/tts", requestExtractor.SetModelName, localai.TTSEndpoint(application.TextToSpeechBackendService))

	// Stores : TODO IS THIS REALLY A SERVICE? OR IS IT PURELY WEB API FEATURE?
	app.Post("/stores/set", localai.StoresSetEndpoint(application.StoresLoader, application.ApplicationConfig))
	app.Post("/stores/delete", localai.StoresDeleteEndpoint(application.StoresLoader, application.ApplicationConfig))
	app.Post("/stores/get", localai.StoresGetEndpoint(application.StoresLoader, application.ApplicationConfig))
	app.Post("/stores/find", localai.StoresFindEndpoint(application.StoresLoader, application.ApplicationConfig))

	// Kubernetes health checks
	ok := func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	}

	app.Get("/healthz", ok)
	app.Get("/readyz", ok)

	app.Get("/metrics", localai.LocalAIMetricsEndpoint())

	app.Get("/backend/monitor", localai.BackendMonitorEndpoint(application.BackendMonitorService))
	app.Post("/backend/shutdown", localai.BackendShutdownEndpoint(application.BackendMonitorService))

	app.Get("/version", func(c *fiber.Ctx) error {
		return c.JSON(struct {
			Version string `json:"version"`
		}{Version: internal.PrintableVersion()})
	})

}
