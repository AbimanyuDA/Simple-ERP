ALTER TABLE users DROP CONSTRAINT users_role_check;
UPDATE users SET role = 'OPERATOR' WHERE role = 'STAFF';
UPDATE users SET role = 'ADMIN' WHERE role IN ('SUPERVISOR', 'SUPER_ADMIN');
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('ADMIN', 'OPERATOR'));
