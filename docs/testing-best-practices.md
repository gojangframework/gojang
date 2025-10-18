# Testing Best Practices

This comprehensive guide covers testing strategies, patterns, and best practices for Gojang applications.

## Overview

Gojang supports multiple levels of testing:
- 🧪 **Unit Tests** - Test individual functions and methods
- 🔗 **Integration Tests** - Test components working together
- 🌐 **Handler Tests** - Test HTTP handlers and routes
- 🗄️ **Database Tests** - Test database operations
- 🎭 **End-to-End Tests** - Test complete user flows

**Benefits of testing:**
- ✅ Catch bugs early
- ✅ Safe refactoring
- ✅ Living documentation
- ✅ Confidence in deployments
- ✅ Better code design

---

## Testing Structure

### Project Layout

```
gojang/
├── config/
│   ├── config.go
│   └── config_test.go
├── http/
│   ├── handlers/
│   │   ├── auth.go
│   │   └── auth_test.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── auth_test.go
│   └── security/
│       ├── password.go
│       └── password_test.go
└── models/
    └── schema/
```

**Naming convention:**
- Test files: `*_test.go`
- Test functions: `TestFunctionName(t *testing.T)`
- Benchmark functions: `BenchmarkFunctionName(b *testing.B)`

---

## Unit Testing

### Basic Unit Test

```go
// gojang/http/security/password_test.go
package security

import (
    "testing"
)

func TestHashPassword(t *testing.T) {
    password := "testpassword123"
    
    hash, err := HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword failed: %v", err)
    }
    
    if hash == "" {
        t.Error("Expected non-empty hash")
    }
    
    if hash == password {
        t.Error("Hash should not equal plaintext password")
    }
}

func TestCheckPassword_ValidPassword(t *testing.T) {
    password := "testpassword123"
    
    hash, err := HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword failed: %v", err)
    }
    
    match, err := CheckPassword(hash, password)
    if err != nil {
        t.Fatalf("CheckPassword failed: %v", err)
    }
    
    if !match {
        t.Error("Expected password to match hash")
    }
}

func TestCheckPassword_InvalidPassword(t *testing.T) {
    password := "testpassword123"
    wrongPassword := "wrongpassword"
    
    hash, err := HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword failed: %v", err)
    }
    
    match, err := CheckPassword(hash, wrongPassword)
    if err != nil {
        t.Fatalf("CheckPassword failed: %v", err)
    }
    
    if match {
        t.Error("Expected password not to match hash")
    }
}
```

**Best practices:**
- ✅ Use descriptive test names
- ✅ Test both success and failure cases
- ✅ Use `t.Fatalf()` for setup failures
- ✅ Use `t.Error()` for test assertions
- ✅ Keep tests independent

### Table-Driven Tests

Perfect for testing multiple scenarios:

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        want    bool
        wantErr bool
    }{
        {
            name:    "valid email",
            email:   "test@example.com",
            want:    true,
            wantErr: false,
        },
        {
            name:    "invalid - no @",
            email:   "testexample.com",
            want:    false,
            wantErr: false,
        },
        {
            name:    "invalid - no domain",
            email:   "test@",
            want:    false,
            wantErr: false,
        },
        {
            name:    "empty string",
            email:   "",
            want:    false,
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ValidateEmail(tt.email)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if got != tt.want {
                t.Errorf("ValidateEmail() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Benefits:**
- ✅ Easy to add new test cases
- ✅ Clear test coverage
- ✅ Named subtests for better output
- ✅ DRY (Don't Repeat Yourself)

---

## Handler Testing

### Basic Handler Test

```go
// gojang/http/handlers/pages_test.go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gojangframework/gojang/gojang/views/renderers"
)

func TestPageHandler_About(t *testing.T) {
    // Setup
    renderer := renderers.NewRenderer("../../views/templates")
    handler := &PageHandler{
        Renderer: renderer,
    }
    
    // Create request
    req := httptest.NewRequest("GET", "/about", nil)
    rec := httptest.NewRecorder()
    
    // Execute
    handler.About(rec, req)
    
    // Assert
    if rec.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", rec.Code)
    }
    
    body := rec.Body.String()
    if !contains(body, "About Us") {
        t.Error("Expected 'About Us' in response body")
    }
}

func contains(s, substr string) bool {
    return len(s) > 0 && len(substr) > 0 && 
           len(s) >= len(substr) && 
           indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return i
        }
    }
    return -1
}
```

### Testing with Dependencies

Use interfaces and test doubles:

```go
// Define interface
type UserStore interface {
    GetUser(ctx context.Context, id int) (*User, error)
}

// Handler uses interface
type Handler struct {
    Users UserStore
}

// Mock implementation for testing
type MockUserStore struct {
    GetUserFunc func(ctx context.Context, id int) (*User, error)
}

func (m *MockUserStore) GetUser(ctx context.Context, id int) (*User, error) {
    if m.GetUserFunc != nil {
        return m.GetUserFunc(ctx, id)
    }
    return nil, errors.New("not implemented")
}

// Test
func TestHandler_GetUser(t *testing.T) {
    // Setup mock
    mockStore := &MockUserStore{
        GetUserFunc: func(ctx context.Context, id int) (*User, error) {
            return &User{
                ID:    id,
                Email: "test@example.com",
            }, nil
        },
    }
    
    handler := &Handler{Users: mockStore}
    
    // Test
    req := httptest.NewRequest("GET", "/users/1", nil)
    rec := httptest.NewRecorder()
    
    handler.GetUser(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", rec.Code)
    }
}
```

### Testing POST Requests

```go
func TestAuthHandler_LoginPOST(t *testing.T) {
    // Setup handler (with dependencies)
    handler := setupAuthHandler(t)
    
    // Create user for testing
    testUser := createTestUser(t, handler.Client)
    
    // Create form data
    form := url.Values{}
    form.Set("email", testUser.Email)
    form.Set("password", "password123")
    
    // Create request
    req := httptest.NewRequest("POST", "/login", 
        strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    rec := httptest.NewRecorder()
    
    // Execute
    handler.LoginPOST(rec, req)
    
    // Assert redirect
    if rec.Code != http.StatusSeeOther {
        t.Errorf("Expected redirect (303), got %d", rec.Code)
    }
    
    location := rec.Header().Get("Location")
    if location != "/dashboard" {
        t.Errorf("Expected redirect to /dashboard, got %s", location)
    }
}
```

---

## Database Testing

### Setup Test Database

```go
// gojang/models/testing.go
package models

import (
    "context"
    "testing"
    
    "entgo.io/ent/dialect"
    _ "github.com/mattn/go-sqlite3"
)

// NewTestClient creates a test database client
func NewTestClient(t *testing.T) *Client {
    t.Helper()
    
    // Use in-memory SQLite for tests
    client, err := NewClient(Options{
        Driver: dialect.SQLite,
        DSN:    "file:ent?mode=memory&cache=shared&_fk=1",
    })
    if err != nil {
        t.Fatalf("Failed to create test client: %v", err)
    }
    
    // Run migrations
    ctx := context.Background()
    if err := client.Schema.Create(ctx); err != nil {
        t.Fatalf("Failed to create schema: %v", err)
    }
    
    // Cleanup after test
    t.Cleanup(func() {
        client.Close()
    })
    
    return client
}

// CreateTestUser creates a user for testing
func CreateTestUser(t *testing.T, client *Client) *User {
    t.Helper()
    
    hash, err := security.HashPassword("password123")
    if err != nil {
        t.Fatalf("Failed to hash password: %v", err)
    }
    
    user, err := client.User.Create().
        SetEmail("test@example.com").
        SetPasswordHash(hash).
        Save(context.Background())
    
    if err != nil {
        t.Fatalf("Failed to create test user: %v", err)
    }
    
    return user
}
```

### Database Operation Tests

```go
func TestUserCRUD(t *testing.T) {
    client := NewTestClient(t)
    ctx := context.Background()
    
    // Create
    user, err := client.User.Create().
        SetEmail("test@example.com").
        SetPasswordHash("hash").
        Save(ctx)
    
    if err != nil {
        t.Fatalf("Failed to create user: %v", err)
    }
    
    if user.ID == 0 {
        t.Error("Expected user ID to be set")
    }
    
    // Read
    found, err := client.User.Get(ctx, user.ID)
    if err != nil {
        t.Fatalf("Failed to get user: %v", err)
    }
    
    if found.Email != user.Email {
        t.Errorf("Expected email %s, got %s", user.Email, found.Email)
    }
    
    // Update
    updated, err := client.User.UpdateOne(user).
        SetIsStaff(true).
        Save(ctx)
    
    if err != nil {
        t.Fatalf("Failed to update user: %v", err)
    }
    
    if !updated.IsStaff {
        t.Error("Expected IsStaff to be true")
    }
    
    // Delete
    err = client.User.DeleteOne(user).Exec(ctx)
    if err != nil {
        t.Fatalf("Failed to delete user: %v", err)
    }
    
    // Verify deletion
    _, err = client.User.Get(ctx, user.ID)
    if err == nil {
        t.Error("Expected error when getting deleted user")
    }
}
```

### Testing Relationships

```go
func TestPostAuthorRelationship(t *testing.T) {
    client := NewTestClient(t)
    ctx := context.Background()
    
    // Create user
    user := CreateTestUser(t, client)
    
    // Create post with author
    post, err := client.Post.Create().
        SetSubject("Test Post").
        SetBody("Test content").
        SetAuthor(user).
        Save(ctx)
    
    if err != nil {
        t.Fatalf("Failed to create post: %v", err)
    }
    
    // Load post with author
    found, err := client.Post.Query().
        Where(post.IDEQ(post.ID)).
        WithAuthor().
        Only(ctx)
    
    if err != nil {
        t.Fatalf("Failed to load post: %v", err)
    }
    
    // Check relationship
    if found.Edges.Author == nil {
        t.Fatal("Expected author to be loaded")
    }
    
    if found.Edges.Author.ID != user.ID {
        t.Errorf("Expected author ID %d, got %d", 
            user.ID, found.Edges.Author.ID)
    }
}
```

---

## Middleware Testing

### Testing Authentication Middleware

```go
func TestRequireAuth_NotLoggedIn(t *testing.T) {
    // Setup
    sessionManager := scs.New()
    client := NewTestClient(t)
    
    // Create handler that requires auth
    protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Protected content"))
    })
    
    handler := middleware.RequireAuth(sessionManager, client)(protected)
    
    // Create request without session
    req := httptest.NewRequest("GET", "/protected", nil)
    rec := httptest.NewRecorder()
    
    // Execute
    handler.ServeHTTP(rec, req)
    
    // Assert redirect to login
    if rec.Code != http.StatusSeeOther {
        t.Errorf("Expected redirect (303), got %d", rec.Code)
    }
    
    location := rec.Header().Get("Location")
    if !contains(location, "/login") {
        t.Errorf("Expected redirect to login, got %s", location)
    }
}

func TestRequireAuth_LoggedIn(t *testing.T) {
    // Setup
    sessionManager := scs.New()
    client := NewTestClient(t)
    ctx := context.Background()
    
    // Create test user
    user := CreateTestUser(t, client)
    
    // Create handler
    protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check user in context
        u := middleware.GetUser(r.Context())
        if u == nil {
            t.Error("Expected user in context")
        }
        w.WriteHeader(http.StatusOK)
    })
    
    handler := middleware.RequireAuth(sessionManager, client)(protected)
    
    // Create request with session
    req := httptest.NewRequest("GET", "/protected", nil)
    
    // Add session data
    ctx = sessionManager.Load(ctx)
    sessionManager.Put(ctx, "user_id", user.ID)
    req = req.WithContext(ctx)
    
    rec := httptest.NewRecorder()
    
    // Execute
    handler.ServeHTTP(rec, req)
    
    // Assert success
    if rec.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", rec.Code)
    }
}
```

---

## Integration Testing

### Testing Complete Flows

```go
func TestUserRegistrationFlow(t *testing.T) {
    // Setup
    client := NewTestClient(t)
    sessionManager := scs.New()
    renderer := renderers.NewRenderer("../../views/templates")
    
    handler := NewAuthHandler(client, sessionManager, renderer)
    
    // 1. GET registration form
    req := httptest.NewRequest("GET", "/register", nil)
    rec := httptest.NewRecorder()
    
    handler.RegisterGET(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Fatalf("Expected 200 for GET, got %d", rec.Code)
    }
    
    // 2. POST registration data
    form := url.Values{}
    form.Set("email", "newuser@example.com")
    form.Set("password", "password123")
    form.Set("password_confirm", "password123")
    
    req = httptest.NewRequest("POST", "/register", 
        strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    rec = httptest.NewRecorder()
    
    handler.RegisterPOST(rec, req)
    
    // 3. Verify redirect
    if rec.Code != http.StatusSeeOther {
        t.Fatalf("Expected redirect, got %d", rec.Code)
    }
    
    // 4. Verify user created
    ctx := context.Background()
    users, err := client.User.Query().
        Where(user.EmailEQ("newuser@example.com")).
        All(ctx)
    
    if err != nil {
        t.Fatalf("Failed to query users: %v", err)
    }
    
    if len(users) != 1 {
        t.Fatalf("Expected 1 user, got %d", len(users))
    }
    
    // 5. Verify password hashed
    if users[0].PasswordHash == "password123" {
        t.Error("Password should be hashed")
    }
}
```

---

## Test Fixtures and Helpers

### Reusable Test Setup

```go
// gojang/http/handlers/testing.go
package handlers

import (
    "testing"
    
    "github.com/gojangframework/gojang/gojang/models"
    "github.com/gojangframework/gojang/gojang/views/renderers"
    "github.com/alexedwards/scs/v2"
)

// TestSetup holds common test dependencies
type TestSetup struct {
    Client         *models.Client
    SessionManager *scs.SessionManager
    Renderer       *renderers.Renderer
}

// NewTestSetup creates a new test setup
func NewTestSetup(t *testing.T) *TestSetup {
    t.Helper()
    
    return &TestSetup{
        Client:         models.NewTestClient(t),
        SessionManager: scs.New(),
        Renderer:       renderers.NewRenderer("../../views/templates"),
    }
}

// CreateAuthHandler creates an auth handler for testing
func (s *TestSetup) CreateAuthHandler() *AuthHandler {
    return NewAuthHandler(s.Client, s.SessionManager, s.Renderer)
}

// CreateTestUser creates a test user
func (s *TestSetup) CreateTestUser(t *testing.T, email string) *models.User {
    t.Helper()
    
    hash, _ := security.HashPassword("password123")
    user, err := s.Client.User.Create().
        SetEmail(email).
        SetPasswordHash(hash).
        Save(context.Background())
    
    if err != nil {
        t.Fatalf("Failed to create test user: %v", err)
    }
    
    return user
}

// Usage in tests
func TestSomething(t *testing.T) {
    setup := NewTestSetup(t)
    handler := setup.CreateAuthHandler()
    user := setup.CreateTestUser(t, "test@example.com")
    
    // Your test code...
}
```

### Test Data Builders

```go
// Builder pattern for test data
type UserBuilder struct {
    email      string
    password   string
    isStaff    bool
    isActive   bool
}

func NewUserBuilder() *UserBuilder {
    return &UserBuilder{
        email:    "test@example.com",
        password: "password123",
        isStaff:  false,
        isActive: true,
    }
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
    b.email = email
    return b
}

func (b *UserBuilder) WithStaff(isStaff bool) *UserBuilder {
    b.isStaff = isStaff
    return b
}

func (b *UserBuilder) WithActive(isActive bool) *UserBuilder {
    b.isActive = isActive
    return b
}

func (b *UserBuilder) Build(t *testing.T, client *models.Client) *models.User {
    t.Helper()
    
    hash, _ := security.HashPassword(b.password)
    
    user, err := client.User.Create().
        SetEmail(b.email).
        SetPasswordHash(hash).
        SetIsStaff(b.isStaff).
        SetIsActive(b.isActive).
        Save(context.Background())
    
    if err != nil {
        t.Fatalf("Failed to create user: %v", err)
    }
    
    return user
}

// Usage
func TestAdminAccess(t *testing.T) {
    client := NewTestClient(t)
    
    // Create staff user
    adminUser := NewUserBuilder().
        WithEmail("admin@example.com").
        WithStaff(true).
        Build(t, client)
    
    // Create regular user
    regularUser := NewUserBuilder().
        WithEmail("user@example.com").
        Build(t, client)
    
    // Test...
}
```

---

## Benchmarking

### Basic Benchmark

```go
func BenchmarkHashPassword(b *testing.B) {
    password := "testpassword123"
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := security.HashPassword(password)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkCheckPassword(b *testing.B) {
    password := "testpassword123"
    hash, _ := security.HashPassword(password)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := security.CheckPassword(hash, password)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

**Run benchmarks:**

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkHashPassword ./gojang/http/security

# With memory allocation stats
go test -bench=. -benchmem ./...

# Compare benchmarks
go test -bench=. -count=5 | tee old.txt
# Make changes
go test -bench=. -count=5 | tee new.txt
benchstat old.txt new.txt
```

---

## Code Coverage

### Measure Coverage

```bash
# Run tests with coverage
go test -cover ./...

# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage by function
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
start coverage.html  # Windows
```

### Coverage by Package

```bash
# Create coverage for each package
for pkg in $(go list ./...); do
    go test -coverprofile=coverage.out $pkg
    go tool cover -func=coverage.out
done
```

### Enforce Coverage Threshold

```bash
# Fail if coverage below 80%
go test -cover -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' | \
    awk '{if ($1 < 80) exit 1}'
```

---

## Testing Best Practices

### ✅ DO

1. **Write tests first** (TDD)
   ```go
   // Write test
   func TestNewFeature(t *testing.T) { }
   // Then implement feature
   ```

2. **Keep tests simple** and readable
   ```go
   // ✅ Good
   if got != want {
       t.Errorf("got %v, want %v", got, want)
   }
   
   // ❌ Bad - complex logic in tests
   if complexCalculation(got, want, otherStuff) { }
   ```

3. **Test one thing per test**
   ```go
   // ✅ Good - focused tests
   func TestCreateUser(t *testing.T) { }
   func TestCreateUserDuplicate(t *testing.T) { }
   
   // ❌ Bad - testing too much
   func TestUserOperations(t *testing.T) { }
   ```

4. **Use table-driven tests** for multiple scenarios
   ```go
   tests := []struct {
       name string
       input string
       want string
   }{ /* ... */ }
   ```

5. **Clean up after tests**
   ```go
   t.Cleanup(func() {
       client.Close()
   })
   ```

6. **Use test helpers**
   ```go
   func createTestUser(t *testing.T) *User {
       t.Helper()  // Mark as helper
       // ...
   }
   ```

7. **Test edge cases**
   ```go
   // Test empty, nil, zero values
   TestWithEmptyString(t *testing.T)
   TestWithNilValue(t *testing.T)
   TestWithZeroValue(t *testing.T)
   ```

8. **Mock external dependencies**
   ```go
   type MockEmailSender struct { }
   func (m *MockEmailSender) Send(to, body string) error {
       return nil
   }
   ```

9. **Use meaningful test names**
   ```go
   // ✅ Good
   func TestCreateUser_DuplicateEmail_ReturnsError(t *testing.T)
   
   // ❌ Bad
   func TestUser1(t *testing.T)
   ```

10. **Run tests in CI/CD**
    ```yaml
    # .github/workflows/test.yml
    - name: Run tests
      run: go test -v -cover ./...
    ```

### ❌ DON'T

1. **Don't skip error checking** in tests
   ```go
   // ❌ Bad
   user, _ := client.User.Create()...
   
   // ✅ Good
   user, err := client.User.Create()...
   if err != nil {
       t.Fatalf("Failed: %v", err)
   }
   ```

2. **Don't use real external services**
   ```go
   // ❌ Bad - calls real API
   response := http.Get("https://api.example.com")
   
   // ✅ Good - use mock/stub
   mockAPI := NewMockAPI()
   ```

3. **Don't depend on execution order**
   ```go
   // ❌ Bad - tests depend on each other
   func TestA(t *testing.T) { globalVar = 1 }
   func TestB(t *testing.T) { use globalVar }
   ```

4. **Don't use sleep** for timing
   ```go
   // ❌ Bad
   time.Sleep(1 * time.Second)
   
   // ✅ Good - use channels or proper sync
   select {
   case <-done:
   case <-time.After(timeout):
   }
   ```

5. **Don't test implementation details**
   ```go
   // ❌ Bad - testing internal method
   func TestPrivateMethod(t *testing.T)
   
   // ✅ Good - test public API
   func TestPublicBehavior(t *testing.T)
   ```

6. **Don't ignore test failures**
   ```go
   // ❌ Bad
   // fix this test
   // t.Skip("broken")
   ```

7. **Don't commit commented-out tests**
   ```go
   // ❌ Bad
   // func TestOldFeature(t *testing.T) { }
   ```

---

## Testing Patterns

### Testing Error Handling

```go
func TestDivide_ByZero(t *testing.T) {
    _, err := Divide(10, 0)
    
    if err == nil {
        t.Error("Expected error for division by zero")
    }
    
    expectedErr := "division by zero"
    if err.Error() != expectedErr {
        t.Errorf("Expected error '%s', got '%s'", 
            expectedErr, err.Error())
    }
}
```

### Testing Panics

```go
func TestPanicHandling(t *testing.T) {
    defer func() {
        if r := recover(); r == nil {
            t.Error("Expected panic")
        }
    }()
    
    FunctionThatPanics()
}
```

### Testing Timeouts

```go
func TestWithTimeout(t *testing.T) {
    done := make(chan bool)
    
    go func() {
        SlowFunction()
        done <- true
    }()
    
    select {
    case <-done:
        // Success
    case <-time.After(5 * time.Second):
        t.Error("Function took too long")
    }
}
```

---

## Continuous Integration

### GitHub Actions

Create `.github/workflows/test.yml`:

```yaml
name: Test

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run tests
      run: go test -v -race -coverprofile=coverage.out ./...
    
    - name: Check coverage
      run: |
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "Coverage: $coverage%"
        if (( $(echo "$coverage < 70" | bc -l) )); then
          echo "Coverage below 70%"
          exit 1
        fi
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

---

## Tools and Libraries

### Testing Libraries

```go
// Standard library
import "testing"

// Test assertions (optional)
import "github.com/stretchr/testify/assert"

assert.Equal(t, expected, actual)
assert.NotNil(t, object)
assert.NoError(t, err)

// Mocking (optional)
import "github.com/stretchr/testify/mock"

type MockEmailSender struct {
    mock.Mock
}

func (m *MockEmailSender) Send(to, body string) error {
    args := m.Called(to, body)
    return args.Error(0)
}
```

### Useful Commands

```bash
# Run all tests
go test ./...

# Verbose output
go test -v ./...

# Run specific test
go test -run TestFunctionName ./...

# Run specific package
go test ./gojang/http/handlers

# Race detection
go test -race ./...

# Short mode (skip long tests)
go test -short ./...

# Parallel execution
go test -parallel 4 ./...

# Timeout
go test -timeout 30s ./...

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. ./...
go test -bench=. -benchmem ./...
```

---

## Next Steps

- ✅ **Write:** Add tests to your existing code
- ✅ **Measure:** Check test coverage
- ✅ **Automate:** Set up CI/CD pipeline
- ✅ **Practice:** Use TDD for new features
- ✅ **Read:** [Authentication Guide](./authentication-authorization.md)
- ✅ **Read:** [Deployment Guide](./deployment-guide.md)

---

## Quick Reference

### Test Structure

```go
func TestName(t *testing.T) {
    // Arrange (setup)
    
    // Act (execute)
    
    // Assert (verify)
}
```

### Common Assertions

```go
// Equality
if got != want {
    t.Errorf("got %v, want %v", got, want)
}

// Error checking
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}

if err == nil {
    t.Error("expected error")
}

// Nil checking
if value == nil {
    t.Error("expected non-nil value")
}

// Boolean
if !condition {
    t.Error("expected true")
}
```

---

Happy testing! 🧪
