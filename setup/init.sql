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

-- insert known pilots to retrieve
INSERT INTO pilot(id, name, home, orgs, tracker_type)
VALUES
  ('id', 'Pilot name', 'home', '{"org1", "org2"}', 'spot');

