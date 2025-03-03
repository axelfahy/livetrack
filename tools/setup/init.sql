CREATE DATABASE tracking;

\c tracking

-- pilot table
CREATE TABLE IF NOT EXISTS pilot (
   id VARCHAR(100) PRIMARY KEY,
   name VARCHAR(100),
   home VARCHAR(100),
   orgs VARCHAR(100)[],
   tracker_type VARCHAR(100)
);

-- track table
CREATE TABLE IF NOT EXISTS track (
    pilot_id VARCHAR(100),
    unix_time TIMESTAMPTZ,
    latitude REAL,
    longitude REAL,
    altitude INTEGER,
    msg_type VARCHAR(100),
    msg_content VARCHAR(200),
    PRIMARY KEY (pilot_id, unix_time)
);

-- Create a function to send a NOTIFY event with pilot details
CREATE FUNCTION notify_new_track_data() RETURNS TRIGGER AS $$
DECLARE
    pilot_data RECORD;
BEGIN
    -- Fetch pilot details
    SELECT id, name, home, orgs, tracker_type INTO pilot_data
    FROM pilot
    WHERE id = NEW.pilot_id;

    -- Send notification with track point and pilot info
    PERFORM pg_notify('new_track_data', json_build_object(
        'pilot_id', NEW.pilot_id,
        'unix_time', NEW.unix_time,
        'latitude', NEW.latitude,
        'longitude', NEW.longitude,
        'altitude', NEW.altitude,
        'msg_type', NEW.msg_type,
        'msg_content', NEW.msg_content,
        'pilot', json_build_object(
            'id', pilot_data.id,
            'name', pilot_data.name,
            'home', pilot_data.home,
            'orgs', pilot_data.orgs,
            'tracker_type', pilot_data.tracker_type
        )
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger that calls the function on insert
CREATE TRIGGER track_insert_trigger
AFTER INSERT ON track
FOR EACH ROW
EXECUTE FUNCTION notify_new_track_data();

-- insert known pilots to retrieve
INSERT INTO pilot(id, name, home, orgs, tracker_type)
VALUES
  ('id', 'Pilot name', 'home', '{"org1", "org2"}', 'spot');
