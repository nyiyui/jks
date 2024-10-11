CREATE TABLE tasks(
  id INTEGER PRIMARY KEY,
  description TEXT,
  quick_title TEXT
);

CREATE TABLE activity_log(
  id INTEGER PRIMARY KEY,
  task_id INTEGER,
  location TEXT,
  time_start DATETIME, -- in Unix time
  time_end DATETIME, -- in Unix time
  FOREIGN KEY(id) REFERENCES tasks(id)
);
