# Contributing to ServerOS

Thank you for your interest in contributing to ServerOS! This document provides guidelines and instructions for contributing.

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue with:
- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior vs actual behavior
- Your environment (OS, PHP version, etc.)
- Screenshots if applicable

### Suggesting Features

Feature suggestions are welcome! Please:
- Check if the feature already exists or is planned
- Describe the feature in detail
- Explain the use case and benefits
- Consider implementation complexity

### Code Contributions

1. **Fork the repository**

2. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes:**
   - Follow the existing code style
   - Add comments where necessary
   - Test your changes thoroughly

4. **Commit your changes:**
   ```bash
   git add .
   git commit -m "Add feature: your feature description"
   ```

5. **Push to your fork:**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request:**
   - Provide a clear description of changes
   - Reference any related issues
   - Include screenshots for UI changes

## Development Setup

### Quick Development Environment

```bash
# Clone your fork
git clone https://github.com/yourusername/serveros.git
cd serveros

# Install dependencies (if not already installed)
sudo apt install php php-mysql mariadb-server

# Setup database
sudo mysql < init.sql

# Start development server
./quickstart.sh
```

Access at: `http://localhost:8000`

### Project Structure

```
serveros/
├── config/         # Configuration files
├── public/         # Web root (PHP pages, assets)
├── src/           # Core classes and logic
├── views/         # Template files
└── *.sh           # Installation scripts
```

## Coding Standards

### PHP Code Style

- Use 4 spaces for indentation (no tabs)
- Opening braces on same line for functions/classes
- Use meaningful variable and function names
- Add PHPDoc comments for functions
- Follow PSR-12 coding standard when possible

Example:
```php
/**
 * Get user by ID
 *
 * @param int $id User ID
 * @return array|false User data or false if not found
 */
public function getById($id) {
    $sql = "SELECT * FROM users WHERE id = ?";
    return $this->db->fetchOne($sql, [$id]);
}
```

### SQL Guidelines

- Always use prepared statements (no raw SQL with user input)
- Use meaningful table and column names
- Add indexes for frequently queried columns
- Include comments for complex queries

### Security Requirements

**Critical: All contributions must follow these security practices:**

- ✓ Use password_hash() for password storage
- ✓ Use prepared statements for all SQL queries
- ✓ Sanitize all user input with htmlspecialchars()
- ✓ Validate and sanitize file uploads
- ✓ Use CSRF tokens for forms (when implemented)
- ✓ Never store passwords in plain text
- ✓ Never expose sensitive data in error messages

### Frontend Guidelines

- Keep CSS organized and commented
- Use existing color variables from style.css
- Ensure responsive design (mobile-friendly)
- Test in multiple browsers
- Use semantic HTML5 elements

## Module Development

When adding new modules (e.g., Disk Management, Docker, etc.):

1. **Create the main page:**
   ```php
   public/yourmodule.php
   ```

2. **Create the backend class:**
   ```php
   src/YourModule.php
   ```

3. **Add database tables (if needed):**
   - Add to init.sql
   - Use proper foreign keys and indexes

4. **Update navigation:**
   - Already configured in views/layout/navbar.php
   - Ensure active state works correctly

5. **Add permissions:**
   - Use Auth::requireLogin() or Auth::requireAdmin()
   - Check user roles before sensitive operations

## Testing

Before submitting a pull request:

1. **Test basic functionality:**
   - Login/logout works
   - User management works (if affected)
   - No PHP errors in logs
   - Database queries execute correctly

2. **Test security:**
   - SQL injection attempts fail
   - XSS attempts are escaped
   - Authentication is enforced
   - Session handling works

3. **Test UI:**
   - Responsive on mobile/tablet/desktop
   - Works in Chrome, Firefox, Safari
   - No console errors
   - Forms validate properly

4. **Check logs:**
   ```bash
   tail -f /var/log/apache2/serveros_error.log
   ```

## Module Priority List

These modules are planned and contributions are welcome:

### High Priority:
1. **Disk Management** - Detect and manage disks, arrays, filesystems
2. **Network Shares** - SMB/NFS share management
3. **System Monitoring** - CPU, RAM, disk usage graphs

### Medium Priority:
4. **Docker Management** - Container list, start/stop, logs
5. **Backup System** - Scheduled backups, restore functionality
6. **Notifications** - Email/webhook alerts for system events

### Future:
7. **VM Management** - KVM/QEMU integration
8. **Plugin System** - Allow third-party plugins
9. **API** - RESTful API for automation
10. **Mobile App** - Native mobile monitoring app

## Documentation

When adding features, update:
- README.md - If it affects installation or major features
- INSTALL.md - If it changes installation steps
- Code comments - For complex logic
- Database schema - For new tables/columns

## Questions?

- Open an issue for questions
- Check existing issues and pull requests
- Review the code to understand patterns

## Code of Conduct

- Be respectful and constructive
- Help others learn and improve
- Focus on the code, not the person
- Accept feedback gracefully
- Keep discussions professional

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to ServerOS! 🚀
