# CLAUDE.md - Developer Instructions for AI Assistants

This file contains **critical** instructions for AI assistants (like Claude) when working on this codebase.

---

## ⚠️ MOST IMPORTANT RULE - UPDATE MECHANISM

**THE UPDATE SCRIPT MUST ALWAYS WORK!**

When making ANY changes to ServerOS, remember:

### ✅ The Update Workflow MUST Always Function

Users can update their installation by simply re-running:

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

Or:

```bash
sudo /opt/serveros/update.sh
```

### What This Means:

1. **NEVER break the update mechanism**
   - The `install.sh` script detects existing installations
   - The `update.sh` script updates without user prompts
   - Both MUST preserve user data and configuration

2. **What gets UPDATED:**
   - ✅ All PHP files in `/opt/serveros/public/`
   - ✅ All PHP files in `/opt/serveros/src/`
   - ✅ All template files in `/opt/serveros/views/`
   - ✅ All tools in `/opt/serveros/tools/`
   - ✅ CSS and JavaScript files
   - ✅ `init.sql` schema file

3. **What gets PRESERVED:**
   - ✅ `/opt/serveros/config/config.php` (database credentials!)
   - ✅ Database and all tables
   - ✅ All users in the database
   - ✅ Logs in `/opt/serveros/logs/`
   - ✅ Data in `/opt/serveros/storage/`
   - ✅ Apache configuration

4. **Testing Updates:**
   When you make changes, ALWAYS test:
   ```bash
   # On test system with existing installation
   git clone https://github.com/chukfinley/tso.git
   cd tso
   sudo ./install.sh
   # Should detect existing installation and offer update
   ```

---

## Project Architecture

### Directory Structure

```
/opt/serveros/              # Production installation directory
├── config/
│   └── config.php          # Database credentials - NEVER OVERWRITE ON UPDATE
├── public/                 # Web root - safe to update
│   ├── *.php              # Page files - safe to update
│   ├── css/               # Stylesheets - safe to update
│   └── js/                # JavaScript - safe to update
├── src/                   # Backend classes - safe to update
│   ├── Database.php
│   ├── User.php
│   └── Auth.php
├── views/                 # Templates - safe to update
│   └── layout/
├── tools/                 # Admin tools - safe to update
│   ├── check-db.php
│   └── reset-admin.php
├── logs/                  # DO NOT DELETE ON UPDATE
├── storage/               # DO NOT DELETE ON UPDATE
└── init.sql               # Schema - safe to update (won't re-run)
```

### Key Files That Control Updates

1. **install.sh** - Main installer
   - Lines 412-462: `detect_existing_installation()` and `perform_update()`
   - CRITICAL: Must backup config before updating
   - CRITICAL: Must restore config after updating

2. **update.sh** - Dedicated update script
   - Designed for non-interactive updates
   - Used by automated deployments

3. **bootstrap.sh** - One-liner entry point
   - Clones repo and runs install.sh
   - Must always work for updates

---

## Development Workflow

### Making Changes to ServerOS

1. **For Application Code Changes** (PHP, CSS, JS):
   ```bash
   # Edit files as needed
   vim public/users.php
   vim public/css/style.css

   # Commit changes
   git add .
   git commit -m "Add feature X"
   git push

   # Users can update with:
   # curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
   ```

2. **For Database Schema Changes**:
   ```bash
   # Edit init.sql
   vim init.sql

   # Add migration logic to install.sh if needed
   # DO NOT just re-run init.sql on updates (will fail on existing tables)

   # Instead, add migration in install.sh:
   # php -r "
   #   require_once '/opt/serveros/config/config.php';
   #   \$pdo = new PDO(...);
   #   \$pdo->exec('ALTER TABLE users ADD COLUMN new_field VARCHAR(255)');
   # "
   ```

3. **For Configuration Changes**:
   ```bash
   # Edit config/config.php
   vim config/config.php

   # Add new config with DEFAULT values
   # DO NOT require users to manually update config

   # Example:
   define('NEW_FEATURE', getenv('NEW_FEATURE') ?: 'default_value');
   ```

---

## Critical Patterns

### ✅ DO: Make Changes Update-Safe

```php
// Good: Check if column exists before adding
$stmt = $pdo->query("SHOW COLUMNS FROM users LIKE 'new_column'");
if ($stmt->rowCount() === 0) {
    $pdo->exec("ALTER TABLE users ADD COLUMN new_column VARCHAR(255)");
}
```

### ❌ DON'T: Break Existing Installations

```php
// Bad: Will fail on update if column exists
$pdo->exec("ALTER TABLE users ADD COLUMN new_column VARCHAR(255)");
```

### ✅ DO: Preserve User Data

```bash
# Good: Backup before changing
cp ${INSTALL_DIR}/config/config.php /tmp/backup.php
# ... make changes ...
cp /tmp/backup.php ${INSTALL_DIR}/config/config.php
```

### ❌ DON'T: Overwrite User Configuration

```bash
# Bad: Will lose database credentials!
cp config/config.php ${INSTALL_DIR}/config/config.php
```

---

## Testing Checklist

Before committing changes, verify:

- [ ] Fresh install works: `sudo ./install.sh` on clean system
- [ ] Update works: `sudo ./install.sh` on existing installation
- [ ] Login still works after update
- [ ] Database credentials preserved
- [ ] User accounts preserved
- [ ] No data loss in logs/storage
- [ ] Apache restarts successfully
- [ ] Tools in `/opt/serveros/tools/` still work

---

## Common Tasks

### Adding a New Page

1. Create page file:
   ```bash
   vim public/newpage.php
   ```

2. Follow existing pattern:
   ```php
   <?php
   require_once __DIR__ . '/../config/config.php';
   require_once SRC_PATH . '/Database.php';
   require_once SRC_PATH . '/User.php';
   require_once SRC_PATH . '/Auth.php';

   $auth = new Auth();
   $auth->requireLogin();  // or requireAdmin()

   $pageTitle = 'New Page';
   ?>

   <?php include VIEWS_PATH . '/layout/header.php'; ?>
   <?php include VIEWS_PATH . '/layout/navbar.php'; ?>

   <div class="container">
       <!-- Your content -->
   </div>

   <?php include VIEWS_PATH . '/layout/footer.php'; ?>
   ```

3. Add to navigation (views/layout/navbar.php)

4. Test and commit

### Adding a New Backend Class

1. Create class:
   ```bash
   vim src/NewFeature.php
   ```

2. Follow existing pattern (see User.php, Auth.php)

3. Use dependency injection for Database

4. Add to pages that need it

### Adding Database Tables

1. Add to init.sql:
   ```sql
   CREATE TABLE IF NOT EXISTS new_table (
       id INT AUTO_INCREMENT PRIMARY KEY,
       ...
   ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
   ```

2. Create migration logic for updates:
   ```bash
   # In install.sh, add to perform_update():
   php -r "
       require_once '${INSTALL_DIR}/config/config.php';
       \$pdo = new PDO(...);
       \$pdo->exec('CREATE TABLE IF NOT EXISTS new_table (...)');
   "
   ```

---

## Security Considerations

### Always Apply:

1. **Password Hashing**:
   ```php
   $hash = password_hash($password, PASSWORD_BCRYPT);
   ```

2. **Prepared Statements**:
   ```php
   $stmt = $pdo->prepare("SELECT * FROM users WHERE id = ?");
   $stmt->execute([$userId]);
   ```

3. **Output Escaping**:
   ```php
   echo htmlspecialchars($userInput);
   ```

4. **Authentication Checks**:
   ```php
   $auth->requireLogin();    // For any logged-in user
   $auth->requireAdmin();    // For admin-only pages
   ```

---

## Documentation Updates

When adding features, update:

1. **README.md** - If major feature
2. **INSTALL.md** - If installation changes
3. **TROUBLESHOOTING.md** - If common issues expected
4. **CONTRIBUTING.md** - If dev workflow changes
5. **This file (CLAUDE.md)** - If architecture changes

---

## Emergency Procedures

### If Update Breaks Installation:

1. Users can restore from backup:
   ```bash
   sudo mysql servermanager < backup.sql
   sudo cp -r /tmp/serveros-backup /opt/serveros
   ```

2. Users can reinstall:
   ```bash
   sudo /opt/serveros/uninstall.sh
   curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
   ```

3. Fix the bug in GitHub and users re-update

### If Database Migration Fails:

1. Provide manual fix in TROUBLESHOOTING.md
2. Create tool in `/tools/` to fix it
3. Update install.sh to handle it

---

## Version Control

### Commit Message Format:

```
[type] Brief description

Longer description if needed

- Change 1
- Change 2

Update mechanism: [verified/not-applicable]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### Example:

```
feat: Add Docker management page

- Create public/docker.php
- Add Docker class in src/Docker.php
- Update navigation with Docker link

Update mechanism: verified
- Tested fresh install
- Tested update from previous version
- Config preserved, no data loss
```

---

## Questions for AI Assistants

When working on this project, always consider:

1. ✅ Will this change work when user runs update.sh?
2. ✅ Does this preserve user configuration?
3. ✅ Does this preserve database data?
4. ✅ Can this be rolled back if it fails?
5. ✅ Is this documented for troubleshooting?

---

## Quick Reference

### Update Testing Commands:

```bash
# Test fresh install
sudo ./install.sh

# Test update (on system with existing installation)
sudo ./update.sh

# Test bootstrap one-liner
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash

# Check database
sudo /opt/serveros/tools/check-db.php

# Reset admin if needed
sudo /opt/serveros/tools/reset-admin.php
```

### File Safety Matrix:

| File/Directory | Safe to Overwrite on Update? |
|----------------|------------------------------|
| public/*.php | ✅ YES |
| src/*.php | ✅ YES |
| views/*.php | ✅ YES |
| tools/*.php | ✅ YES |
| config/config.php | ❌ NO - PRESERVE! |
| logs/ | ❌ NO - PRESERVE! |
| storage/ | ❌ NO - PRESERVE! |
| Database tables | ❌ NO - PRESERVE! |

---

## Final Reminder

> **Every change you make must support the update workflow.**
>
> Users trust that running the update won't break their system.
>
> Test updates before committing!

---

**This is a living document. Update it when architecture changes.**
