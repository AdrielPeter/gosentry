package main

import (
	"goSentry/controllers/users"
	"goSentry/database"
	"log"
	"time"

	usersModel "goSentry/controllers/users/models"

	"github.com/getsentry/sentry-go"
	sentryfiber "github.com/getsentry/sentry-go/fiber"
	sentryotel "github.com/getsentry/sentry-go/otel"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"

	otelfiber "github.com/gofiber/contrib/otelfiber/v2"

	otelgorm "github.com/uptrace/opentelemetry-go-extra/otelgorm"
)

func main() {
	// ... (código de inicialização do Sentry e OpenTelemetry)

	// Initialize Sentry
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              "https://634e7a840fa4ab2f67c5091dd4b943bc@o4509395197034496.ingest.us.sentry.io/4509492844036096",
		EnableTracing:    true,
		TracesSampleRate: 1.0,
		SendDefaultPII:   true,
		Debug:            true,
	}); err != nil {
		log.Fatalf("sentry.Init: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	// Configure OpenTelemetry com Sentry Span Processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(sentryotel.NewSentryPropagator())

	// Inicializa o banco de dados.
	database.InitDB()
	database.DB.AutoMigrate(&usersModel.Users{}) // Garanta que a tabela Users seja migrada
	database.DB.Use(otelgorm.NewPlugin())

	app := fiber.New()

	// Use o middleware OpenTelemetry Fiber primeiro
	app.Use(otelfiber.Middleware())

	// Em seguida, use o middleware Sentry Fiber
	app.Use(sentryfiber.New(sentryfiber.Options{
		Repanic:         true,
		WaitForDelivery: true,
		Timeout:         2 * time.Second,
	}))

	// Rota para criar um novo usuário
	app.Post("/users", func(c *fiber.Ctx) error {
		spanCtx := oteltrace.SpanContextFromContext(c.UserContext())
		log.Printf("Handler POST /users - OTel TraceID: %s, SpanID: %s\n", spanCtx.TraceID().String(), spanCtx.SpanID().String())

		// Create child span for request parsing
		tracer := otel.Tracer("goSentry/handlers")
		ctx, parseSpan := tracer.Start(c.UserContext(), "request.parsing", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
		parseSpan.SetAttributes(
			attribute.String("http.method", "POST"),
			attribute.String("http.route", "/users"),
			attribute.String("http.url", c.OriginalURL()),
		)

		var newUser usersModel.Users // Struct para receber os dados do corpo da requisição

		// Faz o parsing do corpo da requisição para a struct newUser
		if err := c.BodyParser(&newUser); err != nil {
			parseSpan.SetAttributes(
				attribute.String("parsing.error", err.Error()),
				attribute.Bool("parsing.success", false),
			)
			parseSpan.End()
			log.Printf("Error parsing request body: %v", err)
			sentry.CaptureException(err) // Captura o erro no Sentry
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message":  "Invalid request body",
				"trace_id": spanCtx.TraceID().String(),
				"span_id":  spanCtx.SpanID().String(),
			})
		}
		parseSpan.SetAttributes(attribute.Bool("parsing.success", true))
		parseSpan.End()

		// Create child span for validation
		ctx, validationSpan := tracer.Start(ctx, "request.validation", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
		validationSpan.SetAttributes(
			attribute.String("user.username", newUser.Username),
			attribute.String("user.email", newUser.Email),
		)

		// Validação básica dos campos (apenas Username e Email)
		if newUser.Username == "" || newUser.Email == "" {
			validationSpan.SetAttributes(
				attribute.String("validation.error", "missing_required_fields"),
				attribute.Bool("validation.success", false),
			)
			validationSpan.End()
			log.Println("Username and Email are required")
			sentry.CaptureMessage("Missing required fields for user creation")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message":  "Username and Email are required",
				"trace_id": spanCtx.TraceID().String(),
				"span_id":  spanCtx.SpanID().String(),
			})
		}
		validationSpan.SetAttributes(attribute.Bool("validation.success", true))
		validationSpan.End()

		// Create child span for user creation
		ctx, createSpan := tracer.Start(ctx, "user.creation", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))

		// Chama a função CreateUser2, passando o contexto da requisição, a instância do DB
		// e a struct newUser com os dados já parseados.
		if err := users.CreateUser2(ctx, database.DB, newUser); err != nil {
			createSpan.SetAttributes(
				attribute.String("creation.error", err.Error()),
				attribute.Bool("creation.success", false),
			)
			createSpan.End()
			// O erro já foi capturado dentro de users.CreateUser se for um erro de DB
			// ou você pode capturar erros específicos aqui se a função retornar erros de validação
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message":  "Failed to create user",
				"trace_id": spanCtx.TraceID().String(),
				"span_id":  spanCtx.SpanID().String(),
			})
		}

		createSpan.SetAttributes(attribute.Bool("creation.success", true))
		createSpan.End()

		log.Println("User created successfully")
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message":  "User created successfully",
			"data":     newUser, // Retorna os dados do usuário criado
			"trace_id": spanCtx.TraceID().String(),
			"span_id":  spanCtx.SpanID().String(),
		})
	})

	// Rota para listar usuários
	app.Get("/users", func(c *fiber.Ctx) error {
		spanCtx := oteltrace.SpanContextFromContext(c.UserContext())
		log.Printf("Handler GET /users - OTel TraceID: %s, SpanID: %s\n", spanCtx.TraceID().String(), spanCtx.SpanID().String())

		// Create child span for user retrieval
		tracer := otel.Tracer("goSentry/handlers")
		ctx, retrieveSpan := tracer.Start(c.UserContext(), "user.retrieval", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
		retrieveSpan.SetAttributes(
			attribute.String("http.method", "GET"),
			attribute.String("http.route", "/users"),
			attribute.String("http.url", c.OriginalURL()),
		)

		list, err := users.GetUsers2(ctx, database.DB)
		if err != nil {
			retrieveSpan.SetAttributes(
				attribute.String("retrieval.error", err.Error()),
				attribute.Bool("retrieval.success", false),
			)
			retrieveSpan.End()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message":  "failed to fetch users",
				"trace_id": spanCtx.TraceID().String(),
				"span_id":  spanCtx.SpanID().String(),
			})
		}

		retrieveSpan.SetAttributes(
			attribute.Bool("retrieval.success", true),
			attribute.Int("users.count", len(list)),
		)
		retrieveSpan.End()

		if len(list) == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message":  "no users found",
				"trace_id": spanCtx.TraceID().String(),
				"span_id":  spanCtx.SpanID().String(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":  "users found",
			"data":     list,
			"trace_id": spanCtx.TraceID().String(),
			"span_id":  spanCtx.SpanID().String(),
		})
	})

	log.Fatal(app.Listen(":3123"))
}
