# Contributing to Gojang

First off, thank you for considering contributing to Gojang! It's people like you that make Gojang a great framework for building web applications.

## Code of Conduct

This project and everyone participating in it is governed by common sense and mutual respect. By participating, you are expected to uphold this standard. Please be kind and courteous in all interactions.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

**Great Bug Report Template:**

```markdown
**Description:**
A clear and concise description of the bug.

**To Reproduce:**
Steps to reproduce the behavior:
1. Go to '...'
2. Click on '...'
3. See error

**Expected Behavior:**
What you expected to happen.

**Actual Behavior:**
What actually happened.

**Environment:**
- OS: [e.g. Windows 11, macOS 13, Ubuntu 22.04]
- Go Version: [e.g. 1.21.5]
- Gojang Version: [e.g. commit hash or tag]

**Additional Context:**
Any other relevant information, screenshots, or error logs.
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the suggested enhancement
- **Explain why this enhancement would be useful** to most Gojang users
- **List some examples** of how it would be used
- **Mention if you'd be willing to implement it**

### Pull Requests

1. **Fork the repo** and create your branch from `main`
2. **Make your changes** with clear, concise commits
3. **Test your changes** thoroughly
4. **Update documentation** if needed
5. **Submit a pull request!**

#### Pull Request Process

1. Update the README.md or docs/ with details of changes if applicable
2. Update the CHANGELOG.md with a note describing your changes
3. The PR will be merged once you have sign-off from maintainers

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./gojang/http/handlers
```

### Code Style

- Follow standard Go formatting: `go fmt ./...`
- Run the linter: `go vet ./...`
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and single-purpose

**Good:**
```go
// CreateProduct creates a new product with the given details
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

**Less Good:**
```go
func (h *ProductHandler) CP(w http.ResponseWriter, r *http.Request) { // What's CP?
    // Implementation
}
```

### Naming Conventions

- **Files:** `lowercase_with_underscores.go` or `lowercasecamelcase.go`
- **Types:** `PascalCase` (exported) or `camelCase` (unexported)
- **Functions:** `PascalCase` (exported) or `camelCase` (unexported)
- **Variables:** `camelCase` or short names for short scopes (`i`, `err`)

## Performance Considerations

When contributing, keep performance in mind:

- ✅ Use eager loading for relationships: `.WithAuthor()`
- ✅ Add database indexes on frequently queried fields
- ✅ Paginate large result sets
- ✅ Avoid N+1 queries
- ✅ Cache static assets
- ❌ Don't load unbounded result sets
- ❌ Don't make unnecessary database calls in loops

## Security Considerations

Security is paramount. When contributing:

- ✅ Validate all user input
- ✅ Use parameterized queries (Ent handles this)
- ✅ Sanitize HTML output (templates handle this)
- ✅ Use CSRF protection (already implemented)
- ✅ Hash passwords properly (use `security.HashPassword`)
- ❌ Never log sensitive data (passwords, tokens, etc.)
- ❌ Never trust user input
- ❌ Never build raw SQL queries

## Release Process

For maintainers:

1. Update CHANGELOG.md
2. Update version in relevant files
3. Create git tag: `git tag -a v1.0.0 -m "Version 1.0.0"`
4. Push tag: `git push origin v1.0.0`
5. Create GitHub release with changelog

## Questions?

- 💬 Open a [Discussion](https://github.com/your-repo/discussions)
- 📧 Email: gojangframework@gmail.com
- 💬 Join our community chat *(coming soon)*

## Thank You!

Every contribution helps make Gojang better for everyone. We appreciate your time and effort! 🙏

---

**Happy Contributing! 🚀**
