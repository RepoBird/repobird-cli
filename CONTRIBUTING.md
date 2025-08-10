# Contributing to RepoBird CLI

Thank you for your interest in contributing to RepoBird CLI! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Accept feedback gracefully
- Prioritize the project's best interests

## How to Contribute

### Reporting Issues

Before creating an issue:
1. Check existing issues to avoid duplicates
2. Use the issue search to see if it's already reported
3. Check if the issue is fixed in the latest version

When creating an issue, include:
- Clear, descriptive title
- Steps to reproduce the problem
- Expected vs actual behavior
- System information (OS, Go version)
- Relevant logs or error messages
- Screenshots if applicable

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:
1. Check if the feature already exists
2. Search existing issues for similar suggestions
3. Provide a clear use case
4. Explain why this would be useful
5. Consider implementation complexity

### Pull Requests

#### Before You Start

1. **Discuss First**: For significant changes, open an issue first
2. **Check Issues**: Look for issues tagged `good first issue` or `help wanted`
3. **Read Documentation**: Familiarize yourself with [DEV.md](DEV.md) and [CLAUDE.md](CLAUDE.md)

#### Development Process

1. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/repobird-cli.git
   cd repobird-cli
   ```

2. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

3. **Make Changes**
   - Follow the code style guide
   - Write clear, self-documenting code
   - Add comments for complex logic
   - Update documentation as needed

4. **Write Tests**
   - Add unit tests for new functions
   - Ensure existing tests pass
   - Aim for >70% coverage on new code
   ```bash
   make test
   make coverage
   ```

5. **Lint and Format**
   ```bash
   make fmt
   make lint-fix
   make check
   ```

6. **Commit Changes**
   ```bash
   git add .
   git commit -m "type: brief description

   Detailed explanation of what changed and why.
   Reference any related issues: Fixes #123"
   ```

7. **Push and Create PR**
   ```bash
   git push origin your-branch-name
   ```

#### Commit Message Guidelines

Format: `<type>: <short summary>`

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code restructuring
- `test`: Test additions or fixes
- `chore`: Maintenance tasks
- `perf`: Performance improvements

Examples:
```
feat: add JSON schema validation for task files
fix: resolve timeout issue in status polling
docs: update installation instructions
refactor: simplify error handling in API client
```

#### Pull Request Guidelines

Your PR should:
1. Have a clear, descriptive title
2. Reference any related issues
3. Include a description of changes
4. List any breaking changes
5. Include test results
6. Have all CI checks passing

PR Description Template:
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings generated
- [ ] Tests added/updated
- [ ] All tests passing
```

### Code Style Guidelines

#### Go Code Standards

1. **Formatting**: Use `gofmt` (enforced by CI)
2. **Linting**: Pass `golangci-lint` checks
3. **Naming**: Follow Go naming conventions
   - Exported names start with capital letter
   - Use camelCase for variables
   - Use descriptive names

4. **Error Handling**:
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to parse task: %w", err)
   }
   
   // Avoid
   if err != nil {
       panic(err)
   }
   ```

5. **Comments**:
   ```go
   // Package api provides the HTTP client for RepoBird API communication.
   package api
   
   // CreateRun submits a new task to the RepoBird API.
   // It returns the created run or an error if the request fails.
   func CreateRun(task *Task) (*Run, error) {
       // Implementation
   }
   ```

#### Testing Standards

1. **Test Files**: Place alongside source files with `_test.go` suffix
2. **Test Names**: Use descriptive names starting with `Test`
3. **Table-Driven Tests**: Preferred for multiple test cases
4. **Mocking**: Mock external dependencies
5. **Coverage**: Aim for >70% on new code

Example:
```go
func TestCreateRun(t *testing.T) {
    tests := []struct {
        name    string
        task    *Task
        want    *Run
        wantErr bool
    }{
        {
            name: "valid task",
            task: &Task{Prompt: "test"},
            want: &Run{ID: "123"},
        },
        {
            name:    "nil task",
            task:    nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CreateRun(tt.task)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateRun() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("CreateRun() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Documentation

#### When to Update Documentation

Update docs when you:
- Add new features
- Change existing behavior
- Add new configuration options
- Modify command-line interface
- Fix documentation errors

#### Documentation Standards

1. **Clarity**: Write for users who may be new to the project
2. **Examples**: Include practical examples
3. **Accuracy**: Ensure docs match actual behavior
4. **Formatting**: Use proper Markdown formatting
5. **Links**: Verify all links work

### Review Process

#### What to Expect

1. **Initial Review**: Maintainers will review within 2-3 business days
2. **Feedback**: You may receive suggestions or change requests
3. **Discussion**: Feel free to discuss feedback if you disagree
4. **Approval**: Once approved, your PR will be merged
5. **Recognition**: Contributors are credited in release notes

#### Review Criteria

Your PR will be evaluated on:
- Code quality and style
- Test coverage
- Documentation updates
- Commit message quality
- CI/CD checks passing
- Alignment with project goals

### Getting Help

If you need help:
1. Check existing documentation
2. Search closed issues
3. Ask in PR comments
4. Open a discussion issue

### Development Setup Tips

#### Recommended Tools

```bash
# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/cosmtrek/air@latest  # For live reload
```

#### Useful Commands

```bash
# Quick development cycle
make clean build test

# Full CI simulation
make ci

# Debug mode
./build/repobird --debug status

# Live reload during development
air -c .air.toml
```

### Community

#### Communication Channels

- **Issues**: Bug reports and feature requests
- **Pull Requests**: Code contributions and discussions
- **Discussions**: General questions and ideas

#### Recognition

Contributors are recognized in:
- Release notes
- Contributors file
- Project documentation

### Release Cycle

- **Major releases**: Significant features or breaking changes
- **Minor releases**: New features, backward compatible
- **Patch releases**: Bug fixes and minor improvements

### Legal

By contributing, you agree that your contributions will be licensed under the same license as the project.

## Quick Checklist for Contributors

Before submitting your PR, ensure:

- [ ] Fork is up to date with main branch
- [ ] Branch follows naming convention
- [ ] Code compiles without warnings
- [ ] All tests pass locally
- [ ] New code has tests
- [ ] Documentation is updated
- [ ] Commit messages follow guidelines
- [ ] PR description is complete
- [ ] CI checks are passing

## Thank You!

Your contributions make RepoBird CLI better for everyone. We appreciate your time and effort in improving the project!