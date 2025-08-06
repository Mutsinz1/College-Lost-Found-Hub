-- Enable PostGIS extension for spatial data
CREATE EXTENSION IF NOT EXISTS postgis;

-- Drop existing types if they exist (to avoid conflicts)
DROP TYPE IF EXISTS post_type CASCADE;
DROP TYPE IF EXISTS post_status CASCADE;
DROP TYPE IF EXISTS post_category CASCADE;

-- Create ENUM types
CREATE TYPE post_type AS ENUM ('lost', 'found');
CREATE TYPE post_status AS ENUM ('active', 'claimed', 'resolved');
CREATE TYPE post_category AS ENUM ('pet', 'document', 'item', 'other');

-- Buildings table (set up by admin)
CREATE TABLE buildings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL, -- 'Science Building', 'Library', 'Gym'
    description TEXT,
    location geography(Point, 4326),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Lost & Found Areas table (set up by admin)
CREATE TABLE lost_found_areas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    building_id UUID NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL, -- 'Main Office', 'Reception', 'Security Desk'
    location_description TEXT,
    contact_person VARCHAR(255),
    hours_of_operation TEXT,
    pickup_instructions TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Users table (populated via SSO)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sso_id VARCHAR(255) UNIQUE NOT NULL, -- SSO identifier
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user', -- 'user', 'admin'
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Posts table (enhanced for college system)
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type post_type NOT NULL,
    category post_category NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    location geography(Point, 4326) NOT NULL,
    lost_found_area_id UUID REFERENCES lost_found_areas(id),
    posted_by UUID NOT NULL REFERENCES users(id),
    claimed_by UUID REFERENCES users(id),
    claimed_at TIMESTAMP WITH TIME ZONE,
    pickup_scheduled_at TIMESTAMP WITH TIME ZONE,
    picked_up_at TIMESTAMP WITH TIME ZONE,
    is_lost_item BOOLEAN DEFAULT false, -- true for lost items, false for found items
    status post_status DEFAULT 'active',
    contact_email VARCHAR(255),
    poster_name VARCHAR(100),
    edit_token VARCHAR(64) UNIQUE NOT NULL,
    image_urls TEXT[],
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (now() + INTERVAL '30 days'),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_posts_location ON posts USING GIST(location);
CREATE INDEX IF NOT EXISTS idx_posts_type ON posts(type);
CREATE INDEX IF NOT EXISTS idx_posts_category ON posts(category);
CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_expires_at ON posts(expires_at);
CREATE INDEX IF NOT EXISTS idx_posts_lost_found_area ON posts(lost_found_area_id);
CREATE INDEX IF NOT EXISTS idx_posts_posted_by ON posts(posted_by);
CREATE INDEX IF NOT EXISTS idx_posts_claimed_by ON posts(claimed_by);
CREATE INDEX IF NOT EXISTS idx_posts_is_lost_item ON posts(is_lost_item);

-- Function to clean up expired posts
CREATE OR REPLACE FUNCTION cleanup_expired_posts()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM posts 
    WHERE expires_at < now() AND status != 'resolved';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Interactions table (claims, help offers, etc.)
CREATE TABLE interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    interaction_type VARCHAR(50) NOT NULL, -- 'claim', 'help', 'report'
    contact_email VARCHAR(255),
    contact_name VARCHAR(100),
    message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    status VARCHAR(50) DEFAULT 'pending' -- 'pending', 'accepted', 'rejected'
);

CREATE INDEX IF NOT EXISTS idx_interactions_post_id ON interactions(post_id);
CREATE INDEX IF NOT EXISTS idx_interactions_type ON interactions(interaction_type);
CREATE INDEX IF NOT EXISTS idx_interactions_created_at ON interactions(created_at DESC);

-- Reports table for moderation
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    reporter_email VARCHAR(255),
    reason VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    status VARCHAR(50) DEFAULT 'pending' -- 'pending', 'reviewed', 'resolved'
);

CREATE INDEX IF NOT EXISTS idx_reports_post_id ON reports(post_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at DESC);

-- Alerts table for future email notifications
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    location geography(Point, 4326) NOT NULL,
    radius_meters INTEGER NOT NULL DEFAULT 5000,
    categories post_category[],
    keywords TEXT[],
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    last_triggered_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_alerts_location ON alerts USING GIST(location);
CREATE INDEX IF NOT EXISTS idx_alerts_email ON alerts(email);
CREATE INDEX IF NOT EXISTS idx_alerts_active ON alerts(is_active);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';



-- Triggers to automatically update updated_at
CREATE TRIGGER update_buildings_updated_at 
    BEFORE UPDATE ON buildings 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_lost_found_areas_updated_at 
    BEFORE UPDATE ON lost_found_areas 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_posts_updated_at 
    BEFORE UPDATE ON posts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Insert sample buildings for testing
INSERT INTO buildings (name, description, location) VALUES
('Science Building', 'Main science building with labs and classrooms', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326)),
('Library', 'Central library with study spaces', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326)),
('Student Center', 'Student union building with dining and activities', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326)),
('Gym', 'Athletic center with gym and pool', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326));

-- Insert sample lost & found areas
INSERT INTO lost_found_areas (building_id, name, location_description, contact_person, hours_of_operation, pickup_instructions) VALUES
((SELECT id FROM buildings WHERE name = 'Science Building'), 'Main Office', 'First floor, room 101', 'Dr. Smith', 'Mon-Fri 9AM-5PM', 'Check in at reception desk'),
((SELECT id FROM buildings WHERE name = 'Library'), 'Circulation Desk', 'Main floor, near entrance', 'Ms. Johnson', 'Mon-Sun 8AM-10PM', 'Ask at circulation desk'),
((SELECT id FROM buildings WHERE name = 'Student Center'), 'Information Desk', 'Main lobby', 'Student Assistant', 'Mon-Fri 8AM-8PM', 'Check information desk'),
((SELECT id FROM buildings WHERE name = 'Gym'), 'Equipment Room', 'Lower level', 'Coach Wilson', 'Mon-Fri 6AM-10PM', 'Ask at equipment room');

-- Insert sample admin user
INSERT INTO users (sso_id, email, name, role) VALUES
('admin_sso_123', 'admin@college.edu', 'System Administrator', 'admin');

-- Insert some sample data for testing (found items)
INSERT INTO posts (type, category, title, description, location, lost_found_area_id, posted_by, is_lost_item, edit_token, image_urls) VALUES
('found', 'item', 'Lost iPhone 14', 'Found iPhone 14 with blue case near Science Building entrance', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326), (SELECT id FROM lost_found_areas WHERE name = 'Main Office'), (SELECT id FROM users WHERE role = 'admin'), false, 'edit_token_123', ARRAY['/uploads/sample_phone1.jpg']),
('found', 'document', 'Found Student ID Card', 'Found student ID card near Library entrance', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326), (SELECT id FROM lost_found_areas WHERE name = 'Circulation Desk'), (SELECT id FROM users WHERE role = 'admin'), false, 'edit_token_456', ARRAY['/uploads/sample_id1.jpg']),
('found', 'item', 'Found Laptop Charger', 'Found laptop charger in Student Center cafeteria', ST_GeomFromText('POINT(-73.935242 40.730610)', 4326), (SELECT id FROM lost_found_areas WHERE name = 'Information Desk'), (SELECT id FROM users WHERE role = 'admin'), false, 'edit_token_789', ARRAY['/uploads/sample_charger1.jpg']); 