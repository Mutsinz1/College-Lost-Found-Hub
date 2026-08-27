-- Development seed data.
--
-- NOT run by `go run cmd/migrate/main.go`; the runner only reads .sql files
-- directly in migrations/ and skips subdirectories. Apply it explicitly with:
--
--     go run cmd/migrate/main.go -seed
--
-- This data is for local development and demos only. It must never be loaded
-- into a production database: it creates a user with the admin role and posts
-- with hardcoded, publicly-known edit tokens.

-- Insert sample buildings for testing
INSERT INTO buildings (name, description, location)
SELECT * FROM (VALUES
('Science Building', 'Main science building with labs and classrooms', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326)),
('Library', 'Central library with study spaces', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326)),
('Student Center', 'Student union building with dining and activities', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326)),
('Gym', 'Athletic center with gym and pool', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326))
) AS v(name, description, location)
WHERE NOT EXISTS (SELECT 1 FROM buildings);

-- Insert sample lost & found areas
INSERT INTO lost_found_areas (building_id, name, location_description, contact_person, hours_of_operation, pickup_instructions)
SELECT * FROM (VALUES
((SELECT id FROM buildings WHERE name = 'Science Building' LIMIT 1), 'Main Office', 'First floor, room 101', 'Dr. Smith', 'Mon-Fri 9AM-5PM', 'Check in at reception desk'),
((SELECT id FROM buildings WHERE name = 'Library' LIMIT 1), 'Circulation Desk', 'Main floor, near entrance', 'Ms. Johnson', 'Mon-Sun 8AM-10PM', 'Ask at circulation desk'),
((SELECT id FROM buildings WHERE name = 'Student Center' LIMIT 1), 'Information Desk', 'Main lobby', 'Student Assistant', 'Mon-Fri 8AM-8PM', 'Check information desk'),
((SELECT id FROM buildings WHERE name = 'Gym' LIMIT 1), 'Equipment Room', 'Lower level', 'Coach Wilson', 'Mon-Fri 6AM-10PM', 'Ask at equipment room')
) AS v(building_id, name, location_description, contact_person, hours_of_operation, pickup_instructions)
WHERE NOT EXISTS (SELECT 1 FROM lost_found_areas);

-- Insert sample admin user
INSERT INTO users (sso_id, email, name, role) VALUES
('admin_sso_123', 'admin@college.edu', 'System Administrator', 'admin')
ON CONFLICT (email) DO NOTHING;

-- Insert some sample data for testing (found items)
INSERT INTO posts (type, category, title, description, location, lost_found_area_id, posted_by, is_lost_item, edit_token, image_urls)
SELECT * FROM (VALUES
('found'::post_type, 'item'::post_category, 'Lost iPhone 14', 'Found iPhone 14 with blue case near Science Building entrance', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326), (SELECT id FROM lost_found_areas WHERE name = 'Main Office' LIMIT 1), (SELECT id FROM users WHERE email = 'admin@college.edu' LIMIT 1), false, 'edit_token_123', ARRAY['/uploads/sample_phone1.jpg']),
('found', 'document', 'Found Student ID Card', 'Found student ID card near Library entrance', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326), (SELECT id FROM lost_found_areas WHERE name = 'Circulation Desk' LIMIT 1), (SELECT id FROM users WHERE email = 'admin@college.edu' LIMIT 1), false, 'edit_token_456', ARRAY['/uploads/sample_id1.jpg']),
('found', 'item', 'Found Laptop Charger', 'Found laptop charger in Student Center cafeteria', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326), (SELECT id FROM lost_found_areas WHERE name = 'Information Desk' LIMIT 1), (SELECT id FROM users WHERE email = 'admin@college.edu' LIMIT 1), false, 'edit_token_789', ARRAY['/uploads/sample_charger1.jpg'])
) AS v(type, category, title, description, location, lost_found_area_id, posted_by, is_lost_item, edit_token, image_urls)
WHERE NOT EXISTS (SELECT 1 FROM posts); 