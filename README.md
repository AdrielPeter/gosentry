# GoSentry - Application with Sentry Tracing

This is a Go application that demonstrates Sentry integration with OpenTelemetry tracing, including detailed span children for better observability.

## Features

- **Sentry Integration**: Full Sentry integration with error tracking and performance monitoring
- **OpenTelemetry Tracing**: Distributed tracing with detailed span hierarchy
- **Span Children**: Each request creates multiple child spans for different operations:
  - Request parsing
  - Validation
  - Database operations
  - Data processing
- **Tracing in Responses**: All API responses include trace_id and span_id for correlation

## API Endpoints

### POST /users
Creates a new user with detailed tracing:
- `request.parsing` - Parses the request body
- `request.validation` - Validates user data
- `user.creation` - Orchestrates user creation
- `validation` - Validates required fields
- `db.create` - Database insertion operation

### GET /users
Retrieves all users with detailed tracing:
- `user.retrieval` - Orchestrates user retrieval
- `db.query` - Main database query span
- `db.find` - Database find operation
- `data.processing` - Processes retrieved data

## Setup

1. Create a `.env` file with your database configuration:
```env
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=your_password
DATABASE_NAME=gosentry
```

2. Run the application:
```bash
go run main.go
```

## Tracing Information

Each API response includes:
- `trace_id`: The unique identifier for the entire request trace
- `span_id`: The identifier for the current span

This allows you to correlate API responses with the detailed trace information in Sentry.

## Span Hierarchy

The application creates a rich hierarchy of spans:
- HTTP request spans (created by Fiber middleware)
- Handler spans (created by Sentry middleware)
- Custom operation spans (created manually)
- Database operation spans (created by otelgorm)

All spans are automatically sent to Sentry for monitoring and analysis.
