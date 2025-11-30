# AGENTS.md - ASAP Project Manager

## Build/Lint/Test Commands

### Build
- `go build .` - Build the binary
- `go build -o asap-pm .` - Build with custom output name

### Test
- `go test ./...` - Run all tests
- `go test -v ./...` - Run tests with verbose output
- `go test -run TestName ./...` - Run specific test

### Lint & Format
- `gofmt -d .` - Check formatting (dry run)
- `gofmt -w .` - Format code
- `go vet ./...` - Run static analysis
- `golangci-lint run` - Run comprehensive linting (if installed)

### Dependencies
- `go mod tidy` - Clean up dependencies
- `go mod download` - Download dependencies

## Code Style Guidelines

### Imports
- Group standard library imports first, then third-party, then local
- Use blank lines between import groups
- Remove unused imports

### Naming
- Use camelCase for variables/functions, PascalCase for exported
- Acronyms in names should be all caps (e.g., `JSONData`, not `JsonData`)
- Functions: descriptive names, verbs for actions

### Error Handling
- Always handle errors explicitly
- Use `fmt.Errorf` for wrapping errors with context
- Return errors early, avoid nested error handling

### Types & Structs
- Use meaningful struct field names
- Prefer struct embedding over inheritance
- Use pointer receivers for methods that modify state

### Formatting
- Use `gofmt` for consistent formatting
- Max line length: 120 characters
- Use meaningful whitespace

### Testing
- Write table-driven tests when possible
- Use descriptive test names: `TestFunctionName_Scenario_Result`
- Test both success and error cases

### Project Structure
- Keep main.go minimal, delegate to packages
- Use clear package names that reflect functionality
- Separate concerns: config, ui, core logic