-- Fix admin password hash to use Go-compatible bcrypt format
-- This hash is for password: admin123
-- Generated using golang.org/x/crypto/bcrypt

UPDATE users 
SET password = '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
WHERE username = 'admin';

