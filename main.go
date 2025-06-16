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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

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

	// Use o middleware Sentry Fiber
	app.Use(sentryfiber.New(sentryfiber.Options{
		Repanic:         true,
		WaitForDelivery: true,
		Timeout:         2 * time.Second,
	}))

	app.Use(otelfiber.Middleware())

	// Rota para criar um novo usuário
	app.Post("/users", func(c *fiber.Ctx) error {
		var newUser usersModel.Users // Struct para receber os dados do corpo da requisição

		// Faz o parsing do corpo da requisição para a struct newUser
		if err := c.BodyParser(&newUser); err != nil {
			log.Printf("Error parsing request body: %v", err)
			sentry.CaptureException(err) // Captura o erro no Sentry
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Invalid request body",
			})
		}

		// Validação básica dos campos (apenas Username e Email)
		if newUser.Username == "" || newUser.Email == "" {
			log.Println("Username and Email are required")
			sentry.CaptureMessage("Missing required fields for user creation")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Username and Email are required",
			})
		}

		// Chama a função CreateUser, passando o contexto da requisição, a instância do DB
		// e a struct newUser com os dados já parseados.
		if err := users.CreateUser2(c.UserContext(), database.DB, newUser); err != nil {
			// O erro já foi capturado dentro de users.CreateUser se for um erro de DB
			// ou você pode capturar erros específicos aqui se a função retornar erros de validação
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to create user",
			})
		}

		log.Println("User created successfully")
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "User created successfully",
			"data":    newUser, // Retorna os dados do usuário criado
		})
	})

	// routes.go ou no main, logo depois de criar o app e injetar o DB
	app.Get("/users", func(c *fiber.Ctx) error {
		// 1. Busca no banco usando o contexto da requisição (para manter o trace)
		list, err := users.GetUsers2(c.UserContext(), database.DB)
		if err != nil {
			// já está logado/CaptureException lá dentro; aqui só devolvemos HTTP 500
			return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch users")
		}

		// 2. Nenhum registro encontrado → HTTP 404
		if len(list) == 0 {
			return fiber.NewError(fiber.StatusNotFound, "no users found")
		}

		// 3. Sucesso → HTTP 200
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "users found",
			"data":    list, // slice retornado pelo use-case
		})
	})

	// ... (outras rotas e log.Fatal(app.Listen(":3123")))
	log.Fatal(app.Listen(":3123"))
}
