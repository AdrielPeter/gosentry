package users

import (
	"context"
	"fmt"
	"goSentry/controllers/users/models"
	"goSentry/database"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
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
	if err := database.DB.WithContext(c.UserContext()).Create(&user).Error; err != nil {
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

	if user.Username == "" || user.Email == "" {
		return fmt.Errorf("username and email are required")
	}

	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		sentry.CaptureException(err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUsers devolve o slice ou erro; não sabe nada de HTTP.
func GetUsers2(ctx context.Context, db *gorm.DB) ([]models.Users, error) {
    var users []models.Users

    if err := db.WithContext(ctx).Find(&users).Error; err != nil {
        sentry.CaptureException(err) // mantém trace no Sentry
        return nil, fmt.Errorf("failed to fetch users: %w", err)
    }

    return users, nil
}
