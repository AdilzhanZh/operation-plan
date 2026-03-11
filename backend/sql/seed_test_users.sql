INSERT INTO users (
  first_name,
  last_name,
  middle_name,
  full_name,
  username,
  password_plain,
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
    'AdminTest1',
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
    'ViewerTest1',
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
    'ProrectorA1',
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
    'ProrectorB1',
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
  password_plain = EXCLUDED.password_plain,
  role = EXCLUDED.role,
  updated_at = NOW();
