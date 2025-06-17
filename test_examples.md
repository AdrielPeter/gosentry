# Test Examples for GoSentry Tracing

## Testing the API with Tracing

### 1. Create a User (POST /users)

```bash
curl -X POST http://localhost:3123/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com"
  }'
```

**Expected Response:**
```json
{
  "message": "User created successfully",
  "data": {
    "username": "testuser",
    "email": "test@example.com"
  },
  "trace_id": "1234567890abcdef1234567890abcdef",
  "span_id": "abcdef1234567890abcdef1234567890"
}
```

### 2. Get All Users (GET /users)

```bash
curl -X GET http://localhost:3123/users
```

**Expected Response:**
```json
{
  "message": "users found",
  "data": [
    {
      "username": "testuser",
      "email": "test@example.com"
    }
  ],
  "trace_id": "1234567890abcdef1234567890abcdef",
  "span_id": "abcdef1234567890abcdef1234567890"
}
```

### 3. Test Error Handling

```bash
# Missing required fields
curl -X POST http://localhost:3123/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser"
  }'
```

**Expected Response:**
```json
{
  "message": "Username and Email are required",
  "trace_id": "1234567890abcdef1234567890abcdef",
  "span_id": "abcdef1234567890abcdef1234567890"
}
```

## Span Hierarchy in Sentry

When you view the traces in Sentry, you'll see a hierarchy like this:

### POST /users Trace
```
HTTP POST /users
├── request.parsing
├── request.validation
├── user.creation
    ├── validation
    └── db.create
        └── (otelgorm database span)
```

### GET /users Trace
```
HTTP GET /users
├── user.retrieval
├── db.query
    ├── db.find
    │   └── (otelgorm database span)
    └── data.processing
```

## Observing Children Spans

1. **In Sentry Dashboard**: Navigate to Performance → Traces
2. **Click on any trace**: You'll see the detailed span hierarchy
3. **Expand spans**: Each span shows its children with timing and attributes
4. **Attributes**: Each span includes relevant metadata like:
   - Database operations
   - Validation results
   - Error information
   - Performance metrics

## Key Benefits

- **Correlation**: Use trace_id from API responses to find the exact trace in Sentry
- **Detailed Analysis**: Each operation is broken down into child spans
- **Error Tracking**: Errors are captured at the appropriate span level
- **Performance Monitoring**: Measure performance of individual operations
- **Debugging**: Easily identify bottlenecks in the request flow 