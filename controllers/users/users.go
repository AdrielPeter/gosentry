package users

import (
	"context"
	"fmt"
	"goSentry/controllers/users/models"
	"log"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func GetUsers(c *fiber.Ctx, ctx context.Context, db *gorm.DB) error {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}

	// Start span for database operation with enhanced metadata
	dbSpan := sentry.StartSpan(ctx, "db.query")
	dbSpan.Description = "SELECT * FROM users"
	dbSpan.SetData("db.system", "postgresql")
	dbSpan.SetData("db.type", "SELECT")
	dbSpan.SetData("db.table", "users")

	var users []models.Users
	if err := db.WithContext(dbSpan.Context()).Find(&users).Error; err != nil {
		dbSpan.Finish()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "DB error"})
	}
	dbSpan.Finish()

	// Start span for response handling
	responseSpan := sentry.StartSpan(ctx, "function")
	responseSpan.Description = "Process and return user data"
	responseSpan.SetTag("url", c.OriginalURL())
	responseSpan.SetData("query_params", c.Request().URI().QueryArgs().String())

	if len(users) == 0 {
		responseSpan.Finish()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "No users found"})
	}
	responseSpan.Finish()

	return c.JSON(fiber.Map{"message": "Users found", "data": users})
}

func CreateUser(c *fiber.Ctx, db *gorm.DB) error {
	var user models.Users
	if err := c.BodyParser(&user); err != nil {
		sentry.CaptureException(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}
	if user.Username == "" || user.Email == "" {
		err := fiber.NewError(fiber.StatusBadRequest, "Username and Email are required")
		sentry.CaptureException(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Username and Email are required",
		})
	}
	if err := db.WithContext(c.UserContext()).Create(&user).Error; err != nil {
		sentry.CaptureException(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to create user",
		})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"data":    user,
	})
}

func CreateUser2(ctx context.Context, db *gorm.DB, user models.Users) error {
	// Use OpenTelemetry Tracer to explicitly create a span
	tracer := otel.Tracer("goSentry/users")
	// Ensure the span is a child of the incoming context (from Fiber middleware)
	ctx, span := tracer.Start(ctx, "db.insert", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	spanCtx := trace.SpanContextFromContext(ctx)
	log.Printf("CreateUser2 (OTel) - TraceID: %s, SpanID: %s\n", spanCtx.TraceID().String(), spanCtx.SpanID().String())

	// Add attributes to the span
	span.SetAttributes(
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "users"),
		attribute.String("user.username", user.Username),
		attribute.String("user.email", user.Email),
	)

	// Create child span for validation
	ctx, validationSpan := tracer.Start(ctx, "validation", trace.WithSpanKind(trace.SpanKindInternal))
	if user.Username == "" || user.Email == "" {
		validationSpan.SetAttributes(
			attribute.String("validation.error", "missing_required_fields"),
			attribute.Bool("validation.success", false),
		)
		validationSpan.End()
		sentry.CaptureMessage("Username and Email are required for user creation")
		return fmt.Errorf("username and email are required")
	}
	validationSpan.SetAttributes(attribute.Bool("validation.success", true))
	validationSpan.End()

	// Create child span for database operation
	ctx, dbSpan := tracer.Start(ctx, "db.create", trace.WithSpanKind(trace.SpanKindClient))
	dbSpan.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "CREATE"),
		attribute.String("db.table", "users"),
	)

	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		dbSpan.SetAttributes(
			attribute.String("db.error", err.Error()),
			attribute.Bool("db.success", false),
		)
		dbSpan.End()
		sentry.CaptureException(err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	dbSpan.SetAttributes(
		attribute.Bool("db.success", true),
		attribute.Int("db.rows_affected", 1),
	)
	dbSpan.End()

	return nil
}

func GetUsers2(ctx context.Context, db *gorm.DB) ([]models.Users, error) {
	// Use OpenTelemetry Tracer to explicitly create a span
	tracer := otel.Tracer("goSentry/users")
	// Ensure the span is a child of the incoming context (from Fiber middleware)
	ctx, span := tracer.Start(ctx, "db.query", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	spanCtx := trace.SpanContextFromContext(ctx)
	log.Printf("GetUsers2 (OTel) - TraceID: %s, SpanID: %s\n", spanCtx.TraceID().String(), spanCtx.SpanID().String())

	// Add attributes to the span
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "users"),
		attribute.String("db.system", "postgresql"),
	)

	// Create child span for database query
	ctx, dbSpan := tracer.Start(ctx, "db.find", trace.WithSpanKind(trace.SpanKindClient))
	dbSpan.SetAttributes(
		attribute.String("db.query", "SELECT * FROM users"),
		attribute.String("db.operation", "FIND"),
	)

	var users []models.Users

	if err := db.WithContext(ctx).Find(&users).Error; err != nil {
		dbSpan.SetAttributes(
			attribute.String("db.error", err.Error()),
			attribute.Bool("db.success", false),
		)
		dbSpan.End()
		sentry.CaptureException(err)
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	dbSpan.SetAttributes(
		attribute.Bool("db.success", true),
		attribute.Int("db.rows_returned", len(users)),
	)
	dbSpan.End()

	// Create child span for data processing
	ctx, processSpan := tracer.Start(ctx, "data.processing", trace.WithSpanKind(trace.SpanKindInternal))
	processSpan.SetAttributes(
		attribute.Int("users.count", len(users)),
		attribute.Bool("processing.success", true),
	)
	processSpan.End()

	return users, nil
}
