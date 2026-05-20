```markdown
# daily-english Development Patterns

> Auto-generated skill from repository analysis

## Overview
This skill teaches you the development patterns and conventions used in the `daily-english` Go codebase. You'll learn about file naming, import/export styles, commit message conventions, and how to write and organize tests. This guide will help you contribute code that matches the project's standards and maintain consistency across the repository.

## Coding Conventions

### File Naming
- Use **camelCase** for file names.
  - Example: `dailyLesson.go`, `userProfile.go`

### Import Style
- Use **relative imports** within the project.
  - Example:
    ```go
    import "./utils"
    ```

### Export Style
- Use **named exports** for functions, types, and variables.
  - Example:
    ```go
    // In dailyLesson.go
    package dailylesson

    func GetLessonOfTheDay() string {
        // ...
    }
    ```

### Commit Messages
- Follow **conventional commit** patterns.
- Prefixes: `feat`, `ci`
- Example:
  ```
  feat: add daily lesson retrieval endpoint
  ci: update build pipeline for Go modules
  ```

## Workflows

_No automated workflows detected in the repository._

## Testing Patterns

- Test files follow the `*.test.*` pattern.
  - Example: `dailyLesson.test.go`
- The testing framework is **unknown**, but tests are likely written using Go's standard `testing` package.
- Example test file:
  ```go
  // dailyLesson.test.go
  package dailylesson

  import "testing"

  func TestGetLessonOfTheDay(t *testing.T) {
      result := GetLessonOfTheDay()
      if result == "" {
          t.Error("Expected a lesson, got empty string")
      }
  }
  ```

## Commands
| Command | Purpose |
|---------|---------|
| /test   | Run all test files matching `*.test.go` |
| /commit | Generate a conventional commit message |
```