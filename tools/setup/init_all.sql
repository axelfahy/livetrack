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
  ('0Xt9612cBl2Qyflho7aO2Pa8bzHzcTugT', 'Alan', 'St-Cergue', '{"alpsfreeride", "followus"}', 'spot'),
  ('0Z7eRKM9rCcrima9ic2qqvNFjDjgf87fG', 'Axel', 'Biel', '{"alpsfreeride", "followus", "axlair"}', 'spot'),
  ('0D3D3Gdn4JqV4hEkp4TRiRoc02Hk5frJa', 'Calina', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0RKUQmnYcUhGflhlrrsm9jthBJo2WjNOq', 'Colin', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0Sqp9zyH3ZOfaWhPi4KeUd2GNfqTW43aG', 'Damien', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0Ejvlgbq6FODAqqQ2q1ANSdvMGeUczp59', 'Franco', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0RIzKreNFXTbzMKu27phCpZYiRCumqgtR', 'Gerald', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0hAiGNOqiCJBrO00pihgqCgi21DnzEays', 'Gilles', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0u6Lf1qaMcGnAshKof1AKJu3N9fpFPbxK', 'Jedi', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0nBdBHPY5ecq1R3DcavOBylPkcE9FzcdY', 'Jose', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0fjDmqApjzhZBSjsUeXOHlmDBSZOfSGzd', 'Martin', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0lYnQKE32G6yP1Q5uSzrdxtNflelETudC', 'Nicolas', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0exLUaY494ItFWhB9GGqgFH7xIUaBJ0AX', 'Patrick', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0zoYne4G1uMCxWlYtmoWVa21oQETsRENa', 'Paul', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('09jZgRjaqWGSO5q9g7z6MYCE9myY3Fl3T', 'Pierre', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0taTqoVWbU53AiJipF0HY1tREDdRN5iuQ', 'PH', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0I7evhLHt03RRZxv3gu1gqibM7aIdbm2i', 'Reynald', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0qm3EuuaEbIKRzUvi2EJzYHn77YAMcLi5', 'Romain', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0XhukSgCEf8qqVlWui3vd1AgC4uqsXWMr', 'Ryan', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0J6GDsRMyPHggJiQZgIMkJ5VqoCNCtu6o', 'Sebastian', 'Zurich', '{"alpsfreeride"}', 'spot'),
  ('serenafly', 'Serena', 'Nyon', '{"alpsfreeride"}', 'garmin'),
  ('rohnipfu', 'Dominic', 'Zurich', '{"alpsfreeride"}', 'garmin'),
  ('ricoforthelost', 'Rico', 'Zurich', '{"alpsfreeride"}', 'garmin'),
  ('Oxotorok', 'Livio', 'Geneva', '{"alpsfreeride"}', 'garmin'),
  ('45TUV', 'Daniel', 'Geneva', '{"alpsfreeride"}', 'garmin'),
  ('0smxuLcDXXlQkR6Uzu2HcDvp7MmW7TCLc', 'Stefanie', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('0DqDbA7W8aYtYeog1KuYxWzxsGY9RDPOk', 'Stephane', 'Geneva', '{"alpsfreeride"}', 'spot'),
  ('06nzc733u3mGUkCqNKM2PZMsqvFmSWQxs', 'Yael', 'Verbier', '{"alpsfreeride"}', 'spot');
