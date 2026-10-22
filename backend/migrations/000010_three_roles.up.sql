-- Keep the earliest existing administrator as the account owner so upgrades retain
-- a way to invite supervisors. Other administrators become supervisors.
ALTER TABLE users DROP CONSTRAINT users_role_check;
UPDATE users SET role = 'STAFF' WHERE role = 'OPERATOR';
WITH first_admin AS (
    SELECT id FROM users WHERE role = 'ADMIN' ORDER BY (status = 'ACTIVE') DESC, created_at, id LIMIT 1
)
UPDATE users SET role = CASE WHEN id = (SELECT id FROM first_admin) THEN 'SUPER_ADMIN' ELSE 'SUPERVISOR' END
WHERE role = 'ADMIN';
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('STAFF', 'SUPERVISOR', 'SUPER_ADMIN'));
