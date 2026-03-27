INSERT INTO users (
  first_name,
  last_name,
  middle_name,
  full_name,
  username,
  email,
  position,
  password_hash,
  role,
  created_at,
  updated_at
)
VALUES
  (
    'Aruzhan',
    'Sarsembayeva',
    'Nurlankyzy',
    'Sarsembayeva Aruzhan Nurlankyzy',
    'admin_test',
    'admin_test@example.com',
    'System Administrator',
    '$2a$10$IUyHZIlf7kFssvtOLgq2L.OTv8DAEIFXe.oylCZzaP8rXyW0QeQo2',
    'admin',
    NOW(),
    NOW()
  ),
  (
    'Timur',
    'Beketov',
    'Serikuly',
    'Beketov Timur Serikuly',
    'viewer_test',
    'viewer_test@example.com',
    'Methodist',
    '$2a$10$wL7JUR00I79w/5uBi0NtUOFiVnDMqr2iCh0eHBN3tkyDgodcL.wGW',
    'viewer',
    NOW(),
    NOW()
  ),
  (
    'Murat',
    'Aitimov',
    'Zholdasbekovich',
    'Aitimov Murat Zholdasbekovich',
    'prorector_aitimov',
    'aitimov@example.com',
    'Проректор',
    '$2a$10$7mB9lRVUo34mmKPvROQjuOOzRQkFfiyWmNLhLUnNIGqwEtsqHyEKu',
    'prorector',
    NOW(),
    NOW()
  ),
  (
    'Erkebulan',
    'Toktarov',
    'Toktarovich',
    'Toktarov Erkebulan Toktarovich',
    'prorector_toktarov',
    'toktarov@example.com',
    'Проректор',
    '$2a$10$muiuCGteoieW9j0.jLExXeVbgW6hiRPhdAe25xUZvujMWYcmeDj0S',
    'prorector',
    NOW(),
    NOW()
  )
ON CONFLICT (username)
DO UPDATE SET
  first_name = EXCLUDED.first_name,
  last_name = EXCLUDED.last_name,
  middle_name = EXCLUDED.middle_name,
  full_name = EXCLUDED.full_name,
  email = EXCLUDED.email,
  position = EXCLUDED.position,
  password_hash = EXCLUDED.password_hash,
  role = EXCLUDED.role,
  updated_at = NOW();
