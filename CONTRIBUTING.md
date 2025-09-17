# Contributing to srake

Thank you for your interest in contributing to srake! We welcome contributions from the community and are grateful for any help you can provide.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:

- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on what is best for the community
- Show empathy towards other community members

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

1. **Use a clear and descriptive title**
2. **Describe the exact steps to reproduce the problem**
3. **Provide specific examples**
4. **Describe the behavior you observed and expected**
5. **Include system information:**
   - OS and version
   - Go version (`go version`)
   - srake version (`srake --version`)
   - Database size and type of data

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

1. **Use a clear and descriptive title**
2. **Provide a detailed description of the suggested enhancement**
3. **Explain why this enhancement would be useful**
4. **List any alternative solutions you've considered**

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes:**
   - Write clear, commented code
   - Follow the existing code style
   - Update documentation as needed
3. **Write or update tests** for your changes
4. **Ensure all tests pass:**
   ```bash
   go test ./...
   ```
5. **Format your code:**
   ```bash
   go fmt ./...
   ```
6. **Create a Pull Request** with a clear title and description

## Development Setup

### Prerequisites

- Go 1.19 or later
- Git
- SQLite3
- Make (optional, for using Makefile)

### Getting Started

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/YOUR_USERNAME/srake.git
   cd srake
   ```

2. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/nishad/srake.git
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Build the project:
   ```bash
   go build -o srake ./cmd/srake
   ```

5. Run tests:
   ```bash
   go test ./...
   ```

### Development Workflow

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit:
   ```bash
   git add .
   git commit -m "Add: brief description of changes"
   ```

3. Keep your fork synchronized:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

4. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

5. Create a Pull Request on GitHub

## Code Style Guidelines

### Go Code

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused
- Handle errors explicitly
- Use table-driven tests where appropriate

### Commit Messages

Follow the conventional commits format:

```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `style:` Code style changes (formatting, etc.)
- `refactor:` Code refactoring
- `test:` Adding or updating tests
- `chore:` Maintenance tasks
- `perf:` Performance improvements

Example:
```
feat: add support for filtering by date range

- Add --start-date and --end-date flags
- Update query builder to handle date filters
- Add tests for date range validation
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/processor

# Run with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./internal/processor
```

### Writing Tests

- Write unit tests for new functionality
- Aim for >80% code coverage
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error conditions
- Include benchmarks for performance-critical code

Example test structure:
```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "test",
            want:  "expected",
        },
        {
            name:    "invalid input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Documentation

- Update README.md for user-facing changes
- Add godoc comments for exported functions
- Update command help text when adding flags
- Include examples in documentation
- Keep documentation concise and clear

## Performance Considerations

When contributing performance-related changes:

1. **Benchmark before and after** your changes
2. **Profile the code** to identify bottlenecks
3. **Consider memory allocation** and garbage collection
4. **Test with realistic data sizes**
5. **Document performance improvements**

## Security

- Never commit credentials or sensitive data
- Validate and sanitize all user inputs
- Use secure defaults
- Follow Go security best practices
- Report security vulnerabilities privately

## Project Structure

```
srake/
├── cmd/srake/          # Command-line interface
├── internal/           # Internal packages
│   ├── processor/      # Stream processing logic
│   ├── database/       # Database operations
│   ├── server/         # HTTP server
│   ├── config/         # Configuration
│   └── cli/           # CLI utilities
├── configs/           # Configuration files
├── scripts/           # Utility scripts
├── sra-schemas/       # XSD schemas
└── tests/            # Integration tests
```

## Getting Help

- Check the [documentation](https://github.com/nishad/srake/wiki)
- Search [existing issues](https://github.com/nishad/srake/issues)
- Join our [discussions](https://github.com/nishad/srake/discussions)
- Ask questions in issues with the `question` label

## Review Process

All submissions require review. We use GitHub pull requests for this purpose. The review process:

1. Automated checks run (tests, linting)
2. Code review by maintainers
3. Address feedback and iterate
4. Approval and merge

## Release Process

1. Update version numbers
2. Update CHANGELOG.md
3. Create release notes
4. Tag the release
5. Build and upload binaries

## Recognition

Contributors will be recognized in:
- The project's README
- Release notes
- GitHub contributors page

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Questions?

Feel free to open an issue with the `question` label or reach out to the maintainers.

Thank you for contributing to srake! 🎉